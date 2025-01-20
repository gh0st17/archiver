package arc

import (
	"archiver/arc/header"
	"archiver/errtype"
	"fmt"
	"slices"
	"strings"
)

// Печатает информации об архиве
func (arc Arc) ViewStat() error {
	headers, _, err := arc.readHeaders()
	if err != nil {
		return errtype.ErrRuntime(ErrReadHeaders, err)
	}
	arc.sortHeaders(headers)

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
	arc.sortHeaders(headers)

	for _, h := range headers {
		if si, ok := h.(*header.SymItem); ok {
			fmt.Println(si.PathInArc(), "->", si.PathOnDisk())
		} else {
			fmt.Println(h.PathOnDisk())
		}
	}

	return nil
}

func (Arc) sortHeaders(headers []header.Header) {
	slices.SortFunc(headers, func(a, b header.Header) int {
		return strings.Compare(
			strings.ToLower(a.PathInArc()),
			strings.ToLower(b.PathInArc()),
		)
	})
}
