package header

import (
	"fmt"
	"io"
	"time"
)

// Описание директории
type DirItem struct {
	Base
}

func (DirItem) Type() HeaderType { return Directory }

func (di *DirItem) Read(r io.Reader) error {
	if err := di.Base.Read(r); err != nil {
		return err
	}

	return nil
}

func (di DirItem) Write(w io.Writer) error {
	if err := di.Base.Write(w); err != nil {
		return err
	}

	return nil
}

// Реализация fmt.Stringer
func (di DirItem) String() string {
	filename := prefix(di.Filepath)
	mtime := di.ModTime.Format(time.UnixDate)

	return fmt.Sprintf(
		"%-*s %41s  %s", maxFilePathWidth,
		filename, "Директория", mtime,
	)
}
