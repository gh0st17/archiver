package arc

import (
	"archiver/arc/header"
	c "archiver/compressor"
	"archiver/filesystem"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// Записывает заголовки в файл архива
func (arc Arc) writeItems(headers []header.Header) error {
	// Создаем файл
	arcFile, err := os.Create(arc.ArchivePath)
	if err != nil {
		return err
	}
	defer arcFile.Close()

	// Пишем магическое число
	err = binary.Write(arcFile, binary.LittleEndian, magicNumber)
	if err != nil {
		return err
	}

	// Пишем тип компрессора
	err = binary.Write(arcFile, binary.LittleEndian, arc.CompType)
	if err != nil {
		return err
	}

	// Пишем размер блока
	err = binary.Write(arcFile, binary.LittleEndian, arc.maxCompLen)
	if err != nil {
		return err
	}

	// Пишем количество заголовков
	err = binary.Write(arcFile, binary.LittleEndian, int64(len(headers)))
	if err != nil {
		return err
	}

	// Пишем заголовки
	for _, h := range headers {
		if fi, ok := h.(*header.FileItem); ok {
			fi.SetPath(filesystem.CleanPath(fi.Path()))
		} else if di, ok := h.(*header.DirItem); ok {
			di.SetPath(filesystem.CleanPath(di.Path()))
		}
		h.Write(arcFile)
	}

	// Пишем сжатые данные
	tmpFile, err := os.OpenFile(tmpPath, os.O_RDONLY, 0644)
	if err != nil {
		return fmt.Errorf("writeItems: %v", err)
	}
	defer tmpFile.Close()

	buffer := make([]byte, c.BufferSize)

	for {
		bytesRead, err := tmpFile.Read(buffer)
		if bytesRead > 0 {
			_, err := arcFile.Write(buffer[:bytesRead])
			if err != nil {
				return err
			}
		}

		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
	}

	return nil
}
