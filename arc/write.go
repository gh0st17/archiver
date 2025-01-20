package arc

import (
	"archiver/arc/header"
	"archiver/errtype"
	"archiver/filesystem"
	"io"
	"os"
)

// Создает файл архива и пишет информацию об архиве
func (arc Arc) writeArcHeader() (arcFile *os.File, err error) {
	// Создаем файл
	arcFile, err = os.Create(arc.arcPath)
	if err != nil {
		return nil, errtype.Join(ErrCreateArc, err)
	}

	// Пишем магическое число
	if err = filesystem.BinaryWrite(arcFile, magicNumber); err != nil {
		return nil, errtype.Join(ErrMagic, err)
	}

	// Пишем тип компрессора
	if err = filesystem.BinaryWrite(arcFile, arc.ct); err != nil {
		return nil, errtype.Join(ErrWriteCompType, err)
	}

	return arcFile, nil
}

// Записывает заголовки директории в файл архива
func (Arc) writeHeaders(writers []header.Writer, arcFile io.Writer) error {
	// Пишем количество заголовков директории
	if err := filesystem.BinaryWrite(arcFile, int64(len(writers))); err != nil {
		return errtype.Join(ErrWriteHeadersCount, err)
	}

	// Пишем заголовки
	for _, ds := range writers {
		if err := ds.Write(arcFile); err != nil {
			return errtype.Join(ErrWriteHeaderIO, err)
		}
	}

	return nil
}
