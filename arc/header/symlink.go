package header

import (
	"archiver/filesystem"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Описание символической ссылки
type SymItem struct {
	basePaths
}

func NewSymDirItem(symlink, target string) *SymItem {
	return &SymItem{
		basePaths{pathOnDisk: target, pathInArc: symlink},
	}
}

// Создает директорию
func (si SymItem) RestorePath(outDir string) error {
	outDir = filepath.Join(outDir, si.pathInArc)

	if err := filesystem.CreatePath(filepath.Dir(outDir)); err != nil {
		return err
	}

	err := os.Symlink(si.pathOnDisk, outDir)
	if err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}

	fmt.Println(outDir, "->", si.pathOnDisk)

	return nil
}

// Реализация fmt.Stringer
func (si SymItem) String() string {
	filename := prefix(si.pathInArc, maxInArcWidth)
	target := prefix(si.pathOnDisk, maxOnDiskWidth)

	typ := func() string {
		if info, err := os.Stat(si.pathOnDisk); err != nil {
			return "Недейств."
		} else if info.Mode()&os.ModeDir != 0 {
			return "Директория"
		} else {
			return "Файл"
		}
	}()

	return fmt.Sprintf(
		"%-*s -> %s %*s", maxInArcWidth, filename,
		target, 38-len([]rune(target)), typ,
	)
}

// Сериализует в себя данные из r
func (si *SymItem) Read(r io.Reader) error {
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
func (si *SymItem) Write(w io.Writer) (err error) {
	filesystem.BinaryWrite(w, Symlink)

	// Пишем длину строки имени файла или директории
	if err = writePath(w, si.pathOnDisk); err != nil {
		return err
	}
	// Пишем длину строки имени файла или директории
	if err = writePath(w, si.pathInArc); err != nil {
		return err
	}
	fmt.Println(si.pathInArc, "->", si.pathOnDisk)

	return nil
}
