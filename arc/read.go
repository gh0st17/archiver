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
)

// Проверяет чем является path, директорией или файлом,
// возвращает интерфейс заголовка, указывающий на
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
			h = header.NewSymDirItem(path, target)
		}
	} else if info.Mode()&os.ModeDir != 0 {
		h = header.NewDirItem(b)
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

		headers = append(headers, header)
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

		if header, err = fetchPath(path); err == nil { // Добавалние файла в заголовок
			headers = append(headers, header)
		} else {
			return nil, errtype.Join(ErrFetchDirs, err)
		}
	}
	return headers, nil
}

// Читает заголовки из архива, определяет смещение данных
func (arc *Arc) readHeaders() (headers []header.Header, arcFile io.ReadSeekCloser, err error) {
	var (
		dirsSyms []header.PathProvider
		files    []header.FileItem
	)

	arcFile, err = os.OpenFile(arc.arcPath, os.O_RDONLY, 0644)
	if err != nil {
		return nil, nil, errtype.Join(ErrOpenArc, err)
	}

	arcFile.Seek(3, io.SeekCurrent) // Пропускаем магическое число и тип компрессора

	if dirsSyms, err = arc.readDirsSyms(arcFile); err != nil {
		return nil, nil, errtype.Join(ErrReadDirsSyms, err)
	}

	dataPos, _ := arcFile.Seek(0, io.SeekCurrent)
	if files, err = arc.readFileHeaders(arcFile); err != nil {
		return nil, nil, errtype.Join(ErrReadFileHeader, err)
	}
	arcFile.Seek(dataPos, io.SeekStart)

	for i := range dirsSyms {
		headers = append(headers, dirsSyms[i].(header.Header))
	}
	for _, file := range files {
		headers = append(headers, &file)
	}

	return headers, arcFile, nil
}

// Читает заголовки директории из архива
func (arc *Arc) readDirsSyms(arcFile io.Reader) (paths []header.PathProvider, err error) {
	var reader header.Reader
	var headersCount int64 // Количество заголовков директории
	if err = filesystem.BinaryRead(arcFile, &headersCount); err != nil {
		return nil, errtype.Join(ErrReadHeadersCount, err)
	}
	paths = make([]header.PathProvider, headersCount)

	// Читаем заголовки директории
	for i := int64(0); i < headersCount; i++ {
		var typ byte
		if err = filesystem.BinaryRead(arcFile, &typ); err != nil {
			return nil, errtype.Join(ErrReadHeaderType, err)
		}

		switch typ {
		case 0:
			reader = &header.DirItem{}
		case 1:
			reader = &header.SymItem{}
		default:
			return nil, errtype.Join(ErrHeaderType, err)
		}

		if err = reader.Read(arcFile); err != nil {
			return nil, err
		}
		paths[i] = reader.(header.PathProvider)
	}

	return paths, nil
}

// Читает и возвращает заголовки файлов
func (arc Arc) readFileHeaders(arcFile io.ReadSeeker) ([]header.FileItem, error) {
	var (
		files    []header.FileItem
		dataSize header.Size
		err      error
		crc      uint32
	)

	for { // One cycle is one file
		var fi header.FileItem

		if err = fi.Read(arcFile); err != nil {
			if err == io.EOF {
				break
			}
			return nil, errtype.Join(ErrReadFileHeader, err)
		}

		pos, _ := arcFile.Seek(0, io.SeekCurrent)
		log.Println("Читаю размер сжатых данных с позиции:", pos)
		if dataSize, err = arc.skipFileData(arcFile, false); err != nil {
			return nil, errtype.Join(ErrReadCompLen, err)
		}
		fi.SetCSize(dataSize)

		if err = filesystem.BinaryRead(arcFile, &crc); err != nil {
			return nil, errtype.Join(ErrReadCRC, err)
		}
		fi.SetCRC(crc)

		files = append(files, fi)
	}

	return files, nil
}

// Пропускает файл в дескрипторе файла архива
func (arc Arc) skipFileData(arcFile io.ReadSeeker, skipCRC bool) (read header.Size, err error) {
	var bufferSize int64

	for {
		if err = filesystem.BinaryRead(arcFile, &bufferSize); err != nil {
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
