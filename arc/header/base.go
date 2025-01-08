package header

import (
	"archiver/filesystem"
	"encoding/binary"
	"fmt"
	"io"
	"time"
)

type Base struct {
	path    string
	accTime time.Time
	modTime time.Time
}

func (b Base) Path() string    { return b.path }
func (b Base) Atim() time.Time { return b.accTime }
func (b Base) Mtim() time.Time { return b.modTime }

func NewBase(path string, atim, mtim time.Time) Base {
	return Base{
		path:    path,
		accTime: atim,
		modTime: mtim,
	}
}

func (b *Base) Read(r io.Reader) error {
	var (
		err                  error
		length               int64
		filePathBytes        []byte
		unixMTime, unixATime int64
	)

	// Читаем размер строки имени файла или директории
	if err = binary.Read(r, binary.LittleEndian, &length); err != nil {
		return err
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

	*b = Base{
		path:    string(filePathBytes),
		modTime: time.Unix(unixMTime, 0),
		accTime: time.Unix(unixATime, 0),
	}

	return nil
}

func (b Base) Write(w io.Writer) (err error) {
	// Пишем длину строки имени файла или директории
	if checkRootSlash(&(b.path)) {
		fmt.Println("Удаляется начальный /")
	}

	b.path = filesystem.Clean(b.path)
	if err = binary.Write(w, binary.LittleEndian, int64(len(b.path))); err != nil {
		return err
	}

	// Пишем имя файла или директории
	if err = binary.Write(w, binary.BigEndian, []byte(b.path)); err != nil {
		return err
	}

	atime, mtime := b.accTime.Unix(), b.modTime.Unix()

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
