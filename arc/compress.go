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
)

// Создает архив
func (arc Arc) Compress(paths []string) error {
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

	return nil
}

// Сжимает файл блоками
func (arc *Arc) compressFile(fi *header.FileItem, arcFile io.Writer) error {
	inFile, err := os.Open(fi.Path())
	if err != nil {
		return fmt.Errorf("compressFile: %v", err)
	}
	defer inFile.Close()

	var (
		totalRead header.Size
		n         int64
		crc       uint32
	)

	for {
		if n, err = arc.loadUncompressedBuf(inFile); err != nil {
			if err != io.EOF && err != io.ErrUnexpectedEOF {
				return fmt.Errorf("compressFile: can't load buf: %v", err)
			}
		}

		if err = arc.compressBuffers(); err != nil {
			return fmt.Errorf("compressFile: can't compress buf: %v", err)
		}

		for i := 0; i < ncpu && compressedBuf[i].Len() > 0; i++ {
			err = binary.Write(arcFile, binary.LittleEndian, int64(compressedBuf[i].Len()))
			if err != nil {
				return fmt.Errorf("compressFile: can't binary write %v", err)
			}

			crc ^= crc32.Checksum(compressedBuf[i].Bytes(), crct)

			if _, err = compressedBuf[i].WriteTo(arcFile); err != nil {
				return fmt.Errorf("compressFile: can't write '%s' %v", arc.ArchivePath, err)
			}
			log.Println("Written compressed data:", n)
		}

		if n == 0 {
			break
		}

		totalRead += header.Size(n)
	}
	fi.SetCRC(crc)

	// Пишем признак конца файла
	err = binary.Write(arcFile, binary.LittleEndian, int64(-1))
	if err != nil {
		return fmt.Errorf("compressFile: can't binary write EOF: %v", err)
	}
	log.Println("Written EOF")

	// Пишем контрольную сумму
	if err = binary.Write(arcFile, binary.LittleEndian, fi.CRC()); err != nil {
		return err
	}
	log.Printf("Written CRC: %X\n", fi.CRC())

	fmt.Println(fi.Path())

	return nil
}

// Загружает данные в буферы несжатых данных
func (Arc) loadUncompressedBuf(r io.Reader) (read int64, err error) {
	var n int64

	for i := 0; i < ncpu; i++ {
		lim := io.LimitReader(r, c.BufferSize)
		if n, err = decompressedBuf[i].ReadFrom(lim); err != nil {
			return 0, fmt.Errorf("loadUncompressedBuf: can't read: %v", err)
		}

		read += n
		if err == io.EOF {
			break
		}
	}

	return read, nil
}

// Сжимает данные в буферах несжатых данных
func (arc Arc) compressBuffers() error {
	var (
		wg      sync.WaitGroup
		errChan = make(chan error, ncpu)
	)

	for i := 0; i < ncpu && decompressedBuf[i].Len() > 0; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			compressor := c.NewWriter(arc.CompType, compressedBuf[i], c.Level(-1))
			_, err := decompressedBuf[i].WriteTo(compressor)
			if err != nil {
				errChan <- err
			}
			if err = compressor.Close(); err != nil {
				errChan <- err
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
