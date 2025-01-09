package arc

import (
	"archiver/arc/header"
	"encoding/binary"
	"os"
)

// Записывает информацию об архиве и заголовки
// директории в файл архива
func (arc Arc) writeHeaderDirs(dirs []*header.DirItem) (*os.File, error) {
	// Создаем файл
	arcFile, err := os.Create(arc.ArchivePath)
	if err != nil {
		return nil, err
	}

	// Пишем магическое число
	err = binary.Write(arcFile, binary.LittleEndian, magicNumber)
	if err != nil {
		return nil, err
	}

	// Пишем тип компрессора
	err = binary.Write(arcFile, binary.LittleEndian, arc.CompType)
	if err != nil {
		return nil, err
	}

	// Пишем количество заголовков директории
	err = binary.Write(arcFile, binary.LittleEndian, int64(len(dirs)))
	if err != nil {
		return nil, err
	}

	// Пишем заголовки
	for _, di := range dirs {
		di.Write(arcFile)
	}

	return arcFile, nil
}
