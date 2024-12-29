package arc

import (
	"archiver/arc/header"
	"bufio"
	"encoding/binary"
	"os"
	fp "path/filepath"
)

// Собирает элементы из списка файлов
func fetchFile(filepath string, info os.FileInfo) (h header.Header, err error) {
	atime, mtime := AMtimes(info)

	b := header.Base{
		Filepath: fp.ToSlash(filepath),
		AccTime:  atime,
		ModTime:  mtime,
	}

	if info.IsDir() {
		h = &header.DirItem{Base: b}
	} else {
		h = &header.FileItem{
			Base:             b,
			UncompressedSize: header.Size(info.Size()),
		}
	}

	return h, nil
}

// Рекурсивно собирает элементы в директории
func fetchDir(path string) ([]header.Header, error) {
	var headers []header.Header
	err := fp.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := fetchFile(path, info)
		if err != nil {
			return err
		}

		headers = append(headers, header)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return headers, nil
}

// Читает заголовки из архива
func readHeaders(r *bufio.Reader) ([]header.Header, error) {
	var (
		err         error
		headerCount int64
		htype       header.HeaderType
	)

	r.Discard(3) // Пропускаем магическое число и тип компрессора

	// Читаем количество элементов
	if err = binary.Read(r, binary.LittleEndian, &headerCount); err != nil {
		return nil, err
	}
	headers := make([]header.Header, headerCount)

	// Читаем заголовки
	for i := int64(0); i < headerCount; i++ {
		// Читаем тип заголовка
		if err = binary.Read(r, binary.LittleEndian, &htype); err != nil {
			return nil, err
		}

		switch htype {
		case header.File:
			headers[i] = &header.FileItem{}
		case header.Directory:
			headers[i] = &header.DirItem{}
		}

		headers[i].Read(r)
	}

	return headers, nil
}
