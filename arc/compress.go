package arc

import (
	"archiver/arc/header"
	c "archiver/compressor"
	"archiver/filesystem"
	"bufio"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"log"
	"os"
	"runtime"
	"sort"
	"sync"
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

// Сжимает данные указанные в заголовках во временный файл
func (arc Arc) compressHeaders(headers []header.Header) error {
	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		return err
	}

	for _, h := range headers {
		if _, ok := h.(*header.DirItem); ok {
			continue
		}

		if err := arc.compressFile(h.(*header.FileItem), tmpFile); err != nil {
			return err
		}
	}

	tmpFile.Close()
	err = arc.writeItems(headers)
	if err != nil {
		return err
	}

	return os.Remove(tmpPath)
}

// Сжимает файл блоками
func (arc Arc) compressFile(fi *header.FileItem, tmpFile io.Writer) (err error) {
	fmt.Println(fi.Filepath)

	inFile, err := os.Open(fi.Filepath)
	if err != nil {
		return err
	}
	defer inFile.Close()

	var (
		totalRead   header.Size
		crct        = crc32.MakeTable(crc32.Koopman)
		ncpu        = runtime.NumCPU()
		unCompBytes = make([][]byte, ncpu)
		compBytes   = make([][]byte, ncpu)
		wg          sync.WaitGroup
	)

	for i := range unCompBytes {
		unCompBytes[i] = make([]byte, c.GetBufferSize())
		compBytes[i] = make([]byte, c.GetBufferSize())
	}

	clearBuffers := func(buffers [][]byte) {
		for i := range buffers {
			buffers[i] = buffers[i][:cap(buffers[i])]
			for j := range buffers[i] {
				buffers[i][j] = 0
			}
		}
	}

	for totalRead < fi.UncompressedSize {
		n, err := fillBlocks(unCompBytes, inFile, int(fi.UncompressedSize-totalRead))
		if err != nil {
			return err
		}
		totalRead += header.Size(n)

		errChan := make(chan error, ncpu)
		for i, unComp := range unCompBytes {
			if len(unComp) == 0 {
				log.Println("unComp", i, "breaking foo-loop with zero len")
				break
			}

			wg.Add(1)
			go func(i int) {
				defer wg.Done()

				compBytes[i], err = c.CompressBlock(unCompBytes[i], arc.Compressor)
				if err != nil {
					errChan <- err
				}
			}(i)
		}

		go func() {
			wg.Wait()
			close(errChan)
		}()

		for err := range errChan {
			return fmt.Errorf("arc: compress: %v", err)
		}

		for i := range compBytes {
			if len(unCompBytes[i]) == 0 || len(compBytes[i]) == 0 {
				break
			}

			err = binary.Write(tmpFile, binary.LittleEndian, int64(len(compBytes[i])))
			if err != nil {
				return err
			}
			log.Println("Written block length:", len(compBytes[i]))

			fi.CRC ^= crc32.Checksum(compBytes[i], crct)
			fi.CompressedSize += header.Size(len(compBytes[i]))

			_, err = tmpFile.Write(compBytes[i])
			if err != nil {
				return fmt.Errorf("arc: tmpFile.Write: %v", err)
			}
		}

		clearBuffers(compBytes)
		clearBuffers(unCompBytes)
	}

	return nil
}

func fillBlocks(blocks [][]byte, r io.Reader, remaining int) (int, error) {
	var read int
	buf := bufio.NewReader(r)

	for i := range blocks {
		if remaining == 0 {
			blocks[i] = blocks[i][:0]
			log.Println("block", i, "now have len:", len(blocks[i]))
			continue
		}

		if int64(remaining) < c.GetBufferSize() {
			blocks[i] = blocks[i][:remaining]
		}

		if n, err := io.ReadFull(buf, blocks[i]); err != nil {
			return 0, err
		} else {
			read += n
			remaining -= n
		}
	}

	return read, nil
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
