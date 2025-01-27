package arc

import (
	"archiver/arc/header"
	"archiver/errtype"
	"fmt"
	"os"
	"sort"
)

// Печатает информацию об архиве
func (arc Arc) ViewStat() error {
	arcFile, err := os.OpenFile(arc.arcPath, os.O_RDONLY, 0644)
	if err != nil {
		return errtype.ErrRuntime(
			errtype.Join(ErrOpenArc, err),
		)
	}

	headers, err := arc.readHeaders(arcFile)
	if err != nil {
		return errtype.ErrRuntime(
			errtype.Join(ErrReadHeaders, err),
		)
	}
	sort.Sort(header.ByPathInArc(headers))

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
	arcFile, err := os.OpenFile(arc.arcPath, os.O_RDONLY, 0644)
	if err != nil {
		return errtype.ErrIntegrity(
			errtype.Join(ErrOpenArc, err),
		)
	}

	headers, err := arc.readHeaders(arcFile)
	if err != nil {
		return errtype.ErrRuntime(
			errtype.Join(ErrReadHeaders, err),
		)
	}
	sort.Sort(header.ByPathInArc(headers))

	for _, h := range headers {
		if si, ok := h.(*header.SymItem); ok {
			fmt.Println(si.PathInArc(), "->", si.PathOnDisk())
		} else {
			fmt.Println(h.PathOnDisk())
		}
	}

	return nil
}
