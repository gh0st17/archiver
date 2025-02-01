package arc

import (
	"archiver/arc/internal/decompress"
	"archiver/arc/internal/generic"
	"archiver/arc/internal/header"
	"archiver/errtype"
	"fmt"
	"io"
	"os"
)

// Проверяет целостность данных в архиве
func (arc Arc) IntegrityTest() error {
	arcFile, err := os.OpenFile(arc.path, os.O_RDONLY, 0644)
	if err != nil {
		return errtype.ErrIntegrity(
			errtype.Join(ErrOpenArc, err),
		)
	}
	defer arcFile.Close()

	// Пропускаем магическое число и тип компрессора
	arcFile.Seek(headerLen, io.SeekStart)

	err = generic.ProcessHeaders(arcFile, headerLen, arc.integrityHeaderHandler)
	if err != nil {
		return errtype.ErrIntegrity(err)
	}

	return nil
}

// Обработчик заголовков архива для проверки целостности
func (arc Arc) integrityHeaderHandler(typ header.HeaderType, arcFile io.ReadSeekCloser) (err error) {
	switch typ {
	case header.File:
		if err = arc.checkFile(arcFile); err != nil {
			return errtype.ErrIntegrity(
				errtype.Join(ErrCheckFile, err),
			)
		}
	case header.Symlink:
		sym := &header.SymItem{} // Фактически пропускаем до следующего файла
		if err = sym.Read(arcFile); err != nil && err != io.EOF {
			return errtype.ErrIntegrity(
				errtype.Join(ErrReadSymHeader, err),
			)
		}
	default:
		return errtype.ErrIntegrity(ErrHeaderType)
	}
	return nil
}

// Распаковывает файл с проверкой CRC каждого блока сжатых данных
func (arc Arc) checkFile(arcFile io.ReadSeeker) (err error) {
	fi := &header.FileItem{}
	if err := fi.Read(arcFile); err != nil && err != io.EOF {
		return errtype.Join(ErrReadFileHeader, err)
	}

	if _, err = decompress.CheckCRC(arcFile, arc.Ct); err == ErrWrongCRC {
		fmt.Println(fi.PathOnDisk() + ": Файл поврежден")
	} else if err != nil {
		return errtype.Join(ErrCheckCRC, err)
	} else {
		fmt.Println(fi.PathOnDisk() + ": OK")
	}

	return nil
}
