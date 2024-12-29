package arc

import (
	"archiver/arc/header"
	"encoding/binary"
	"os"
)

// Записывает заголовки в файл архива
func writeItems(arcParams *Arc, headers []header.Header) error {
	// Создаем файл
	f, err := os.Create(arcParams.ArchivePath)
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
	err = binary.Write(f, binary.LittleEndian, arcParams.CompType)
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
		// Пишем тип заголовка
		err = binary.Write(f, binary.LittleEndian, h.Type())
		if err != nil {
			return err
		}

		h.Write(f)
	}

	// Пишем данные
	for len(headers) > 0 {
		if headers[0].Type() == header.File {
			if _, err := f.Write(headers[0].(*header.FileItem).Bytes()); err != nil {
				return err
			}
		}

		headers = headers[1:]
	}

	return nil
}
