package arc

import (
	"archiver/arc/internal/compress"
	"archiver/arc/internal/generic"
	"archiver/arc/internal/header"
	"archiver/errtype"
	"io"
	"sort"
)

// Создает файл архива с содержимым путей path
func (arc Arc) Compress(paths []string) error {
	var (
		headers []header.Header
		arcFile io.WriteCloser
		err     error
	)

	if headers, err = compress.PrepareHeaders(paths); err != nil {
		return errtype.ErrCompress(err)
	}
	sort.Sort(header.ByPathInArc(headers)) // Сортруем без учета регистра

	arcFile, err = arc.writeArcHeader() // Пишем заголовок архива
	if err != nil {
		return errtype.ErrCompress(
			errtype.Join(ErrWriteArcHeaders, err),
		)
	}

	if err = generic.InitCompressors(arc.ct, arc.cl); err != nil {
		return errtype.ErrCompress(
			errtype.Join(ErrCompressorInit, err),
		)
	}

	// Установка размера буфера записи
	generic.SetWriteBufSize((generic.BufferSize() * generic.Ncpu()) << 1)

	if err = compress.ProcessingHeaders(arcFile, headers); err != nil {
		arc.closeRemove(arcFile)
		return errtype.ErrCompress(err)
	}
	arcFile.Close()

	return nil
}
