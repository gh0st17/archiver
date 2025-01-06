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
	"path/filepath"
	"sync"
	"time"
)

// Распаковывает архив
func (arc Arc) Decompress(outputDir string) error {
	headers, err := arc.readHeaders()
	if err != nil {
		return fmt.Errorf("decompress: can't read headers: %v", err)
	}

	arcFile, err := os.Open(arc.ArchivePath)
	if err != nil {
		return fmt.Errorf("decompress: can't open archive: %v", err)
	}
	defer arcFile.Close()

	_, err = arcFile.Seek(int64(arc.DataOffset), io.SeekCurrent)
	if err != nil {
		return fmt.Errorf("decompress: can't seek: %v", err)
	}

	// 	Создаем файлы и директории
	var (
		arcBuf       = bufio.NewReader(arcFile)
		outPath      string
		atime, mtime time.Time
	)
	for _, h := range headers {
		outPath = filepath.Join(outputDir, h.Path())

		if fi, ok := h.(*header.FileItem); ok {
			if err = filesystem.CreatePath(filepath.Dir(outPath)); err != nil {
				return fmt.Errorf(
					"decompress: can't create path '%s': %v",
					outPath,
					err,
				)
			}
			if err = arc.decompressFile(fi, arcBuf, outPath); err != nil {
				return fmt.Errorf("decompress: %v", err)
			}

			atime, mtime = fi.AccTime, fi.ModTime
		} else {
			di := h.(*header.DirItem)
			if err = filesystem.CreatePath(outPath); err != nil {
				return fmt.Errorf(
					"decompress: can't create path '%s': %v",
					outPath,
					err,
				)
			}

			atime, mtime = di.AccTime, di.ModTime
		}

		if err = os.Chtimes(outPath, atime, mtime); err != nil {
			return err
		}
	}

	return nil
}

// Распаковывает файл
func (arc Arc) decompressFile(fi *header.FileItem, arcFile io.Reader, outputPath string) error {
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("decompressFile: %v", err)
	}
	defer outFile.Close()

	// Если размер файла равен 0, то пропускаем запись
	if fi.UncompressedSize == 0 {
		return nil
	}

	var totalRead header.Size

	for i := range uncompressedBuf {
		compressedBuf[i] = make([]byte, arc.maxCompLen)
		uncompressedBuf[i] = make([]byte, c.BufferSize)
	}

	for totalRead < fi.CompressedSize {
		remaining := int(fi.CompressedSize - totalRead)
		if n, err := arc.loadCompressedBuf(arcFile, remaining); err != nil {
			return fmt.Errorf("decompressFile: %v", err)
		} else {
			totalRead += header.Size(n)
		}

		if err = arc.decompressBuffers(&(fi.CRC)); err != nil {
			return fmt.Errorf("decompressFile: %v", err)
		}

		for i := 0; i < ncpu && len(compressedBuf[i]) > 0; i++ {
			outFile.Write(uncompressedBuf[i])

			compressedBuf[i] = compressedBuf[i][:cap(compressedBuf[i])]
		}
	}

	if fi.CRC != 0 {
		fmt.Println(outputPath + ": Файл поврежден")
	} else {
		fmt.Println(outputPath)
		log.Println("CRC:", fi.CRC)
	}

	return nil
}

// Загружает данные в буферы сжатых данных
func (Arc) loadCompressedBuf(r io.Reader, remaining int) (int, error) {
	var (
		read       int
		bufferSize int64
	)

	for i := range compressedBuf {
		if remaining == 0 {
			compressedBuf[i] = compressedBuf[i][:0]
			log.Println("compressedBuf", i, "doesn't need, skipping")
			continue
		}

		if err := binary.Read(r, binary.LittleEndian, &bufferSize); err != nil {
			return 0, fmt.Errorf("loadCompressedBuf: can't binary read %v", err)
		}
		log.Println("Read length of compressed data:", bufferSize)
		compressedBuf[i] = compressedBuf[i][:bufferSize]

		if n, err := io.ReadFull(r, compressedBuf[i]); err != nil {
			return 0, fmt.Errorf("loadCompressedBuf: can't read: %v", err)
		} else {
			read += n
			remaining -= n
		}
	}

	return read, nil
}

// Распаковывает данные в буферах сжатых данных
func (arc Arc) decompressBuffers(crc *uint32) error {
	var (
		errChan = make(chan error, ncpu)
		wg      sync.WaitGroup
	)

	for i := 0; i < ncpu && len(compressedBuf[i]) > 0; i++ {
		*crc ^= crc32.Checksum(compressedBuf[i], crct)

		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			buf := bytes.NewBuffer(compressedBuf[i])
			decompressor := c.NewReader(arc.CompType, buf)
			n, err := decompressor.Read(&uncompressedBuf[i])
			if err != nil {
				errChan <- err
			}
			uncompressedBuf[i] = uncompressedBuf[i][:n]
		}(i)
	}

	go func() {
		wg.Wait()
		close(errChan)
	}()

	for err := range errChan {
		return fmt.Errorf("decompressBuf: %v", err)
	}

	return nil
}
