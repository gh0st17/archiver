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
	path    string
	accTime time.Time
	modTime time.Time
}

// Возвращает путь до элемента
func (b Base) Path() string { return b.path }

func NewBase(path string, atim, mtim time.Time) Base {
	return Base{
		path:    path,
		accTime: atim,
		modTime: mtim,
	}
}

// Сериализует в себя данные из r
func (b *Base) Read(r io.Reader) error {
	var (
		err                  error
		length               int64
		filePathBytes        []byte
		unixMTime, unixATime int64
	)

	// Читаем размер строки имени файла или директории
	if err = binary.Read(r, binary.LittleEndian, &length); err != nil {
		return errtype.ErrRuntime("ошибка чтения длины пути элемента", err)
	}

	if length < 1 || length >= 1024 {
		return errtype.ErrRuntime(
			fmt.Sprintf("некорректная длина (%d) пути элемента", length), nil,
		)
	}

	// Читаем имя файла
	filePathBytes = make([]byte, length)
	if _, err := io.ReadFull(r, filePathBytes); err != nil {
		return errtype.ErrRuntime("ошибка чтения пути элемента", err)
	}

	// Читаем время модификации
	if err = binary.Read(r, binary.LittleEndian, &unixMTime); err != nil {
		return errtype.ErrRuntime("ошибка чтения времени модификации", err)
	}

	// Читаем время доступа
	if err = binary.Read(r, binary.LittleEndian, &unixATime); err != nil {
		return errtype.ErrRuntime("ошибка чтения времени доступа", err)
	}

	*b = Base{
		path:    string(filePathBytes),
		modTime: time.Unix(unixMTime, 0),
		accTime: time.Unix(unixATime, 0),
	}

	return nil
}

// Сериализует данные полей в писатель w
func (b Base) Write(w io.Writer) (err error) {
	// Пишем длину строки имени файла или директории
	if checkRootSlash(&(b.path)) {
		fmt.Println("Удаляется начальный /")
	}

	b.path = filesystem.Clean(b.path)
	if err = binary.Write(w, binary.LittleEndian, int64(len(b.path))); err != nil {
		return errtype.ErrRuntime("ошибка записи длины пути элемента", err)
	}

	// Пишем имя файла или директории
	if err = binary.Write(w, binary.BigEndian, []byte(b.path)); err != nil {
		return errtype.ErrRuntime("ошибка записи пути элемента", err)
	}

	atime, mtime := b.accTime.Unix(), b.modTime.Unix()

	// Пишем время модификации
	if err = binary.Write(w, binary.LittleEndian, mtime); err != nil {
		return errtype.ErrRuntime("ошибка записи времени модификации", err)
	}

	// Пишем имя время доступа
	if err = binary.Write(w, binary.LittleEndian, atime); err != nil {
		return errtype.ErrRuntime("ошибка записи времени доступа", err)
	}

	return nil
}
func (b Base) RestoreTime(outDir string) error {
	outDir = filepath.Join(outDir, b.path)
	if err := os.Chtimes(outDir, b.accTime, b.modTime); err != nil {
		return errtype.ErrDecompress(
			fmt.Sprintf(
				"не могу установить аттрибуты времени для директории '%s'", outDir,
			), err,
		)
	}

	return nil
}

func (b Base) createPath(outDir string) error {
	return filesystem.CreatePath(outDir)
}
