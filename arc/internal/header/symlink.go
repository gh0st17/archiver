package header

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/gh0st17/archiver/filesystem"
)

// Описание символической ссылки
type SymItem struct {
	basePaths
}

// Создает заголовок символической ссылки [header.SymItem]
func NewSymItem(symlink, target string) *SymItem {
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

	return nil
}

// Реализация fmt.Stringer
func (si SymItem) String() string {
	filename := prefix(si.pathInArc, nameWidth)
	diff := terminalWidth - len([]rune(filename)) - 4
	target := prefix(si.pathOnDisk, diff)

	return fmt.Sprintf(
		"%-s%-*s", filename+" -> ",
		diff, target,
	)
}

// Десериализует в себя данные из r
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

	newSym := NewSymItem(symlink, target)
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

	return nil
}
