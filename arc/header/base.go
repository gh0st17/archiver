package header

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"
)

type Base struct {
	Filepath string
	AccTime  time.Time
	ModTime  time.Time
}

func (b Base) Path() string { return b.Filepath }

func (b *Base) SetPath(path string) {
	b.Filepath = path
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
		Filepath: string(filePathBytes),
		ModTime:  time.Unix(unixMTime, 0),
		AccTime:  time.Unix(unixATime, 0),
	}

	return nil
}

func (b Base) Write(w io.Writer) (err error) {
	// Пишем длину строки имени файла или директории
	if checkRootSlash(&(b.Filepath)) {
		fmt.Println("Удаляется начальный /")
	}

	if err = binary.Write(w, binary.LittleEndian, int64(len(b.Filepath))); err != nil {
		return err
	}

	// Пишем имя файла или директории
	if err = binary.Write(w, binary.BigEndian, []byte(b.Filepath)); err != nil {
		return err
	}

	atime, mtime := b.AccTime.Unix(), b.ModTime.Unix()

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
