package header

import (
	"fmt"
	"io"
	"path/filepath"
)

// Описание директории
type DirItem struct {
	Base
}

func NewDirItem(base Base) DirItem { return DirItem{base} }

func (di *DirItem) Read(r io.Reader) error {
	if err := di.Base.Read(r); err != nil {
		return err
	}

	return nil
}

func (di DirItem) RestorePath(outDir string) error {
	outDir = filepath.Join(outDir, di.path)
	if err := di.createPath(outDir); err != nil {
		return err
	}

	return nil
}

// Реализация fmt.Stringer
func (di DirItem) String() string {
	filename := prefix(di.path)
	mtime := di.modTime.Format(dateFormat)

	return fmt.Sprintf(
		"%-*s %42s  %s", maxFilePathWidth,
		filename, "Директория", mtime,
	)
}
