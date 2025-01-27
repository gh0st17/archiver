package arc

import (
	"archiver/arc/header"
	c "archiver/compressor"
	"archiver/errtype"
	"archiver/filesystem"
	"bufio"
	"bytes"
	"errors"
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
	var (
		headers []header.Header
		arcFile io.WriteCloser
		err     error
	)

	filesystem.PrintPathsCheck(paths)
	if headers, err = arc.fetchHeaders(paths); err != nil {
		return errtype.ErrCompress(err)
	}
	headers = header.DropDups(headers)
	sort.Sort(header.ByPathInArc(headers))

	if len(headers) == 0 {
		return errtype.ErrCompress(ErrNoEntries)
	}

	arcFile, err = arc.writeArcHeader()
	if err != nil {
		return errtype.ErrCompress(
			errtype.Join(ErrWriteArcHeaders, err),
		)
	}

	for i := 0; i < ncpu; i++ {
		compressor[i], err = c.NewWriter(arc.ct, compressedBuf[i], arc.cl)
		if err != nil {
			return errtype.ErrCompress(
				errtype.Join(ErrCompressorInit, err),
			)
		}
	}

	writeBufSize = (int(bufferSize) * ncpu) << 1
	writeBuf = bytes.NewBuffer(make([]byte, 0, writeBufSize))

	arcBuf := bufio.NewWriter(arcFile)
	for _, h := range headers {
		if fi, ok := h.(*header.FileItem); ok {
			if err = arc.processingFile(fi, arcBuf); err != nil {
				arc.closeRemove(arcFile)
				return errtype.ErrCompress(err)
			}
		} else if di, ok := h.(*header.DirItem); ok {
			arc.processingDir(di)
		} else if si, ok := h.(*header.SymItem); ok {
			arc.processingSym(si, arcBuf)
		}
	}
	arcBuf.Flush()
	arcFile.Close()

	return nil
}

// Обрабатывает заголовок файла
func (arc Arc) processingFile(fi *header.FileItem, arcBuf io.Writer) error {
	err := fi.Write(arcBuf)
	if err != nil {
		return errtype.Join(ErrWriteFileHeader, err)
	}

	if err = arc.compressFile(fi, arcBuf); err != nil {
		return errtype.Join(ErrCompressFile, err)
	}
	return nil
}

// Обрабатывает заголовок директории
func (arc Arc) processingDir(di *header.DirItem) error {
	fmt.Println(di.PathInArc())
	return nil
}

// Обрабатывает заголовок символьной ссылки
func (arc Arc) processingSym(si *header.SymItem, arcBuf io.Writer) error {
	si.Write(arcBuf)
	fmt.Println(si.PathInArc(), "->", si.PathOnDisk())
	return nil
}

// Сжимает файл блоками
func (arc *Arc) compressFile(fi header.PathProvider, arcBuf io.Writer) error {
	inFile, err := os.Open(fi.PathOnDisk())
	if err != nil {
		return errtype.Join(ErrOpenFileCompress(fi.PathOnDisk()), err)
	}
	defer inFile.Close()
	inBuf := bufio.NewReader(inFile)

	var (
		wrote, read int64
		crc         uint32
		wg          = sync.WaitGroup{}
	)

	for {
		// Заполняем буферы несжатыми частями (блоками) файла
		if read, err = arc.loadUncompressedBuf(inBuf); err != nil {
			return errtype.Join(ErrReadUncompressed, err)
		}

		wg.Wait()

		if read == 0 {
			wg.Add(1)
			go arc.flushWriteBuffer(&wg, arcBuf)
			break
		}

		// Сжимаем буферы
		if err = arc.compressBuffers(); err != nil {
			return errtype.Join(ErrCompress, err)
		}

		for i := 0; i < ncpu && compressedBuf[i].Len() > 0; i++ {
			// Пишем длину сжатого блока
			length := int64(compressedBuf[i].Len())
			if err = filesystem.BinaryWrite(writeBuf, length); err != nil {
				return errtype.Join(ErrWriteBufLen, err)
			}

			crc ^= crc32.Checksum(compressedBuf[i].Bytes(), crct)

			// Пишем сжатый блок
			if wrote, err = compressedBuf[i].WriteTo(writeBuf); err != nil {
				return errtype.Join(ErrWriteCompressBuf, err)
			}
			log.Println("В буфер записи записан блок размера:", wrote)
			compressor[i].Reset(compressedBuf[i])

			if writeBuf.Len() >= writeBufSize {
				wg.Add(1)
				go arc.flushWriteBuffer(&wg, arcBuf)

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

	fmt.Println(fi.PathInArc())

	return nil
}

// Загружает данные в буферы несжатых данных
func (Arc) loadUncompressedBuf(inBuf io.Reader) (read int64, err error) {
	var n int64

	for i := 0; i < ncpu && err != io.EOF; i++ {
		n, err = io.CopyN(decompressedBuf[i], inBuf, bufferSize)
		if err != nil && err != io.EOF {
			return 0, errtype.Join(ErrReadUncompressBuf, err)
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

			_, err := decompressedBuf[i].WriteTo(compressor[i])
			if err != nil {
				errors.Join()
				errChan <- errtype.Join(ErrWriteCompressor, err)
				return
			}
			if err = compressor[i].Close(); err != nil {
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
