package decompress

import (
	"io"
	"log"
	fp "path/filepath"
	"sort"

	"github.com/gh0st17/archiver/arc/internal/generic"
	"github.com/gh0st17/archiver/arc/internal/header"
	"github.com/gh0st17/archiver/errtype"
	"github.com/gh0st17/archiver/filesystem"
)

// Читает заголовки из архива, определяет смещение данных
func ReadHeaders(arcFile io.ReadSeeker, arcLenH int64) ([]header.Header, error) {
	var headers []header.Header

	handler := func(typ header.HeaderType, arcFile io.ReadSeeker) (err error) {
		var h header.Header
		switch typ {
		case header.File:
			h, err = readFileHeader(arcFile)
		case header.Symlink:
			h, err = readSymHeader(arcFile)
		default:
			return ErrHeaderType
		}

		if err != nil && err != io.EOF {
			return errtype.Join(ErrReadHeaders, err)
		}
		if h != nil {
			headers = append(headers, h)
		}
		return nil
	}

	// Сохраняем позицию каретки
	pos, err := arcFile.Seek(0, io.SeekCurrent)
	if err = generic.ProcessHeaders(arcFile, handler); err != nil {
		return nil, errtype.Join(ErrReadHeaderType, err)
	}
	// Восстанавливаем позицию каретки
	arcFile.Seek(pos, io.SeekStart)

	dirs := insertDirs(headers)
	headers = append(headers, dirs...)
	sort.Sort(header.ByPathInArc(headers))

	return headers, nil
}

// Читает заголовок файла из arcFile и возвращает его
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

// Читает заголовок символьной ссылки из arcFile и возвращает его
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

// Просматривает срез с заголовками headers, выделяет
// из них уникальные пути к директориям и возвращает их
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
				dirs = append(dirs, header.NewDirItem(fp.ToSlash(path)))
			}
		}
	}

	return dirs
}

// Пропускает файл в читателе файла архива
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
			return 0, errtype.Join(ErrSeek, err)
		}
	}

	if skipCRC {
		if _, err = arcFile.Seek(4, io.SeekCurrent); err != nil {
			return 0, errtype.Join(ErrSeek, err)
		}
	}

	return read, nil
}
