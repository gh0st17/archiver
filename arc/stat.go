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
	headers = arc.sortHeaders(headers)

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
	headers = arc.sortHeaders(headers)

	for _, h := range headers {
		if si, ok := h.(*header.SymItem); ok {
			fmt.Println(si.PathInArc(), "->", si.PathOnDisk())
		} else {
			fmt.Println(h.PathOnDisk())
		}
	}

	return nil
}

func (Arc) sortHeaders(headers []header.Header) []header.Header {
	paths := make([]header.PathProvider, len(headers))
	for i := range paths {
		paths[i] = headers[i].(header.PathProvider)
	}
	sort.Sort(header.ByPathInArc(paths))

	for i := range headers {
		headers[i] = paths[i].(header.Header)
	}

	return headers
}
