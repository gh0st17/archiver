package arc

import (
	"archiver/arc/header"
	"encoding/binary"
	"io"
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

// Читает заголовки из архива, определяет смещение данных
func readHeaders(arc *Arc) ([]header.Header, error) {
	var (
		headerCount int64
		htype       header.HeaderType
	)

	f, err := os.Open(arc.ArchivePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	f.Seek(3, io.SeekCurrent) // Пропускаем магическое число и тип компрессора

	// Читаем количество элементов
	if err = binary.Read(f, binary.LittleEndian, &headerCount); err != nil {
		return nil, err
	}
	headers := make([]header.Header, headerCount)

	// Читаем заголовки
	for i := int64(0); i < headerCount; i++ {
		// Читаем тип заголовка
		if err = binary.Read(f, binary.LittleEndian, &htype); err != nil {
			return nil, err
		}

		switch htype {
		case header.File:
			headers[i] = &header.FileItem{}
		case header.Directory:
			headers[i] = &header.DirItem{}
		}

		headers[i].Read(f)
	}

	arc.DataOffset, err = f.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, err
	}

	return headers, nil
}
