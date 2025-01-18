package header

import (
	"archiver/errtype"
	"archiver/filesystem"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

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

type Base struct {
	basePaths
	timeAttr
}

func NewBase(pathOnDisk string, atim, mtim time.Time) (*Base, error) {
	pathInArc := filesystem.Clean(pathOnDisk)

	if len(pathInArc) > 1023 {
		return nil, errors.New("длина пути в архиве превышает допустимую")
	}

	return &Base{
		basePaths{pathOnDisk, pathInArc},
		timeAttr{atim, mtim},
	}, nil
}

// Сериализует в себя данные из r
func (b *Base) Read(r io.Reader) error {
	var (
		err                error
		length             int16
		filePathBytes      []byte
		unixMtim, unixAtim int64
	)

	// Читаем размер строки имени файла или директории
	if err = binary.Read(r, binary.LittleEndian, &length); err != nil {
		return err
	}

	if length < 1 || length > 1023 {
		return errtype.ErrRuntime(
			fmt.Errorf("некорректная длина (%d) пути элемента", length), nil,
		)
	}

	// Читаем имя файла
	filePathBytes = make([]byte, length)
	if _, err := io.ReadFull(r, filePathBytes); err != nil {
		return err
	}

	// Читаем время модификации
	if err = binary.Read(r, binary.LittleEndian, &unixMtim); err != nil {
		return err
	}

	// Читаем время доступа
	if err = binary.Read(r, binary.LittleEndian, &unixAtim); err != nil {
		return err
	}

	mtim, atim := time.Unix(unixMtim, 0), time.Unix(unixAtim, 0)
	newBase, _ := NewBase(string(filePathBytes), mtim, atim)
	*b = *newBase

	return err
}

// Сериализует данные полей в писатель w
func (b *Base) Write(w io.Writer) (err error) {
	// Пишем длину строки имени файла или директории
	if err = binary.Write(w, binary.LittleEndian, int16(len(b.pathInArc))); err != nil {
		return err
	}

	// Пишем имя файла или директории
	if err = binary.Write(w, binary.LittleEndian, []byte(b.pathInArc)); err != nil {
		return err
	}

	atime, mtime := b.atim.Unix(), b.mtim.Unix()

	// Пишем время модификации
	if err = binary.Write(w, binary.LittleEndian, mtime); err != nil {
		return err
	}

	// Пишем имя время доступа
	if err = binary.Write(w, binary.LittleEndian, atime); err != nil {
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
