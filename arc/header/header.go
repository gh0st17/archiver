package header

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// Максимальная ширина имени файла
// в выводе статистики
const maxFilePathWidth int = 18

const dateFormat string = "02.01.2006 15:04:05"

type HeaderType byte

const (
	Directory HeaderType = iota
	File
)

type Header interface {
	Path() string            // Путь к элементу заголовка
	Read(r io.Reader) error  // Считывет данные из `r`
	Write(w io.Writer) error // Записывает данные в `w`
	String() string          // fmt.Stringer
}

// Реализация sort.Interface
type ByPath []Header

func (a ByPath) Len() int           { return len(a) }
func (a ByPath) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByPath) Less(i, j int) bool { return a[i].Path() < a[j].Path() }

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

// Печатает заголовок статистики
func PrintStatHeader() {
	fmt.Printf( // Заголовок
		"%-*s %11s %11s %6s %10s  %19s %8s\n",
		maxFilePathWidth, "Имя файла", "Размер",
		"Сжатый", "%", "Тип", "Время модификации", "CRC32",
	)
}

// Печатает итог статистики
func PrintSummary(compressed, original Size) {
	ratio := float32(compressed) / float32(original) * 100.0
	fmt.Printf( // Выводим итог
		"%-*s %11d %11d %6.2f\n",
		maxFilePathWidth, "Итого",
		original, compressed, ratio,
	)
}
