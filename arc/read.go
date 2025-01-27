package arc

import (
	"archiver/arc/header"
	"archiver/errtype"
	"archiver/filesystem"
	"fmt"
	"io"
	"log"
	"os"
	fp "path/filepath"
	"sort"
)

// Проверяет чем является path, директорией,
// символьной ссылкой или файлом, возвращает
// интерфейс заголовка, указывающий на
// соответствующий тип
func fetchPath(path string) (h header.Header, err error) {
	if len(path) > 1023 {
		return nil, ErrLongPath(path)
	}

	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	atime, mtime := amTimes(info)

	b, err := header.NewBase(fp.ToSlash(path), atime, mtime)
	if err != nil {
		return nil, err
	}

	if info.Mode()&os.ModeSymlink != 0 {
		if target, err := fp.EvalSymlinks(path); err != nil {
			return nil, err
		} else if target, err = fp.Abs(target); err != nil {
			return nil, err
		} else {
			h = header.NewSymItem(path, target)
		}
	} else if info.Mode()&os.ModeDir != 0 {
		h = header.NewDirItem(b.PathInArc())
	} else {
		h = header.NewFileItem(b, header.Size(info.Size()))
	}

	return h, nil
}

// Рекурсивно собирает элементы в директории
func fetchDir(path string) (headers []header.Header, err error) {
	err = fp.WalkDir(path, func(path string, _ os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		header, err := fetchPath(path)
		if err != nil {
			if err == ErrLongPath(path) {
				fmt.Println(err)
				return nil
			} else {
				return err
			}
		}

		if header != nil {
			headers = append(headers, header)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return headers, nil
}

// Собирает элементы файловой системы в заголовки
func (Arc) fetchHeaders(paths []string) (headers []header.Header, err error) {
	var (
		dirHeaders []header.Header
		header     header.Header
	)

	for _, path := range paths { // Получение списка файлов и директории
		// Добавление директории в заголовок
		// и ее рекурсивный обход
		if filesystem.DirExists(path) {
			if dirHeaders, err = fetchDir(path); err == nil {
				headers = append(headers, dirHeaders...)
			} else {
				return nil, errtype.Join(ErrFetchDirs, err)
			}
			continue
		}

		if header, err = fetchPath(path); err != nil { // Добавалние файла в заголовок
			return nil, errtype.Join(ErrFetchDirs, err)
		} else if header != nil {
			headers = append(headers, header)
		}
	}
	return headers, nil
}

// Читает заголовки из архива, определяет смещение данных
func (arc *Arc) readHeaders(arcFile io.ReadSeekCloser) (headers []header.Header, err error) {
	var (
		sym  *header.SymItem
		file *header.FileItem
		h    header.Header
		typ  header.HeaderType
	)

	// Пропускаем магическое число и тип компрессора
	arcFile.Seek(arcHeaderLen, io.SeekStart)

	for err != io.EOF {
		err = filesystem.BinaryRead(arcFile, &typ)
		if err != io.EOF && err != nil {
			return nil, errtype.Join(ErrReadHeaderType, err)
		} else if err == io.EOF {
			continue
		}

		switch typ {
		case header.File:
			if file, err = arc.readFileHeader(arcFile); err != nil && err != io.EOF {
				return nil, errtype.Join(ErrReadHeaders, err)
			}
			if file != nil {
				h = file
			}
		case header.Symlink:
			if sym, err = arc.readSymHeader(arcFile); err != nil && err != io.EOF {
				return nil, errtype.Join(ErrReadHeaders, err)
			}
			if sym != nil {
				h = sym
			}
		default:
			return nil, ErrHeaderType
		}

		if h != nil {
			headers = append(headers, h)
		}
		h = nil
	}

	// Возврат каретки в начало первого заголовка
	arcFile.Seek(arcHeaderLen, io.SeekStart)

	dirs := arc.insertDirs(headers)
	headers = append(headers, dirs...)
	sort.Sort(header.ByPathInArc(headers))

	return headers, nil
}

// Читает и возвращает заголовки файлов
func (arc Arc) readFileHeader(arcFile io.ReadSeeker) (*header.FileItem, error) {
	var (
		file     = &header.FileItem{}
		dataSize header.Size
		err      error
		crc      uint32
	)

	pos, _ := arcFile.Seek(0, io.SeekCurrent)
	log.Println("Читаю заголовок файла с позиции:", pos)
	if err = file.Read(arcFile); err != nil && err != io.EOF {
		return nil, errtype.Join(ErrReadFileHeader, err)
	}

	pos, _ = arcFile.Seek(0, io.SeekCurrent)
	log.Println("Читаю размер сжатых данных с позиции:", pos)
	if dataSize, err = arc.skipFileData(arcFile, false); err == io.EOF {
		return nil, err
	} else if err != nil {
		return nil, errtype.Join(ErrSkipData, err)
	}
	file.SetCSize(dataSize)

	if err = filesystem.BinaryRead(arcFile, &crc); err != nil {
		return nil, errtype.Join(ErrReadCRC, err)
	}
	file.SetCRC(crc)

	return file, nil
}

// Читает заголовок символьной ссылки из архива
func (arc *Arc) readSymHeader(arcFile io.ReadSeeker) (sym *header.SymItem, err error) {
	sym = &header.SymItem{}
	pos, _ := arcFile.Seek(0, io.SeekCurrent)
	log.Println("Читаю заголовок символьной ссылки с позиции:", pos)
	if err = sym.Read(arcFile); err == io.EOF {
		return nil, err
	} else if err != nil {
		return nil, errtype.Join(ErrReadSymHeader, err)
	}

	return sym, nil
}

// Вставляет в срез с заголовками пути к директориям
func (Arc) insertDirs(headers []header.Header) []header.Header {
	var (
		dirs  []header.Header
		seen  = map[string]struct{}{}
		parts []string
		path  string
	)

	for _, h := range headers {
		if _, ok := h.(*header.SymItem); ok {
			continue // Пропускаем символьные ссылки
		}

		path = fp.Dir(h.PathInArc())
		if _, ok := seen[path]; ok {
			continue // Такой путь уже есть
		}

		parts = filesystem.SplitPath(path)
		path = ""
		for _, p := range parts {
			path = fp.Join(path, p)
			if _, ok := seen[path]; !ok {
				seen[path] = struct{}{} // Такого пути нет
				dirs = append(dirs, header.NewDirItem(path))
			}
		}
	}

	return dirs
}

// Пропускает файл в дескрипторе файла архива
func (arc Arc) skipFileData(arcFile io.ReadSeeker, skipCRC bool) (read header.Size, err error) {
	var bufferSize int64

	for {
		if err = filesystem.BinaryRead(arcFile, &bufferSize); err == io.EOF {
			return 0, err
		} else if err != nil {
			return 0, errtype.Join(ErrReadCompLen, err)
		}

		if bufferSize == -1 {
			break
		} else if arc.checkBufferSize(bufferSize) {
			return 0, errtype.Join(ErrBufSize(bufferSize), err)
		}

		read += header.Size(bufferSize)

		if _, err = arcFile.Seek(bufferSize, io.SeekCurrent); err != nil {
			return 0, errtype.Join(ErrSkipData, err)
		}
	}

	if skipCRC {
		if _, err = arcFile.Seek(4, io.SeekCurrent); err != nil {
			return 0, errtype.Join(ErrSkipCRC, err)
		}
	}

	return read, nil
}
