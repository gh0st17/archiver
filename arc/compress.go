package arc

import (
	"archiver/arc/header"
	c "archiver/compressor"
	"archiver/filesystem"
	"bufio"
	"bytes"
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
				return fmt.Errorf("compress: can't fetch dir: %v", err)
			}
			continue
		}

		if h, err := fetchFile(path); err == nil { // Добавалние файла в заголовок
			headers = append(headers, h)
		} else {
			return fmt.Errorf("compress: can't fetch file: %v", err)
		}
	}

	// Проверяет, содержит ли срез уникалные значения
	// Если нет, то удаляет дубликаты
	seen := make(map[string]struct{})
	uniqueHeaders := make([]header.Header, 0, len(headers))
	for _, h := range headers {
		if _, exists := seen[h.Path()]; !exists {
			seen[h.Path()] = struct{}{}
			uniqueHeaders = append(uniqueHeaders, h)
		}
	}
	headers = uniqueHeaders

	sort.Sort(header.ByPath(headers))

	var dirs []*header.DirItem
	var files []*header.FileItem
	for _, h := range headers {
		if d, ok := h.(*header.DirItem); ok {
			dirs = append(dirs, d)
		} else {
			files = append(files, h.(*header.FileItem))
		}
	}

	closeRemove := func(arcFile io.Closer) {
		arcFile.Close()
		arc.RemoveTmp()
	}

	arcFile, err := arc.writeHeaderDirs(dirs)
	if err != nil {
		closeRemove(arcFile)
		return fmt.Errorf("compress: %v", err)
	}
	defer arcFile.Close()

	for _, fi := range files {
		fi.Write(arcFile)
		if err = arc.compressFile(fi, arcFile); err != nil {
			closeRemove(arcFile)
			return fmt.Errorf("compressHeaders: %v", err)
		}
	}

	arcFile.Seek(3, io.SeekStart)
	err = binary.Write(arcFile, binary.LittleEndian, arc.maxCompLen)
	if err != nil {
		return err
	}

	return nil
}

// Сжимает файл блоками
func (arc *Arc) compressFile(fi *header.FileItem, arcFile io.Writer) (err error) {
	inFile, err := os.Open(fi.Filepath)
	if err != nil {
		return fmt.Errorf("compressFile: %v", err)
	}
	defer inFile.Close()

	var (
		totalRead  header.Size
		maxCompLen atomic.Int64
		inBuf      = bufio.NewReader(inFile)
	)

	if cap(uncompressedBuf[0]) == 0 {
		for i := range uncompressedBuf {
			uncompressedBuf[i] = make([]byte, c.BufferSize)
		}
	}

	for totalRead < fi.UncompressedSize {
		remaining := int(fi.UncompressedSize - totalRead)
		if n, err := arc.loadUncompressedBuf(inBuf, remaining); err != nil {
			return fmt.Errorf("compressFile: can't load buf: %v", err)
		} else {
			totalRead += header.Size(n)
		}

		if err = arc.compressBuffers(&maxCompLen); err != nil {
			return fmt.Errorf("compressFile: can't compress buf: %v", err)
		}

		for i := 0; i < ncpu && len(uncompressedBuf[i]) > 0 && len(compressedBuf[i]) > 0; i++ {
			err = binary.Write(arcFile, binary.LittleEndian, int64(len(compressedBuf[i])))
			if err != nil {
				return fmt.Errorf("compressFile: can't binary write %v", err)
			}

			fi.CRC ^= crc32.Checksum(compressedBuf[i], crct)

			if _, err = arcFile.Write(compressedBuf[i]); err != nil {
				return fmt.Errorf("compressFile: can't write '%s' %v", arc.ArchivePath, err)
			}
			log.Println("Written compressed data:", len(compressedBuf[i]))
		}
	}

	// Пишем признак конца файла
	err = binary.Write(arcFile, binary.LittleEndian, int64(-1))
	if err != nil {
		return fmt.Errorf("compressFile: can't binary write EOF: %v", err)
	}
	log.Println("Written EOF")
	// Пишем контрольную сумму
	if err = binary.Write(arcFile, binary.LittleEndian, fi.CRC); err != nil {
		return err
	}
	log.Printf("Written CRC: %X\n", fi.CRC)

	if arc.maxCompLen < maxCompLen.Load() {
		arc.maxCompLen = maxCompLen.Load()
		log.Println("Max comp len now is:", arc.maxCompLen)
	}

	for i := range uncompressedBuf {
		uncompressedBuf[i] = uncompressedBuf[i][:cap(uncompressedBuf[i])]
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
			log.Println("uncompressedBuf", i, "doesn't need, skipping")
			continue
		}

		if int64(remaining) < c.BufferSize {
			uncompressedBuf[i] = uncompressedBuf[i][:remaining]
		}

		if n, err := io.ReadFull(r, uncompressedBuf[i]); err != nil {
			return 0, fmt.Errorf("loadUncompressedBuf: can't read: %v", err)
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

	for i := 0; i < ncpu && len(uncompressedBuf[i]) > 0; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			buf := bytes.NewBuffer(nil)
			compressor := c.NewWriter(arc.CompType, buf, c.Level(-1))
			_, err := compressor.Write(uncompressedBuf[i])
			if err != nil {
				errChan <- err
			}
			compressedBuf[i] = buf.Bytes()

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
