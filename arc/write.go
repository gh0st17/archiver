package arc

import (
	"archiver/arc/header"
	"archiver/errtype"
	"archiver/filesystem"
	"encoding/binary"
	"os"
)

// Записывает информацию об архиве и заголовки
// директории в файл архива
func (arc Arc) writeHeaderDirsSyms(dirsSyms []header.Header) (arcFile *os.File, err error) {
	// Создаем файл
	arcFile, err = os.Create(arc.arcPath)
	if err != nil {
		return nil, errtype.ErrRuntime(ErrCreateArc, err)
	}

	// Пишем магическое число
	if err = binary.Write(arcFile, binary.LittleEndian, magicNumber); err != nil {
		return nil, errtype.ErrRuntime(ErrMagic, err)
	}

	// Пишем тип компрессора
	if err = binary.Write(arcFile, binary.LittleEndian, arc.ct); err != nil {
		return nil, errtype.ErrRuntime(ErrWriteCompType, err)
	}

	// Пишем количество заголовков директории
	if err = binary.Write(arcFile, binary.LittleEndian, int64(len(dirsSyms))); err != nil {
		return nil, errtype.ErrRuntime(ErrWriteHeadersCount, err)
	}

	// Пишем заголовки
	for _, h := range dirsSyms {
		if di, ok := h.(*header.DirItem); ok {
			filesystem.BinaryWrite(arcFile, byte(0))
			if err = di.Write(arcFile); err != nil {
				return nil, errtype.ErrRuntime(ErrWriteDirHeader, err)
			}
		} else if si, ok := h.(*header.SymDirItem); ok {
			filesystem.BinaryWrite(arcFile, byte(1))
			if err = si.Write(arcFile); err != nil {
				return nil, errtype.ErrRuntime(ErrWriteDirHeader, err)
			}
		}
	}

	return arcFile, nil
}
