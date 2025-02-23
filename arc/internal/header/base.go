package header

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/gh0st17/archiver/filesystem"
)

type PathProvider interface {
	PathOnDisk() string // Путь к элементу заголовка
	PathInArc() string  // Путь к элементу в архиве
}

type timeAttr struct {
	atim time.Time // Последнее время доступа к элементу
	mtim time.Time // Последнее время измения элемента
}

type basePaths struct {
	pathOnDisk string // Путь к элементу на диске
	pathInArc  string // Путь к элементу в архиве
}

func (b basePaths) PathOnDisk() string { return b.pathOnDisk }
func (b basePaths) PathInArc() string  { return b.pathInArc }

// Дериализует путь из r
func readPath(r io.Reader) (_ string, err error) {
	var length int16

	if err = filesystem.BinaryRead(r, &length); err != nil {
		return "", err
	}

	if length < 1 || length > 1023 {
		return "", ErrPathLength(int64(length))
	}

	pathBytes := make([]byte, length)
	if _, err = io.ReadFull(r, pathBytes); err != nil {
		return "", err
	}

	return string(pathBytes), nil
}

// Сериализует путь path в w
func writePath(w io.Writer, path string) (err error) {
	// Пишем длину строки имени файла или директории
	if err = filesystem.BinaryWrite(w, int16(len(path))); err != nil {
		return err
	}
	log.Println("arc.header.writePath: Записана длина пути:", int16(len(path)))

	// Пишем имя файла или директории
	if err = filesystem.BinaryWrite(w, []byte(path)); err != nil {
		return err
	}
	log.Println("arc.header.writePath: Записан путь:", path)

	return nil
}

type Base struct {
	basePaths
	timeAttr
}

// Создает новый [header.Base]
func NewBase(pathOnDisk string, atim, mtim time.Time) (*Base, error) {
	if len(pathOnDisk) > 1023 {
		return nil, ErrLongPath(pathOnDisk)
	}

	pathInArc := filesystem.Clean(pathOnDisk)

	return &Base{
		basePaths{pathOnDisk, pathInArc},
		timeAttr{atim, mtim},
	}, nil
}

// Десериализует в себя данные из r
func (b *Base) Read(r io.Reader) error {
	var (
		err      error
		path     string
		unixMtim int64
		unixAtim int64
	)

	// Читаем имя файла
	if path, err = readPath(r); err != nil {
		return err
	}

	// Читаем время модификации
	if err = filesystem.BinaryRead(r, &unixMtim); err != nil {
		return err
	}

	// Читаем время доступа
	if err = filesystem.BinaryRead(r, &unixAtim); err != nil {
		return err
	}

	mtim, atim := time.Unix(unixMtim, 0), time.Unix(unixAtim, 0)
	newBase, _ := NewBase(path, mtim, atim)
	*b = *newBase

	return err
}

// Сериализует данные полей в писатель w
func (b *Base) Write(w io.Writer) (err error) {
	// Пишем имя файла или директории
	if err = writePath(w, b.pathInArc); err != nil {
		return err
	}

	atime, mtime := b.atim.Unix(), b.mtim.Unix()

	// Пишем время модификации
	if err = filesystem.BinaryWrite(w, mtime); err != nil {
		return err
	}

	// Пишем имя время доступа
	if err = filesystem.BinaryWrite(w, atime); err != nil {
		return err
	}

	return nil
}

func (b Base) RestoreTime(outDir string) error {
	outDir = filepath.Join(outDir, b.pathOnDisk)
	if err := os.Chtimes(outDir, b.atim, b.mtim); err != nil {
		return err
	}

	return nil
}
