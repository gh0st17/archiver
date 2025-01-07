package arc

import (
	"archiver/arc/header"
	"fmt"
	"sort"
)

// Печатает информации о сжатии архива
func (arc Arc) ViewStat() error {
	// 	Читаем элементы
	headers, err := arc.readHeaders()
	if err != nil {
		return err
	}
	sort.Sort(header.ByPath(headers))

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
func (arc Arc) ViewList() error {
	// 	Читаем элементы
	headers, err := arc.readHeaders()
	if err != nil {
		return err
	}
	sort.Sort(header.ByPath(headers))

	for _, h := range headers {
		fmt.Println(h.Path())
	}

	return nil
}
