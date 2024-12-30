package arc

import (
	"archiver/arc/header"
	"fmt"
)

// Печатает информации о сжатии архива
func ViewStat(arc *Arc) error {
	// 	Читаем элементы
	headers, err := readHeaders(arc)
	if err != nil {
		return err
	}

	fmt.Printf("Тип компрессора: %s\n", arc.CompType)
	header.PrintStatHeader()

	var original, compressed header.Size
	// Выводим элементы
	for _, h := range headers {
		fmt.Println(h)

		if fi, ok := h.(*header.FileItem); ok {
			original += fi.UncompressedSize
			compressed += fi.CompressedSize
		}
	}
	header.PrintSummary(compressed, original)

	return nil
}

// Печатает список файлов в архиве
func ViewList(arc *Arc) error {
	// 	Читаем элементы
	headers, err := readHeaders(arc)
	if err != nil {
		return err
	}

	for _, h := range headers {
		fmt.Println(h.Path())
	}

	return nil
}
