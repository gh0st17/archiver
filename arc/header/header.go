package header

import (
	"encoding/binary"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"
)

// Максимальная ширина имени файла
// в выводе статистики
const maxFilePathWidth int = 18

type HeaderType byte

const (
	Directory HeaderType = iota
	File
)

type Header interface {
	Path() string            // Путь к элементу заголовка
	Type() HeaderType        // Идетификация типа элемента структуры
	Read(r io.Reader) error  // Считывет данные из `r`
	Write(w io.Writer) error // Записывает данные в `w`
	String() string          // fmt.Stringer
}

// Реализация sort.Interface
type ByPath []Header

func (a ByPath) Len() int           { return len(a) }
func (a ByPath) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByPath) Less(i, j int) bool { return a[i].Path() < a[j].Path() }

type Base struct {
	Filepath string
	AccTime  time.Time
	ModTime  time.Time
}

func (b Base) Path() string { return b.Filepath }

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

type Size int64

// fmt.Stringer
func (bytes Size) String() string {
	const unit = 1024

	if bytes < unit {
		return fmt.Sprintf("%d Б  ", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cиБ",
		float64(bytes)/float64(div), []rune("КМГТПЭ")[exp])
}

// Проверяет наличие признака абсолютного пути
func checkRootSlash(path *string) bool {
	if filepath.IsAbs(*path) || strings.HasPrefix(*path, "/") {
		*path = strings.TrimPrefix(*path, "/")
		return true
	}

	return false
}

// Сокращает длинные имена файлов, добавляя '...' в начале
func prefix(filename string) string {
	if len(filename) > maxFilePathWidth {
		filename = filename[len(filename)-(maxFilePathWidth-3):]
		return string("..." + filename)
	} else {
		return filename
	}
}

func PrintStatHeader() {
	fmt.Printf( // Заголовок
		"%-*s %11s %11s %6s %10s %28s\n",
		maxFilePathWidth, "Имя файла", "Размер",
		"Сжатый", "%", "Тип", "Время модификации",
	)
}

func PrintSummary(compressed, original Size) {
	ratio := float32(compressed) / float32(original) * 100.0
	fmt.Printf( // Выводим итог
		"%-*s %11s %11s %6.2f\n",
		maxFilePathWidth, "Итого",
		original, compressed, ratio,
	)
}
