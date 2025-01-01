package header

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

// Описание файла
type FileItem struct {
	Base
	UncompressedSize Size
	CompressedSize   Size
	CRC              uint32
}

func (fi *FileItem) Read(r io.Reader) (err error) {
	if err = fi.Base.Read(r); err != nil {
		return err
	}

	// Читаем размер файла до сжатия
	if err = binary.Read(r, binary.LittleEndian, &(fi.UncompressedSize)); err != nil {
		return err
	}

	// Читаем размер файла после сжатия
	if err = binary.Read(r, binary.LittleEndian, &(fi.CompressedSize)); err != nil {
		return err
	}

	// Читаем контрольную сумму
	if err = binary.Read(r, binary.LittleEndian, &(fi.CRC)); err != nil {
		return err
	}

	return nil
}

func (fi FileItem) Write(w io.Writer) (err error) {
	// Пишем тип заголовка
	if err = binary.Write(w, binary.LittleEndian, File); err != nil {
		return err
	}

	if err = fi.Base.Write(w); err != nil {
		return err
	}

	// Пишем размер файла до сжатия
	if err = binary.Write(w, binary.LittleEndian, fi.UncompressedSize); err != nil {
		return err
	}

	// Пишем размер файла после сжатия
	if err = binary.Write(w, binary.LittleEndian, fi.CompressedSize); err != nil {
		return err
	}

	// Пишем контрольную сумму
	if err = binary.Write(w, binary.LittleEndian, fi.CRC); err != nil {
		return err
	}

	return nil
}

// Реализация fmt.Stringer
func (fi FileItem) String() string {
	filepath := prefix(fi.Filepath)

	ratio := float32(fi.CompressedSize) / float32(fi.UncompressedSize) * 100.0
	if math.IsInf(float64(ratio), 1) {
		ratio = 0
	} else if math.IsNaN(float64(ratio)) {
		ratio = 0
	}

	mtime := fi.ModTime.Format(dateFormat)

	return fmt.Sprintf(
		"%-*s %11d %11d %6.2f %10s  %s %8X",
		maxFilePathWidth, filepath, fi.UncompressedSize,
		fi.CompressedSize, ratio, "Файл", mtime, fi.CRC,
	)
}
