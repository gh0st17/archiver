package arc

import (
	"archiver/arc/internal/decompress"
	"archiver/arc/internal/header"
	"archiver/errtype"
	"archiver/filesystem"
	"fmt"
	"io"
	"os"
)

// Проверяет целостность данных в архиве
func (arc Arc) IntegrityTest() error {
	arcFile, err := os.OpenFile(arc.arcPath, os.O_RDONLY, 0644)
	if err != nil {
		return errtype.ErrIntegrity(
			errtype.Join(ErrOpenArc, err),
		)
	}
	defer arcFile.Close()

	// Пропускаем магическое число и тип компрессора
	arcFile.Seek(arcHeaderLen, io.SeekStart)

	var typ header.HeaderType

	for err != io.EOF {
		err = filesystem.BinaryRead(arcFile, &typ)
		if err != io.EOF && err != nil {
			return errtype.ErrIntegrity(
				errtype.Join(ErrReadHeaderType, err),
			)
		} else if err == io.EOF {
			continue
		}

		switch typ {
		case header.File:
			if err = arc.checkFile(arcFile); err != nil {
				return errtype.ErrIntegrity(
					errtype.Join(ErrCheckFile, err),
				)
			}
		case header.Symlink:
			sym := &header.SymItem{}
			if err = sym.Read(arcFile); err != nil && err != io.EOF {
				return errtype.ErrIntegrity(
					errtype.Join(ErrReadSymHeader, err),
				)
			}
		default:
			return errtype.ErrIntegrity(ErrHeaderType)
		}
	}

	return nil
}

// Распаковывает файл с проверкой CRC каждого
// блока сжатых данных
func (arc Arc) checkFile(arcFile io.ReadSeeker) error {
	var err error

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
