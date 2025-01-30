package decompress

import (
	"archiver/arc/internal/generic"
	"archiver/arc/internal/header"
	"archiver/errtype"
	"archiver/filesystem"
	"io"
	"log"
	fp "path/filepath"
	"sort"
)

// Читает заголовки из архива, определяет смещение данных
func ReadHeaders(arcFile io.ReadSeekCloser, arcLenH int64) (headers []header.Header, err error) {
	var (
		sym  *header.SymItem
		file *header.FileItem
		h    header.Header
		typ  header.HeaderType
	)

	// Пропускаем магическое число и тип компрессора
	arcFile.Seek(arcLenH, io.SeekStart)

	for err != io.EOF {
		err = filesystem.BinaryRead(arcFile, &typ)
		if err != io.EOF && err != nil {
			return nil, errtype.Join(ErrReadHeaderType, err)
		} else if err == io.EOF {
			continue
		}

		switch typ {
		case header.File:
			if file, err = readFileHeader(arcFile); err != nil && err != io.EOF {
				return nil, errtype.Join(ErrReadHeaders, err)
			}
			if file != nil {
				h = file
			}
		case header.Symlink:
			if sym, err = readSymHeader(arcFile); err != nil && err != io.EOF {
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
	arcFile.Seek(arcLenH, io.SeekStart)

	dirs := insertDirs(headers)
	headers = append(headers, dirs...)
	sort.Sort(header.ByPathInArc(headers))

	return headers, nil
}

// Читает и возвращает заголовки файлов
func readFileHeader(arcFile io.ReadSeeker) (*header.FileItem, error) {
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
	if dataSize, err = skipFileData(arcFile, false); err == io.EOF {
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
func readSymHeader(arcFile io.ReadSeeker) (sym *header.SymItem, err error) {
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
func insertDirs(headers []header.Header) []header.Header {
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
func skipFileData(arcFile io.ReadSeeker, skipCRC bool) (read header.Size, err error) {
	var bufferSize int64

	for {
		if err = filesystem.BinaryRead(arcFile, &bufferSize); err == io.EOF {
			return 0, err
		} else if err != nil {
			return 0, errtype.Join(ErrReadCompLen, err)
		}

		if bufferSize == -1 {
			break
		} else if generic.CheckBufferSize(bufferSize) {
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
