package filesystem

import (
	"encoding/binary"
	"errors"
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

	dir, last := filepath.Split(path)
	if dir == "" {
		return []string{last}
	}
	return append(SplitPath(filepath.Clean(dir)), last)
}

// Создает директории на всем пути `path`
func CreatePath(path string) error {
	var (
		splitedPath = SplitPath(path)
		fullPath    string
	)

	for _, pathPart := range splitedPath {
		fullPath = filepath.Join(fullPath, pathPart)

		if DirExists(fullPath) {
			continue
		}

		if err := os.Mkdir(fullPath, 0755); err != nil {
			if !errors.Is(err, os.ErrExist) {
				return err
			}
		}
	}

	return nil
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
		path := p
		path = Clean(path)
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

// Оборачивание двоичной записи
func BinaryWrite(w io.Writer, data any) error {
	return binary.Write(w, binary.LittleEndian, data)
}

// Оборачивание двоичного чтения
func BinaryRead(r io.Reader, data any) error {
	return binary.Read(r, binary.LittleEndian, data)
}
