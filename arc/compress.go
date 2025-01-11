package arc

import (
	"archiver/arc/header"
	c "archiver/compressor"
	"archiver/errtype"
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
				return errtype.ErrCompress("не могу получить директории", err)
			}
			continue
		}

		if h, err := fetchFile(path); err == nil { // Добавалние файла в заголовок
			headers = append(headers, h)
		} else {
			return errtype.ErrCompress("не могу получить директории", err)
		}
	}

	dirs, files := arc.sortHeaders(headers)

	closeRemove := func(arcFile io.Closer) {
		if err := arcFile.Close(); err != nil {
			errtype.ErrCompress("ошибка закрытия файла архива", err)
		}
		arc.RemoveTmp()
	}

	arcFile, err := arc.writeHeaderDirs(dirs)
	if err != nil {
		closeRemove(arcFile)
		return errtype.ErrCompress(
			"не могу записать заголовки директории", err,
		)
	}
	defer arcFile.Close()

	for _, fi := range files {
		if err = fi.Write(arcFile); err != nil {
			closeRemove(arcFile)
			return errtype.ErrCompress("ошибка записи заголовка файла", err)
		}
		if err = arc.compressFile(fi.Path(), arcFile); err != nil {
			closeRemove(arcFile)
			return errtype.ErrCompress("не могу сжать файл", err)
		}
	}

	return nil
}

// Проверяет, содержит ли срез уникалные значения
// Если нет, то удаляет дубликаты. Сортирует пути.
// Разделяет заголовки на директории и файлы
func (Arc) sortHeaders(headers []header.Header) ([]*header.DirItem, []*header.FileItem) {
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
	return dirs, files
}

// Сжимает файл блоками
func (arc *Arc) compressFile(inPath string, arcFile io.Writer) error {
	inFile, err := os.Open(inPath)
	if err != nil {
		return errtype.ErrCompress(
			fmt.Sprintf("не могу открыть входной файл '%s' для сжатия", inPath),
			err,
		)
	}
	defer inFile.Close()

	var (
		wrote, read int64
		crc         uint32
	)

	for {
		if read, err = arc.loadUncompressedBuf(inFile); err != nil {
			return errtype.ErrCompress("ошибка чтения не сжатых блоков", err)
		}

		if err = arc.compressBuffers(); err != nil {
			return errtype.ErrCompress("ошибка сжатия буфферов", err)
		}

		for i := 0; i < ncpu && compressedBuf[i].Len() > 0; i++ {
			// Пишем длину сжатого блока
			if err = binary.Write(writeBuf, binary.LittleEndian, int64(compressedBuf[i].Len())); err != nil {
				return errtype.ErrCompress("ошибка записи длины блока", err)
			}

			crc ^= crc32.Checksum(compressedBuf[i].Bytes(), crct)

			// Пишем сжатый блок
			if _, err = writeBuf.ReadFrom(compressedBuf[i]); err != nil {
				return errtype.ErrCompress("ошибка чтения из буфера сжатых данных", err)
			}
		}

		if (writeBuf.Len() >= int(c.BufferSize)) || read == 0 {
			if wrote, err = writeBuf.WriteTo(arcFile); err != nil {
				return errtype.ErrCompress("ошибка записи буфера в файл архива", err)
			}
			log.Println("Записан сжатый буфер:", wrote)
		}

		if read == 0 {
			break
		}
	}

	// Пишем признак конца файла
	if err = binary.Write(arcFile, binary.LittleEndian, int64(-1)); err != nil {
		return errtype.ErrCompress("ошибка записи EOF", err)
	}
	log.Println("Записан EOF")

	// Пишем контрольную сумму
	if err = binary.Write(arcFile, binary.LittleEndian, crc); err != nil {
		return errtype.ErrCompress("ошибка записи CRC", err)
	}
	log.Printf("Записан CRC: %X\n", crc)

	fmt.Println(inPath)

	return nil
}

// Загружает данные в буферы несжатых данных
func (Arc) loadUncompressedBuf(r io.Reader) (read int64, err error) {
	var n int64

	for i := 0; i < ncpu; i++ {
		lim := io.LimitReader(r, c.BufferSize)
		if n, err = decompressedBuf[i].ReadFrom(lim); err != nil {
			return 0, errtype.ErrCompress("ошибка чтения в несжатый буфер", err)
		}

		read += n
	}

	return read, nil
}

// Сжимает данные в буферах несжатых данных
func (arc Arc) compressBuffers() error {
	var (
		errChan = make(chan error, ncpu)
		wg      sync.WaitGroup
	)

	for i := 0; i < ncpu && decompressedBuf[i].Len() > 0; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			compressor, err := c.NewWriter(arc.ct, compressedBuf[i], c.Level(-1))
			if err != nil {
				errChan <- errtype.ErrCompress("ошибка иницализации компрессора", err)
				return
			}
			_, err = decompressedBuf[i].WriteTo(compressor)
			if err != nil {
				errChan <- errtype.ErrCompress("ошибка записи в компрессор", err)
				return
			}
			if err = compressor.Close(); err != nil {
				errChan <- errtype.ErrCompress("ошибка закрытия компрессора", err)
			}
		}(i)
	}

	go func() {
		wg.Wait()
		close(errChan)
	}()

	for err := range errChan {
		return err
	}
	return nil
}
