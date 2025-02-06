// Пакет header предоставляет набор типов для структурирования
// элементов файловой системы в архиве и функции для работы
// с этими типами
package header

import (
	"fmt"
	"math"
	"strings"

	"github.com/gh0st17/archiver/arc/internal/header/platform"
	"github.com/gh0st17/archiver/arc/internal/userinput"
)

// Максимальная ширина имени файла
// в выводе статистики
var terminalWidth, nameWidth int

func IsEnoughWidth() bool { return nameWidth >= 9 }

// Возвращает на сколько столбцов нужно увеличить окно терминала
func ExtraWidth() int {
	if nameWidth-9 < 0 {
		return -(nameWidth - 9)
	}
	return 9 - nameWidth
}

const dateFormat string = "02.01.2006 15:04:05"

type HeaderType byte

const (
	Symlink HeaderType = iota
	File
)

type Header interface {
	PathProvider
	String() string // fmt.Stringer
}

// Реализация sort.Interface
type ByPathInArc []Header

func (a ByPathInArc) Len() int      { return len(a) }
func (a ByPathInArc) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByPathInArc) Less(i, j int) bool {
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
	fmt.Printf(
		"%-*s  %6s  %6s  %7s  %19s  %8s\n",
		nameWidth, "Имя файла", "Размер",
		"Сжатый", "%", "Время модификации", "CRC32",
	)
}

// Печатает итог статистики
func PrintSummary(compressed, original Size) {
	ratio := float32(compressed) / float32(original) * 100.0

	if math.IsNaN(float64(ratio)) {
		ratio = 0.0
	}

	fmt.Printf( // Выводим итог
		"%-*s  %6s  %6s  %7.2f\n",
		nameWidth, "Итого",
		original, compressed, ratio,
	)
}

// Сокращает длинные имена файлов, добавляя '...' в начале
func prefix(filename string, maxWidth int) string {
	runes := []rune(filename)
	count := len(runes)

	if count > maxWidth {
		filename = string(runes[count-(maxWidth-3):])
		return "..." + filename
	} else {
		return filename
	}
}

func init() {
	if userinput.IsNonInteractive() {
		return
	}

	var err error
	terminalWidth, _, err = platform.GetTerminalSize()
	if err != nil {
		panic(fmt.Sprint("header.init: ", err))
	}

	// Ширина терминала минус ширина заголовка
	// без имени первого столбца
	nameWidth = terminalWidth - 56
}
