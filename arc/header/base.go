package header

import (
	"archiver/errtype"
	"archiver/filesystem"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

type Base struct {
	path string    // Путь к элементу
	atim time.Time // Последнее время доступа к элементу
	mtim time.Time // Последнее время измения элемента
}

// Возвращает путь до элемента
func (b Base) Path() string { return b.path }

func NewBase(path string, atim, mtim time.Time) Base {
	return Base{path, atim, mtim}
}

// Сериализует в себя данные из r
func (b *Base) Read(r io.Reader) error {
	var (
		err                  error
		length               int16
		filePathBytes        []byte
		unixMTime, unixATime int64
	)

	// Читаем размер строки имени файла или директории
	if err = binary.Read(r, binary.LittleEndian, &length); err != nil {
		return err
	}

	if length < 1 || length >= 1024 {
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
	if err = binary.Read(r, binary.LittleEndian, &unixMTime); err != nil {
		return err
	}

	// Читаем время доступа
	if err = binary.Read(r, binary.LittleEndian, &unixATime); err != nil {
		return err
	}

	mtim, atim := time.Unix(unixMTime, 0), time.Unix(unixATime, 0)
	*b = NewBase(string(filePathBytes), mtim, atim)

	return nil
}

// Сериализует данные полей в писатель w
func (b *Base) Write(w io.Writer) (err error) {
	// Пишем длину строки имени файла или директории
	b.path = filesystem.Clean(b.path)
	if err = binary.Write(w, binary.LittleEndian, int16(len(b.path))); err != nil {
		return err
	}

	// Пишем имя файла или директории
	if err = binary.Write(w, binary.LittleEndian, []byte(b.path)); err != nil {
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
	outDir = filepath.Join(outDir, b.path)
	if err := os.Chtimes(outDir, b.atim, b.mtim); err != nil {
		return err
	}

	return nil
}

func (b Base) createPath(outDir string) error {
	return filesystem.CreatePath(outDir)
}
