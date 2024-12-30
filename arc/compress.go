package arc

import (
	"archiver/arc/header"
	c "archiver/compressor"
	"archiver/filesystem"
	"bufio"
	"bytes"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"runtime"
	"sort"
	"sync"
)

// Создает архив
func Compress(arc *Arc, paths []string) (err error) {
	var (
		headers    []header.Header
		filesCount int
		info       os.FileInfo
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
			filesCount++
		} else {
			return err
		}
	}

	dropDup(&headers)
	sort.Sort(header.ByPath(headers))

	return compressHeaders(filesCount, headers, arc)
}

// Сжимает данные в заголовках в архив
func compressHeaders(filesCount int, headers []header.Header, arc *Arc) error {
	var (
		sem     = make(chan struct{}, runtime.NumCPU())
		errChan = make(chan error, filesCount)
		wg      sync.WaitGroup
	)

	for i, h := range headers {
		if h.Type() == header.Directory {
			continue
		}

		wg.Add(1)
		go func(h header.Header) { // Горутина для параллельного сжатия
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			if err := compressFile(h, arc.Compressor); err != nil {
				errChan <- err
			}
		}(headers[i])
	}

	go func() {
		wg.Wait()
		close(errChan)
	}()

	for err := range errChan { // Обработка ошибки из горутины
		fmt.Printf("compress: %v", err)
	}

	writeItems(arc, headers)

	return nil
}

// Сжимает файл
func compressFile(h header.Header, c c.Compressor) error {
	fmt.Println(h.Path())

	f, err := os.Open(h.Path())
	if err != nil {
		return err
	}
	defer f.Close()

	var buf bytes.Buffer
	cw, err := c.NewWriter(&buf)
	if err != nil {
		return err
	}

	if _, err = io.Copy(cw, bufio.NewReader(f)); err != nil {
		return err
	}

	if err := cw.Close(); err != nil {
		return err
	}

	crct := crc32.MakeTable(crc32.Koopman)
	h.(*header.FileItem).CRC = crc32.Checksum(buf.Bytes(), crct)
	h.(*header.FileItem).CompressedSize = header.Size(len(buf.Bytes()))
	h.(*header.FileItem).Data = buf.Bytes()

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
