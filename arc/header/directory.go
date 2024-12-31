package header

import (
	"encoding/binary"
	"fmt"
	"io"
)

// Описание директории
type DirItem struct {
	Base
}

func (di *DirItem) Read(r io.Reader) error {
	if err := di.Base.Read(r); err != nil {
		return err
	}

	return nil
}

func (di DirItem) Write(w io.Writer) (err error) {
	// Пишем тип заголовка
	if err = binary.Write(w, binary.LittleEndian, Directory); err != nil {
		return err
	}

	if err = di.Base.Write(w); err != nil {
		return err
	}

	return nil
}

// Реализация fmt.Stringer
func (di DirItem) String() string {
	filename := prefix(di.Filepath)
	mtime := di.ModTime.Format(dateFormat)

	return fmt.Sprintf(
		"%-*s %41s  %s", maxFilePathWidth,
		filename, "Директория", mtime,
	)
}
