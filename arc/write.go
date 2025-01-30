package arc

import (
	"archiver/errtype"
	"archiver/filesystem"
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
		return nil, errtype.Join(ErrWriteMagic, err)
	}

	// Пишем тип компрессора
	if err = filesystem.BinaryWrite(arcFile, arc.Ct); err != nil {
		return nil, errtype.Join(ErrWriteCompType, err)
	}

	return arcFile, nil
}
