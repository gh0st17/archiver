package arc

import (
	"archiver/arc/header"
	"archiver/errtype"
	"encoding/binary"
	"os"
)

// Записывает информацию об архиве и заголовки
// директории в файл архива
func (arc Arc) writeHeaderDirs(dirs []*header.DirItem) (arcFile *os.File, err error) {
	// Создаем файл
	arcFile, err = os.Create(arc.arcPath)
	if err != nil {
		return nil, errtype.ErrRuntime("не могу создать файл архива", err)
	}

	// Пишем магическое число
	if err = binary.Write(arcFile, binary.LittleEndian, magicNumber); err != nil {
		return nil, errtype.ErrRuntime("ошибка записи магического числа", err)
	}

	// Пишем тип компрессора
	if err = binary.Write(arcFile, binary.LittleEndian, arc.ct); err != nil {
		return nil, errtype.ErrRuntime("ошибка записи типа компрессора", err)
	}

	// Пишем количество заголовков директории
	if err = binary.Write(arcFile, binary.LittleEndian, int64(len(dirs))); err != nil {
		return nil, errtype.ErrRuntime("ошибка записи количества заголовков", err)
	}

	// Пишем заголовки
	for _, di := range dirs {
		if err = di.Write(arcFile); err != nil {
			return nil, errtype.ErrRuntime("ошибка записи заголовка директории", err)
		}
	}

	return arcFile, nil
}
