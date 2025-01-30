package compress

import (
	"archiver/arc/internal/compress/platform"
	"archiver/arc/internal/header"
	"archiver/errtype"
	"archiver/filesystem"
	"errors"
	"fmt"
	"os"
	fp "path/filepath"
	"syscall"
)

// Проверяет чем является path, директорией,
// символьной ссылкой или файлом, возвращает
// интерфейс заголовка, указывающий на
// соответствующий тип
func fetchPath(path string) (h header.Header, err error) {
	if len(path) > 1023 {
		return nil, ErrLongPath(path)
	}

	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	atime, mtime := platform.Timestamp(info)

	b, err := header.NewBase(fp.ToSlash(path), atime, mtime)
	if err != nil {
		return nil, err
	}

	if info.Mode()&os.ModeSymlink != 0 {
		target, err := fp.EvalSymlinks(path)
		if errors.Is(err, syscall.ENOENT) {
			fmt.Printf("Символическая ссылка '%s' испорчена\n", path)
			return nil, nil
		} else if err != nil {
			return nil, err
		}

		if target, err = fp.Abs(target); err != nil {
			return nil, err
		} else {
			h = header.NewSymItem(path, target)
		}
	} else if info.Mode()&os.ModeDir != 0 {
		h = header.NewDirItem(b.PathInArc())
	} else {
		h = header.NewFileItem(b, header.Size(info.Size()))
	}

	return h, nil
}

// Рекурсивно собирает элементы в директории
func fetchDir(path string) (headers []header.Header, err error) {
	err = fp.WalkDir(path, func(path string, _ os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		header, err := fetchPath(path)
		if err != nil {
			if err == ErrLongPath(path) {
				fmt.Println(err)
				return nil
			} else {
				return err
			}
		}

		if header != nil {
			headers = append(headers, header)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return headers, nil
}

// Собирает элементы файловой системы в заголовки
func fetchHeaders(paths []string) (headers []header.Header, err error) {
	var (
		dirHeaders []header.Header
		header     header.Header
	)

	for _, path := range paths { // Получение списка файлов и директории
		// Добавление директории в заголовок
		// и ее рекурсивный обход
		if filesystem.DirExists(path) {
			if dirHeaders, err = fetchDir(path); err == nil {
				headers = append(headers, dirHeaders...)
			} else {
				return nil, errtype.Join(ErrFetchDirs, err)
			}
			continue
		}

		if header, err = fetchPath(path); err != nil { // Добавалние файла в заголовок
			return nil, errtype.Join(ErrFetchDirs, err)
		} else if header != nil {
			headers = append(headers, header)
		}
	}
	return headers, nil
}
