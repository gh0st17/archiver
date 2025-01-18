package arc

import (
	"archiver/arc/header"
	c "archiver/compressor"
	"archiver/errtype"
	"archiver/filesystem"
	"bufio"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"log"
	"os"
	"sync"
)

// Создает архив
func (arc Arc) Compress(paths []string) error {
	var (
		headers []header.Header
		err     error
	)

	filesystem.PrintPathsCheck(paths)
	if headers, err = arc.fetchHeaders(paths); err != nil {
		return err
	}
	headers = header.DropDups(headers)

	dirs, files := arc.splitHeaders(headers)

	for i := 0; i < ncpu; i++ {
		compressor[i], err = c.NewWriter(arc.ct, compressedBuf[i], arc.cl)
		if err != nil {
			return errtype.ErrCompress(ErrCompressorInit, err)
		}
	}

	closeRemove := func(arcFile io.Closer) {
		arcFile.Close()
		arc.RemoveTmp()
	}

	arcFile, err := arc.writeHeaderDirs(dirs)
	if err != nil {
		closeRemove(arcFile)
		return errtype.ErrCompress(ErrWriteDirHeaders, err)
	}
	arcBuf := bufio.NewWriter(arcFile)

	for _, fi := range files {
		if err = fi.Write(arcBuf); err != nil {
			closeRemove(arcFile)
			return errtype.ErrCompress(ErrWriteFileHeader, err)
		}

		if err = arc.compressFile(fi, arcBuf); err != nil {
			closeRemove(arcFile)
			return errtype.ErrCompress(ErrCompressFile, err)
		}
	}
	arcBuf.Flush()
	arcFile.Close()

	return nil
}

// Сжимает файл блоками
func (arc *Arc) compressFile(fi *header.FileItem, arcBuf io.Writer) error {
	inFile, err := os.Open(fi.PathOnDisk())
	if err != nil {
		return errtype.ErrCompress(ErrOpenFileCompress(fi.PathOnDisk()), err)
	}
	defer inFile.Close()
	inBuf := bufio.NewReaderSize(inFile, int(c.BufferSize))

	var (
		wrote, read int64
		crc         uint32
	)

	for {
		// Заполняем буферы несжатыми частями (блоками) файла
		if read, err = arc.loadUncompressedBuf(inBuf); err != nil {
			return errtype.ErrCompress(ErrReadUncompressed, err)
		}

		if read == 0 {
			break
		}

		// Сжимаем буферы
		if err = arc.compressBuffers(); err != nil {
			return errtype.ErrCompress(ErrCompress, err)
		}

		for i := 0; i < ncpu && compressedBuf[i].Len() > 0; i++ {
			// Пишем длину сжатого блока
			length := int64(compressedBuf[i].Len())
			if err = binary.Write(arcBuf, binary.LittleEndian, length); err != nil {
				return errtype.ErrCompress(ErrWriteBufLen, err)
			}

			crc ^= crc32.Checksum(compressedBuf[i].Bytes(), crct)

			// Пишем сжатый блок
			if wrote, err = compressedBuf[i].WriteTo(arcBuf); err != nil {
				return errtype.ErrCompress(ErrReadCompressBuf, err)
			}
			log.Println("В буфер записан блок размера:", wrote)
			compressor[i].Reset(compressedBuf[i])
		}
	}

	// Пишем признак конца файла
	if err = binary.Write(arcBuf, binary.LittleEndian, int64(-1)); err != nil {
		return errtype.ErrCompress(ErrWriteEOF, err)
	}
	log.Println("Записан EOF")

	// Пишем контрольную сумму
	if err = binary.Write(arcBuf, binary.LittleEndian, crc); err != nil {
		return errtype.ErrCompress(ErrWriteCRC, err)
	}
	log.Printf("Записан CRC: %X\n", crc)

	fmt.Println(fi.PathInArc())

	return nil
}

// Загружает данные в буферы несжатых данных
func (Arc) loadUncompressedBuf(inBuf io.Reader) (read int64, err error) {
	var n int64

	for i := 0; i < ncpu && err != io.EOF; i++ {
		n, err = io.CopyN(decompressedBuf[i], inBuf, c.BufferSize)
		if err != nil && err != io.EOF {
			return 0, errtype.ErrCompress(ErrReadUncompressBuf, err)
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
				errChan <- errtype.ErrCompress(ErrWriteCompressor, err)
				return
			}
			if err = compressor[i].Close(); err != nil {
				errChan <- errtype.ErrCompress(ErrCloseCompressor, err)
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
