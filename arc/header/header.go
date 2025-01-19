package header

import (
	"fmt"
	"io"
	"math"
	"strings"
)

// Максимальная ширина имени файла
// в выводе статистики
const maxInArcWidth int = 20
const maxOnDiskWidth int = 28

const dateFormat string = "02.01.2006 15:04:05"

type HeaderType byte

const (
	Directory HeaderType = iota
	Symlink
	File
)

type Header interface {
	PathOnDisk() string // Путь к элементу заголовка
	PathInArc() string  // Путь к элементу в архиве
	String() string     // fmt.Stringer
}

type ReadWriter interface {
	Read(io.Reader) error  // Десериализует данные типа
	Write(io.Writer) error // Сериализует данные типа
}

type Restorable interface {
	RestorePath(string) error // Восстанаваливает доступность пути
}

// Реализация sort.Interface
type ByPath []Header

func (a ByPath) Len() int      { return len(a) }
func (a ByPath) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByPath) Less(i, j int) bool {
	return strings.ToLower(a[i].PathInArc()) < strings.ToLower(a[j].PathInArc())
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
func prefix(filename string, maxWidth int) string {
	runes := []rune(filename)
	count := len(runes)

	if count > maxWidth {
		filename = string(runes[count-(maxWidth-3):])
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
		path = h.PathInArc()
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
		maxInArcWidth, "Имя файла", "Размер",
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
		maxInArcWidth, "Итого",
		original, compressed, ratio,
	)
}
