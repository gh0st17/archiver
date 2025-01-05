package arc

import (
	"archiver/arc/header"
	c "archiver/compressor"
	"archiver/filesystem"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"log"
	"os"
	"sort"
	"sync"
	"sync/atomic"
)

// Создает архив
func (arc Arc) Compress(paths []string) (err error) {
	var headers []header.Header

	for _, path := range paths { // Получение списка файлов и директории
		// Добавление директории в заголовок
		// и ее рекурсивный обход
		if filesystem.DirExists(path) {
			if dirHeaders, err := fetchDir(path); err == nil {
				headers = append(headers, dirHeaders...)
			} else {
				return fmt.Errorf("compress: %v", err)
			}
			continue
		}

		if h, err := fetchFile(path); err == nil { // Добавалние файла в заголовок
			headers = append(headers, h)
		} else {
			return fmt.Errorf("compress: %v", err)
		}
	}

	// Проверяет, содержит ли срез уникалные значения
	// Если нет, то удаляет дубликаты
	dropDup := func(headers *[]header.Header) {
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

	dropDup(&headers)
	sort.Sort(header.ByPath(headers))

	return arc.compressHeaders(headers)
}

// Сжимает данные указанные в заголовках во временный файл
func (arc Arc) compressHeaders(headers []header.Header) error {
	closeRemove := func(tmpFile *os.File) {
		tmpFile.Close()
		os.Remove(tmpPath)
	}

	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		closeRemove(tmpFile)
		return fmt.Errorf("compressHeaders: %v", err)
	}

	for _, h := range headers {
		if _, ok := h.(*header.DirItem); ok {
			continue
		}

		if err := arc.compressFile(h.(*header.FileItem), tmpFile); err != nil {
			closeRemove(tmpFile)
			return fmt.Errorf("compressHeaders: %v", err)
		}
	}

	tmpFile.Close()
	if err = arc.writeItems(headers); err != nil {
		closeRemove(tmpFile)
		return fmt.Errorf("compressHeaders: can't write items: %v", err)
	}

	return os.Remove(tmpPath)
}

// Сжимает файл блоками
func (arc *Arc) compressFile(fi *header.FileItem, tmpFile io.Writer) (err error) {
	inFile, err := os.Open(fi.Filepath)
	if err != nil {
		return fmt.Errorf("compressFile: %v", err)
	}
	defer inFile.Close()

	var (
		totalRead  header.Size
		maxCompLen atomic.Int64
	)

	for i := range uncompressedBuf {
		uncompressedBuf[i] = make([]byte, c.BufferSize)
	}

	for totalRead < fi.UncompressedSize {
		remaining := int(fi.UncompressedSize - totalRead)
		if n, err := arc.loadUncompressedBuf(inFile, remaining); err != nil {
			return fmt.Errorf("compressFile: can't load buf: %v", err)
		} else {
			totalRead += header.Size(n)
		}

		if err = arc.compressBuffers(&maxCompLen); err != nil {
			return fmt.Errorf("compressFile: can't compress buf: %v", err)
		}

		for i := range compressedBuf {
			if len(uncompressedBuf[i]) == 0 || len(compressedBuf[i]) == 0 {
				break
			}

			err = binary.Write(tmpFile, binary.LittleEndian, int64(len(compressedBuf[i])))
			if err != nil {
				return err
			}
			log.Println("Written block length:", len(compressedBuf[i]))

			fi.CRC ^= crc32.Checksum(compressedBuf[i], crct)
			fi.CompressedSize += header.Size(len(compressedBuf[i]))

			if _, err = tmpFile.Write(compressedBuf[i]); err != nil {
				return fmt.Errorf("compressFile: can't write '%s' %v", tmpPath, err)
			}
		}
	}

	if arc.maxCompLen < maxCompLen.Load() {
		arc.maxCompLen = maxCompLen.Load()
		log.Println("Max comp len now is:", arc.maxCompLen)
	}

	fmt.Println(fi.Filepath)

	return nil
}

// Загружает данные в буферы несжатых данных
func (Arc) loadUncompressedBuf(r io.Reader, remaining int) (int, error) {
	var read int

	for i := range uncompressedBuf {
		if remaining == 0 {
			uncompressedBuf[i] = uncompressedBuf[i][:0]
			log.Println("uncompressedBuf", i, "not needed")
			continue
		}

		if int64(remaining) < c.BufferSize {
			uncompressedBuf[i] = uncompressedBuf[i][:remaining]
		}

		if n, err := io.ReadFull(r, uncompressedBuf[i]); err != nil {
			return 0, err
		} else {
			read += n
			remaining -= n
		}
	}

	return read, nil
}

// Сжимает данные в буферах несжатых данных
func (arc Arc) compressBuffers(maxCompLen *atomic.Int64) error {
	var (
		wg      sync.WaitGroup
		errChan = make(chan error, ncpu)
	)

	for i := range uncompressedBuf {
		if len(uncompressedBuf[i]) == 0 {
			log.Println("uncompressedBuf", i, "breaking foo-loop with zero len")
			break
		}

		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			var err error

			compressedBuf[i], err = c.Compress(uncompressedBuf[i], arc.Compressor)
			if err != nil {
				errChan <- err
			}

			if len(compressedBuf[i]) > int(maxCompLen.Load()) {
				maxCompLen.Store(int64(len(compressedBuf[i])))
			}
		}(i)
	}

	go func() {
		wg.Wait()
		close(errChan)
	}()

	for err := range errChan {
		return fmt.Errorf("compressBuf: %v", err)
	}
	return nil
}
