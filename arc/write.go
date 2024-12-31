package arc

import (
	"archiver/arc/header"
	"encoding/binary"
	"os"
)

// Записывает заголовки в файл архива
func writeItems(arc *Arc, headers []header.Header) error {
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
	for len(headers) > 0 {
		if fi, ok := headers[0].(*header.FileItem); ok {
			if _, err := f.Write(fi.Data); err != nil {
				return err
			}
		}

		headers = headers[1:]
	}

	return nil
}

// CopyBlocks копирует данные из r в w блоками заданного размера.
// func CopyBlocks(r io.Reader, w io.Writer, blockSize int) (int, error) {
// 	if blockSize <= 0 {
// 		return 0, io.ErrShortBuffer // Ошибка, если размер блока некорректен
// 	}

// 	buffer := make([]byte, blockSize)
// 	sumRead := 0

// 	for {
// 		// Читаем данные из r в буфер
// 		n, err := r.Read(buffer)
// 		if err != nil && err != io.EOF {
// 			return 0, err // Возвращаем ошибку, если чтение не удалось
// 		}
// 		sumRead += n

// 		// Пишем данные из буфера в w
// 		if n > 0 {
// 			if _, writeErr := w.Write(buffer[:n]); writeErr != nil {
// 				return 0, writeErr // Возвращаем ошибку записи
// 			}
// 		}

// 		// Если достигнут конец данных, выходим из цикла
// 		if err == io.EOF {
// 			break
// 		}
// 	}

// 	return sumRead, nil
// }
