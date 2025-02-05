package arc

import (
	"io"
	"os"

	"github.com/gh0st17/archiver/arc/internal/decompress"
	"github.com/gh0st17/archiver/arc/internal/generic"
	"github.com/gh0st17/archiver/arc/internal/header"
	"github.com/gh0st17/archiver/errtype"
)

// Выполняет распаковку архива.
//
// Открывает файл архива, пропускает магическое число и тип
// компрессора, затем обрабатывает содержимое архива, проходя
// по заголовкам разного типа. Обнаруженные заголовки
// обрабатываются соответствующими методами, а после завершения
// работы освобождаются декомпрессоры.
func (arc Arc) Decompress() error {
	arcFile, err := os.OpenFile(arc.path, os.O_RDONLY, 0644)
	if err != nil {
		return errtype.ErrDecompress(errtype.Join(ErrOpenArc, err))
	}
	defer arcFile.Close()

	if err := generic.LoadDict(arc.RestoreParams); err != nil {
		return errtype.ErrDecompress(err)
	}

	// Пропускаем магическое число и тип компрессора
	if _, err = arcFile.Seek(headerLen, io.SeekStart); err != nil {
		return errtype.ErrDecompress(errtype.Join(ErrSeek, err))
	}

	if err := generic.ProcessHeaders(arcFile, arc.restoreHandler); err != nil {
		return errtype.ErrDecompress(err)
	}

	// Сброс декомпрессоров перед новым вызовом этой функции
	generic.ResetDecomp()

	return nil
}

// Обработчик заголовков архива для распаковки
func (arc Arc) restoreHandler(typ header.HeaderType, arcFile io.ReadSeeker) (err error) {
	switch typ {
	case header.File:
		err = decompress.RestoreFile(arcFile, arc.RestoreParams, arc.verbose)
	case header.Symlink:
		err = decompress.RestoreSym(arcFile, arc.RestoreParams, arc.verbose)
	default:
		return ErrHeaderType
	}
	if err != nil && err != io.EOF {
		return err
	}

	return nil
}
