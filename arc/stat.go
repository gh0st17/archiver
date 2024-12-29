package arc

import (
	"archiver/arc/header"
	"bufio"
	"fmt"
	"os"
)

// Печатает информации о сжатии архива
func ViewStat(arcParams *Arc) error {
	f, err := os.Open(arcParams.ArchivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	r := bufio.NewReader(f)
	// 	Читаем элементы
	headers, err := readHeaders(r)
	if err != nil {
		return err
	}

	fmt.Printf("Тип компрессора: %s\n", arcParams.CompType)
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
func ViewList(arcParams *Arc) error {
	f, err := os.Open(arcParams.ArchivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	r := bufio.NewReader(f)
	// 	Читаем элементы
	headers, err := readHeaders(r)
	if err != nil {
		return err
	}

	for _, h := range headers {
		fmt.Println(h.Path())
	}

	return nil
}
