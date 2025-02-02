// Пакет compress предоставляет функции для сжатия файлов
//
// Основные функции:
//   - PrepareHeaders: Подготавливает заголовки для сжатия
//   - ProcessingHeaders: Обработка заголовков
package compress

import (
	"archiver/arc/internal/generic"
	"archiver/arc/internal/header"
	"archiver/errtype"
	"archiver/filesystem"
	"bufio"
	"fmt"
	"hash/crc32"
	"io"
	"log"
	"os"
	"sync"
)

// Подготавливает заголовки для сжатия
func PrepareHeaders(paths []string) (headers []header.Header, err error) {
	// Печать предупреждения о наличии абсолютных путей
	filesystem.PrintPathsCheck(paths)

	// Собираем элементы по путям paths в заголовки
	if headers, err = fetchHeaders(paths); err != nil {
		return nil, err
	}
	headers = header.DropDups(headers) // Удаляем дубликаты

	if len(headers) == 0 { // Если true, то сжимать нечего
		return nil, ErrNoEntries
	}
	return headers, nil
}

// Обработка заголовков
func ProcessingHeaders(arcFile io.WriteCloser, headers []header.Header, verbose bool) error {
	arcBuf := bufio.NewWriter(arcFile)
	for _, h := range headers { // Перебираем заголовки
		if fi, ok := h.(*header.FileItem); ok {
			if err := processingFile(fi, arcBuf, verbose); err != nil {
				return err
			}
		} else if di, ok := h.(*header.DirItem); ok {
			processingDir(di, verbose)
		} else if si, ok := h.(*header.SymItem); ok {
			processingSym(si, arcBuf, verbose)
		}
	}
	arcBuf.Flush()
	return nil
}

// Обрабатывает заголовок файла
func processingFile(fi *header.FileItem, arcBuf io.Writer, verbose bool) error {
	err := fi.Write(arcBuf)
	if err != nil {
		return errtype.Join(ErrWriteFileHeader, err)
	}

	if err = compressFile(fi, arcBuf, verbose); err != nil {
		return errtype.Join(ErrCompressFile, err)
	}
	return nil
}

// Обрабатывает заголовок директории
func processingDir(di *header.DirItem, verbose bool) {
	if verbose {
		fmt.Println(di.PathInArc())
	}
}

// Обрабатывает заголовок символьной ссылки
func processingSym(si *header.SymItem, arcBuf io.Writer, verbose bool) error {
	if err := si.Write(arcBuf); err != nil {
		return errtype.Join(ErrWriteSymHeader, err)
	}
	if verbose {
		fmt.Println(si.PathInArc(), "->", si.PathOnDisk())
	}
	return nil
}

// Сжимает файл блоками
func compressFile(fi header.PathProvider, arcBuf io.Writer, verbose bool) error {
	inFile, err := os.Open(fi.PathOnDisk())
	if err != nil {
		return errtype.Join(ErrOpenFileCompress(fi.PathOnDisk()), err)
	}
	defer inFile.Close()
	inBuf := bufio.NewReader(inFile)

	var (
		crct           = generic.CRCTable()
		ncpu           = generic.Ncpu()
		compressedBufs = generic.CompBuffers()
		compressors    = generic.Compressors()
		writeBuf       = generic.WriteBuffer()

		writeBufSize = (generic.BufferSize * ncpu) << 1
		wrote, read  int64
		crc          uint32
		wg           = sync.WaitGroup{}
	)

	for {
		// Заполняем буферы несжатыми частями (блоками) файла
		if read, err = loadUncompressedBuf(inBuf); err != nil {
			return errtype.Join(ErrReadUncompressed, err)
		}

		wg.Wait()

		if read == 0 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				generic.FlushWriteBuffer(arcBuf)
			}()
			break
		}

		// Сжимаем буферы
		if err = compressBuffers(); err != nil {
			return errtype.Join(ErrCompress, err)
		}

		for i := 0; i < ncpu && compressedBufs[i].Len() > 0; i++ {
			// Пишем длину сжатого блока
			length := int64(compressedBufs[i].Len())
			if err = filesystem.BinaryWrite(writeBuf, length); err != nil {
				return errtype.Join(ErrWriteBufLen, err)
			}

			crc ^= crc32.Checksum(compressedBufs[i].Bytes(), crct)

			// Пишем сжатый блок
			if wrote, err = compressedBufs[i].WriteTo(writeBuf); err != nil {
				return errtype.Join(ErrWriteCompressBuf, err)
			}
			log.Println("В буфер записи записан блок размера:", wrote)
			compressors[i].Reset(compressedBufs[i])

			if writeBuf.Len() >= int(writeBufSize) {
				wg.Add(1)
				go func() {
					defer wg.Done()
					generic.FlushWriteBuffer(arcBuf)
				}()
				if i+1 != ncpu {
					wg.Wait()
				}
			}
		}
	}

	wg.Wait()

	// Пишем признак конца файла
	if err = filesystem.BinaryWrite(arcBuf, int64(-1)); err != nil {
		return errtype.Join(ErrWriteEOF, err)
	}
	log.Println("Записан EOF")

	// Пишем контрольную сумму
	if err = filesystem.BinaryWrite(arcBuf, crc); err != nil {
		return errtype.Join(ErrWriteCRC, err)
	}
	log.Printf("Записан CRC: %X\n", crc)

	if verbose {
		fmt.Println(fi.PathInArc())
	}

	return nil
}

// Загружает данные в буферы несжатых данных
func loadUncompressedBuf(inBuf io.Reader) (read int64, err error) {
	var (
		n                int64
		ncpu             = generic.Ncpu()
		decompressedBufs = generic.DecompBuffers()
		bufferSize       = int64(generic.BufferSize)
	)

	for i := 0; i < ncpu && err != io.EOF; i++ {
		n, err = io.CopyN(decompressedBufs[i], inBuf, bufferSize)
		if err != nil && err != io.EOF {
			return 0, errtype.Join(ErrReadUncompressBuf, err)
		}

		read += n
	}

	return read, nil
}

// Сжимает данные в буферах несжатых данных
func compressBuffers() error {
	var (
		ncpu             = generic.Ncpu()
		decompressedBufs = generic.DecompBuffers()
		compressors      = generic.Compressors()

		errChan = make(chan error, ncpu)
		wg      sync.WaitGroup
	)

	for i := 0; i < ncpu && decompressedBufs[i].Len() > 0; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			_, err := decompressedBufs[i].WriteTo(compressors[i])
			if err != nil {
				errChan <- errtype.Join(ErrWriteCompressor, err)
				return
			}
			if err = compressors[i].Close(); err != nil {
				errChan <- errtype.Join(ErrCloseCompressor, err)
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
