package header

import (
	"archiver/filesystem"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type SymDirItem struct {
	basePaths
}

func NewSymDirItem(symlink, target string) *SymDirItem {
	return &SymDirItem{
		basePaths{pathOnDisk: target, pathInArc: symlink},
	}
}

// Создает директорию
func (si SymDirItem) RestorePath(outDir string) error {
	outDir = filepath.Join(outDir, si.pathInArc)

	if err := filesystem.CreatePath(filepath.Dir(outDir)); err != nil {
		return err
	}

	if err := os.Symlink(si.pathOnDisk, outDir); err != nil {
		return err
	}

	return nil
}

// Реализация fmt.Stringer
func (si SymDirItem) String() string {
	filename := prefix(si.pathInArc)

	return fmt.Sprintf(
		"%-*s %42s", maxFilePathWidth,
		filename, "Ссылка",
	)
}

// Сериализует в себя данные из r
func (si *SymDirItem) Read(r io.Reader) error {
	var (
		err     error
		target  string
		symlink string
	)

	// Читаем размер строки target
	if target, err = readPath(r); err != nil {
		return err
	}

	// Читаем размер строки symlink
	if symlink, err = readPath(r); err != nil {
		return err
	}

	newSym := NewSymDirItem(symlink, target)
	*si = *newSym

	return err
}

// Сериализует данные полей в писатель w
func (si *SymDirItem) Write(w io.Writer) (err error) {
	// Пишем длину строки имени файла или директории
	if err = writePath(w, si.pathOnDisk); err != nil {
		return err
	}
	// Пишем длину строки имени файла или директории
	if err = writePath(w, si.pathInArc); err != nil {
		return err
	}

	return nil
}
