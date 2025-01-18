package header

import (
	"archiver/filesystem"
	"fmt"
	"path/filepath"
)

// Описание директории
type DirItem struct {
	Base
}

func NewDirItem(base *Base) *DirItem { return &DirItem{*base} }

// Создает директорию
func (di DirItem) RestorePath(outDir string) error {
	outDir = filepath.Join(outDir, di.pathOnDisk)
	if err := filesystem.CreatePath(outDir); err != nil {
		return err
	}

	return nil
}

// Реализация fmt.Stringer
func (di DirItem) String() string {
	filename := prefix(di.pathOnDisk)
	mtime := di.mtim.Format(dateFormat)

	return fmt.Sprintf(
		"%-*s %42s  %s", maxFilePathWidth,
		filename, "Директория", mtime,
	)
}
