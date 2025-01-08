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
	ucSize, cSize Size
	crc           uint32
	damaged       bool
}

// Возвращает размер данных в несжатом виде
func (fi FileItem) UcSize() Size { return fi.ucSize }

// Возвращает размер данных в сжатом виде
func (fi FileItem) CSize() Size { return fi.cSize }

// Возвращает констрольную сумму CRC32
func (fi FileItem) CRC() uint32 { return fi.crc }

// Возвращает флаг наличия повреждении
func (fi FileItem) IsDamaged() bool { return fi.damaged }

// Устанавливает размер данных в несжатом виде
func (fi *FileItem) SetUcSize(size Size) { fi.ucSize = size }

// Устанавливает размер данных в сжатом виде
func (fi *FileItem) SetCSize(size Size) { fi.cSize = size }

// Устанавливает констрольную сумму CRC32
func (fi *FileItem) SetCRC(crc uint32) { fi.crc = crc }

// Устанавливает флаг наличия повреждении
func (fi *FileItem) SetDamaged(damaged bool) { fi.damaged = damaged }

func NewFileItem(base Base, ucSize Size) FileItem {
	return FileItem{
		Base:   base,
		ucSize: ucSize,
	}
}

func (fi *FileItem) Read(r io.Reader) (err error) {
	if err = fi.Base.Read(r); err != nil {
		return err
	}

	// Читаем размер файла до сжатия
	if err = binary.Read(r, binary.LittleEndian, &(fi.ucSize)); err != nil {
		return err
	}

	return nil
}

func (fi FileItem) Write(w io.Writer) (err error) {
	if err = fi.Base.Write(w); err != nil {
		return err
	}

	// Пишем размер файла до сжатия
	if err = binary.Write(w, binary.LittleEndian, fi.ucSize); err != nil {
		return err
	}

	return nil
}

// Реализация fmt.Stringer
func (fi FileItem) String() string {
	path := prefix(fi.path)

	ratio := float32(fi.cSize) / float32(fi.ucSize) * 100.0
	if math.IsInf(float64(ratio), 1) {
		ratio = 0
	} else if math.IsNaN(float64(ratio)) {
		ratio = 0
	}

	mtime := fi.modTime.Format(dateFormat)

	return fmt.Sprintf(
		"%-*s %11s %11s %6.2f %10s  %s %8X",
		maxFilePathWidth, path, fi.ucSize,
		fi.cSize, ratio, "Файл", mtime, fi.crc,
	)
}
