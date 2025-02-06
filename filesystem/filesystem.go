// Пакет filesystem предоставляет набор функции для работы
// с файловой системой и ее элементами
package filesystem

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Разбивает путь на компоненты
func SplitPath(path string) []string {
	if path == "/" {
		return []string{path}
	}

	parts := strings.Split(filepath.ToSlash(path), "/")

	var cleanedParts []string
	for _, part := range parts {
		if part != "" {
			cleanedParts = append(cleanedParts, part)
		}
	}

	return cleanedParts
}

// Проверяет существование директории
func DirExists(dirPath string) bool {
	if info, err := os.Stat(dirPath); err != nil {
		return false
	} else {
		return info.IsDir()
	}
}

// Номализует путь
func Clean(path string) string {
	path = filepath.ToSlash(path)
	path = strings.TrimPrefix(path, "/")
	parts := strings.Split(path, "/")
	stack := []string{}

	for _, part := range parts {
		switch part {
		case ".", "":
			// Игнорируем текущую директорию или пустые части
			continue
		case "..":
			// Удаляем предыдущий элемент, если он есть
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		default:
			// Добавляем нормальный компонент пути
			stack = append(stack, part)
		}
	}

	return strings.Join(stack, "/")
}

// Печатает предупреждение что абсолютные и
// относительные пути будут храниться в
// упрощенном виде
func PrintPathsCheck(paths []string) {
	var prefix = map[string]struct{}{}
	for _, p := range paths {
		p = filepath.ToSlash(p)
		path := Clean(p)
		deleted := strings.TrimSuffix(p, path)
		if len(deleted) > 0 {
			if _, exists := prefix[deleted]; !exists {
				prefix[deleted] = struct{}{}

				fmt.Printf(
					"Удаляется начальный '%s' из имен путей\n",
					deleted,
				)
			}
		}
	}
}

// Обертка для [binary.Write] с порядком следования
// байт [binary.LittleEndian]
func BinaryWrite(w io.Writer, data any) error {
	return binary.Write(w, binary.LittleEndian, data)
}

// Обертка для [binary.Read] с порядком следования
// байт [binary.LittleEndian]
func BinaryRead(r io.Reader, data any) error {
	return binary.Read(r, binary.LittleEndian, data)
}
