package header

import (
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"

	"github.com/gh0st17/archiver/filesystem"
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

// Создает заголовок файла [header.FileItem]
func NewFileItem(base *Base, ucSize Size) *FileItem {
	return &FileItem{Base: *base, ucSize: ucSize}
}

// Десериализует заголовок файла из r
func (fi *FileItem) Read(r io.Reader) (err error) {
	if err = fi.Base.Read(r); err != nil {
		return err
	}

	// Читаем размер файла до сжатия
	if err = filesystem.BinaryRead(r, &(fi.ucSize)); err != nil {
		return err
	}

	return nil
}

// Cериализует заголовок файла из r
func (fi *FileItem) Write(w io.Writer) (err error) {
	filesystem.BinaryWrite(w, File)

	if err = fi.Base.Write(w); err != nil {
		return err
	}

	// Пишем размер файла до сжатия
	if err = filesystem.BinaryWrite(w, fi.ucSize); err != nil {
		return err
	}

	return nil
}

// Восстанавливает путь к файлу
func (fi FileItem) RestorePath(outDir string) error {
	outDir = filepath.Join(outDir, fi.pathOnDisk)
	if err := os.MkdirAll(filepath.Dir(outDir), 0755); err != nil {
		return err
	}

	return nil
}

// Реализация fmt.Stringer
func (fi FileItem) String() string {
	path := prefix(fi.pathOnDisk, nameWidth)

	ratio := float32(fi.cSize) / float32(fi.ucSize) * 100.0
	if math.IsInf(float64(ratio), 1) {
		ratio = 0
	} else if math.IsNaN(float64(ratio)) {
		ratio = 0
	}

	mtime := fi.mtim.Format(dateFormat)
	crc := func() string {
		if fi.crc != 0 {
			return fmt.Sprintf("%8X", fi.crc)
		} else {
			return "-"
		}
	}()

	return fmt.Sprintf(
		"%-*s  %6s  %6s  %7.2f  %s  %s",
		nameWidth, path, fi.ucSize,
		fi.cSize, ratio, mtime, crc,
	)
}
