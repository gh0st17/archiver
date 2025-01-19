package arc

import (
	"archiver/arc/header"
	"archiver/errtype"
	"archiver/filesystem"
	"fmt"
	"os"
)

// Записывает информацию об архиве и заголовки
// директории в файл архива
func (arc Arc) writeHeaderDirsSyms(dirsSyms []header.ReadWriter) (arcFile *os.File, err error) {
	// Создаем файл
	arcFile, err = os.Create(arc.arcPath)
	if err != nil {
		return nil, errtype.ErrRuntime(ErrCreateArc, err)
	}

	// Пишем магическое число
	if err = filesystem.BinaryWrite(arcFile, magicNumber); err != nil {
		return nil, errtype.ErrRuntime(ErrMagic, err)
	}

	// Пишем тип компрессора
	if err = filesystem.BinaryWrite(arcFile, arc.ct); err != nil {
		return nil, errtype.ErrRuntime(ErrWriteCompType, err)
	}

	// Пишем количество заголовков директории
	if err = filesystem.BinaryWrite(arcFile, int64(len(dirsSyms))); err != nil {
		return nil, errtype.ErrRuntime(ErrWriteHeadersCount, err)
	}

	// Пишем заголовки
	for _, h := range dirsSyms {
		if err = h.Write(arcFile); err != nil {
			return nil, errtype.ErrRuntime(ErrWriteHeaderIO, err)
		}
		fmt.Println(h.(header.PathProvider).PathInArc())
	}

	return arcFile, nil
}
