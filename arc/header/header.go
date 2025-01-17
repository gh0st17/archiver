package header

import (
	"archiver/filesystem"
	"fmt"
	"io"
	"math"
	"strings"
)

// Максимальная ширина имени файла
// в выводе статистики
const maxFilePathWidth int = 20

const dateFormat string = "02.01.2006 15:04:05"

type HeaderType byte

const (
	Directory HeaderType = iota
	File
)

type Header interface {
	Path() string          // Путь к элементу заголовка
	Read(io.Reader) error  // Считывет данные из `r`
	Write(io.Writer) error // Записывает данные в `w`
	String() string        // fmt.Stringer
}

// Реализация sort.Interface
type ByPath []Header

func (a ByPath) Len() int      { return len(a) }
func (a ByPath) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByPath) Less(i, j int) bool {
	return strings.ToLower(a[i].Path()) < strings.ToLower(a[j].Path())
}

type Size int64

// Реализация fmt.Stringer
func (bytes Size) String() string {
	const unit = 1000

	if bytes < unit {
		return fmt.Sprintf("%dБ", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%c",
		float64(bytes)/float64(div), []rune("КМГТПЭ")[exp])
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

// Проверяет пути к элементам и оставляет только
// уникальные заголовки по этому критерию
func DropDups(headers []Header) []Header {
	var (
		seen = map[string]struct{}{}
		uniq []Header
		path string
	)

	for _, h := range headers {
		path = filesystem.Clean(h.Path())
		if _, exists := seen[path]; !exists {
			seen[path] = struct{}{}
			uniq = append(uniq, h)
		}
	}

	return uniq
}

// Печатает заголовок статистики
func PrintStatHeader() {
	fmt.Printf( // Заголовок
		"%-*s %11s %11s %7s %10s  %19s %8s\n",
		maxFilePathWidth, "Имя файла", "Размер",
		"Сжатый", "%", "Тип", "Время модификации", "CRC32",
	)
}

// Печатает итог статистики
func PrintSummary(compressed, original Size) {
	ratio := float32(compressed) / float32(original) * 100.0

	if math.IsNaN(float64(ratio)) {
		ratio = 0.0
	}

	fmt.Printf( // Выводим итог
		"%-*s %11s %11s %7.2f\n",
		maxFilePathWidth, "Итого",
		original, compressed, ratio,
	)
}
