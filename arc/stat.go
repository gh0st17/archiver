package arc

import (
	"archiver/arc/header"
	"archiver/errtype"
	"fmt"
	"sort"
)

// Печатает информации об архиве
func (arc Arc) ViewStat() error {
	headers, _, err := arc.readHeaders()
	if err != nil {
		return errtype.ErrRuntime(ErrReadHeaders, err)
	}
	sort.Sort(header.ByPath(headers))

	fmt.Printf("Тип компрессора: %s\n", arc.ct)
	header.PrintStatHeader()

	var original, compressed header.Size
	for _, h := range headers {
		fmt.Println(h)

		if fi, ok := h.(*header.FileItem); ok {
			original += fi.UcSize()
			compressed += fi.CSize()
		}
	}
	header.PrintSummary(compressed, original)

	return nil
}

// Печатает список файлов в архиве
func (arc Arc) ViewList() error {
	headers, _, err := arc.readHeaders()
	if err != nil {
		return errtype.ErrRuntime(ErrReadHeaders, err)
	}
	sort.Sort(header.ByPath(headers))

	for _, h := range headers {
		fmt.Println(h.Path())
	}

	return nil
}
