package header

import (
	"archiver/filesystem"
	"fmt"
	"io"
	"path/filepath"
)

// Описание директории
type DirItem struct {
	Base
}

func NewDirItem(base *Base) *DirItem { return &DirItem{*base} }

// Создает директорию
func (di DirItem) RestorePath(outDir string) error {
	completePath := filepath.Join(outDir, di.pathOnDisk)
	if err := filesystem.CreatePath(completePath); err != nil {
		return err
	}
	fmt.Println(completePath)

	return di.RestoreTime(outDir)
}

// Реализация fmt.Stringer
func (di DirItem) String() string {
	filename := prefix(di.pathOnDisk, maxInArcWidth)
	mtime := di.mtim.Format(dateFormat)

	return fmt.Sprintf(
		"%-*s %42s  %s", maxInArcWidth,
		filename, "Директория", mtime,
	)
}

func (di *DirItem) Write(w io.Writer) (err error) {
	filesystem.BinaryWrite(w, Directory)

	if err = di.Base.Write(w); err != nil {
		return err
	}
	fmt.Println(di.pathInArc)

	return nil
}
