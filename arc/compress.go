package arc

import (
	"archiver/arc/header"
	c "archiver/compressor"
	"archiver/filesystem"
	"bufio"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"sort"
)

// Создает архив
func (arc Arc) Compress(paths []string) (err error) {
	var (
		headers []header.Header
		info    os.FileInfo
	)

	for _, path := range paths { // Получение списка файлов и директории
		// Добавление директории в заголовок
		// и ее рекурсивный обход
		if filesystem.DirExists(path) {
			if dirHeaders, err := fetchDir(path); err == nil {
				headers = append(headers, dirHeaders...)
			} else {
				return err
			}
			continue
		}

		info, err = os.Stat(path)
		if err != nil {
			return err
		}

		if h, err := fetchFile(path, info); err == nil { // Добавалние файла в заголовок
			headers = append(headers, h)
		} else {
			return err
		}
	}

	dropDup(&headers)
	sort.Sort(header.ByPath(headers))

	return arc.compressHeaders(headers)
}

// Сжимает данные в заголовках в архив
func (arc Arc) compressHeaders(headers []header.Header) error {
	tmpFile, err := os.Create("arctmp")
	if err != nil {
		return err
	}
	defer tmpFile.Close()

	for _, h := range headers {
		if _, ok := h.(*header.DirItem); ok {
			continue
		}

		if err := arc.compressFile(h.(*header.FileItem), tmpFile); err != nil {
			return err
		}
	}

	arc.writeItems(headers)

	return nil
}

// Сжимает файл
func (arc Arc) compressFile(fi *header.FileItem, tmpFile *os.File) (err error) {
	fmt.Println(fi.Filepath)

	f, err := os.Open(fi.Filepath)
	if err != nil {
		return err
	}
	defer f.Close()

	var compData []byte
	crct := crc32.MakeTable(crc32.Koopman)

	for {
		compData, err = c.CompressBlock(f, arc.Compressor)

		fi.CRC ^= crc32.Checksum(compData, crct)
		fi.CompressedSize += header.Size(len(compData))

		buf := bufio.NewWriter(tmpFile)
		buf.Write(compData)

		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}

			return err
		}
	}

	return nil
}

// Проверяет, содержит ли срез уникалные значения
// Если нет, то удаляет дубликаты
func dropDup(headers *[]header.Header) {
	seen := make(map[string]struct{})
	uniqueHeaders := make([]header.Header, 0, len(*headers))

	for _, h := range *headers {
		if _, exists := seen[h.Path()]; !exists {
			seen[h.Path()] = struct{}{}
			uniqueHeaders = append(uniqueHeaders, h)
		}
	}

	*headers = uniqueHeaders
}
