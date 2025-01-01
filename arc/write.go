package arc

import (
	"archiver/arc/header"
	c "archiver/compressor"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// Записывает заголовки в файл архива
func (arc Arc) writeItems(headers []header.Header) error {
	// Создаем файл
	f, err := os.Create(arc.ArchivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	// Пишем магическое число
	err = binary.Write(f, binary.LittleEndian, magicNumber)
	if err != nil {
		return err
	}

	// Пишем тип компрессора
	err = binary.Write(f, binary.LittleEndian, arc.CompType)
	if err != nil {
		return err
	}

	// Пишем количество заголовков
	err = binary.Write(f, binary.LittleEndian, int64(len(headers)))
	if err != nil {
		return err
	}

	// Пишем заголовки
	for _, h := range headers {
		h.Write(f)
	}

	// Пишем данные
	// Создаем или открываем целевой файл для записи
	tmpFile, err := os.OpenFile(tmpPath, os.O_RDONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("arc: writeItems: %v", err)
	}
	defer tmpFile.Close()

	// Создаем буфер для чтения данных
	buffer := make([]byte, c.GetBufferSize())

	for {
		// Читаем данные из исходного файла в буфер
		bytesRead, readErr := tmpFile.Read(buffer)
		if bytesRead > 0 {
			// Пишем данные из буфера в целевой файл
			_, writeErr := f.Write(buffer[:bytesRead])
			if writeErr != nil {
				return writeErr
			}
		}

		// Проверяем, достигнут ли конец файла или возникла ошибка
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return readErr
		}
	}

	return nil
}
