package arc

import (
	"io"
	"sort"

	"github.com/gh0st17/archiver/arc/internal/compress"
	"github.com/gh0st17/archiver/arc/internal/generic"
	"github.com/gh0st17/archiver/arc/internal/header"
	"github.com/gh0st17/archiver/errtype"
)

// Создает файл архива с содержимым путей paths
func (arc Arc) Compress(paths []string) error {
	var (
		headers []header.Header
		arcFile io.WriteCloser
		err     error
	)

	arcFile, err = arc.writeArcHeader() // Пишем заголовок архива
	if err != nil {
		return errtype.ErrCompress(
			errtype.Join(ErrWriteArcHeaders, err),
		)
	}

	if headers, err = compress.PrepareHeaders(paths); err != nil {
		return errtype.ErrCompress(err)
	}
	sort.Sort(header.ByPathInArc(headers)) // Сортруем без учета регистра

	if err = generic.InitCompressors(arc.RestoreParams); err != nil {
		return errtype.ErrCompress(
			errtype.Join(ErrCompressorInit, err),
		)
	}

	if err = compress.ProcessingHeaders(arcFile, headers, arc.verbose); err != nil {
		arc.closeRemove(arcFile)
		return errtype.ErrCompress(err)
	}

	if err = arcFile.Close(); err != nil {
		return errtype.ErrCompress(
			errtype.Join(ErrCloseFile, err),
		)
	}

	return nil
}
