package arc

import (
	"archiver/arc/internal/decompress"
	"archiver/arc/internal/generic"
	"archiver/arc/internal/header"
	"archiver/errtype"
	"archiver/filesystem"
	"io"
	"os"
)

// Выполняет распаковку архива.
//
// Открывает файл архива, пропускает магическое число и тип
// компрессора, затем обрабатывает содержимое архива, проходя
// по заголовкам разного типа. Обнаруженные заголовки
// обрабатываются соответствующими методами, а после завершения
// работы освобождаются декомпрессоры.
func (arc Arc) Decompress() error {
	arcFile, err := os.OpenFile(arc.arcPath, os.O_RDONLY, 0644)
	if err != nil {
		return errtype.ErrDecompress(
			errtype.Join(ErrOpenArc, err),
		)
	}

	// Пропускаем магическое число и тип компрессора
	arcFile.Seek(arcHeaderLen, io.SeekStart)

	// Установка размера буфера записи
	generic.SetWriteBufSize(generic.BufferSize() * generic.Ncpu())

	var typ header.HeaderType
	for err != io.EOF {
		err = filesystem.BinaryRead(arcFile, &typ) // Читаем тип заголовка
		if err != io.EOF && err != nil {
			return errtype.ErrDecompress(
				errtype.Join(ErrReadHeaderType, err),
			)
		} else if err == io.EOF {
			continue
		}

		switch typ {
		case header.File:
			err = decompress.RestoreFile(arcFile, arc.RestoreParams)
			if err != nil && err != io.EOF {
				return errtype.ErrDecompress(
					errtype.Join(ErrDecompressFile, err),
				)
			}
		case header.Symlink:
			err = decompress.RestoreSym(arcFile, arc.RestoreParams)
			if err != nil && err != io.EOF {
				return errtype.ErrDecompress(
					errtype.Join(ErrDecompressSym, err),
				)
			}
		default:
			return errtype.ErrDecompress(ErrHeaderType)
		}
	}

	// Сброс декомпрессоров перед новым
	// использованием этой функции
	generic.ResetDecomp()

	return nil
}
