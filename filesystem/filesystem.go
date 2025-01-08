package filesystem

import (
	"os"
	"path/filepath"
	"strings"
)

// Разбивает путь на компоненты
func splitPath(path string) []string {
	if path == "/" {
		return []string{path}
	}

	dir, last := filepath.Split(path)
	if dir == "" {
		return []string{last}
	}
	return append(splitPath(filepath.Clean(dir)), last)
}

// Создает директории на всем пути `path`
func CreatePath(path string) (err error) {
	var (
		splitedPath = splitPath(path)
		fullPath    string
	)

	for _, pathPart := range splitedPath {
		fullPath = filepath.Join(fullPath, pathPart)

		if !DirExists(fullPath) {
			if err = os.Mkdir(fullPath, 0755); err != nil {
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
