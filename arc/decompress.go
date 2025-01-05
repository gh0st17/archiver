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
	"path/filepath"
	"sync"
)

// Распаковывает архив
func (arc Arc) Decompress(outputDir string) error {
	headers, err := arc.readHeaders()
	if err != nil {
		return err
	}

	arcFile, err := os.Open(arc.ArchivePath)
	if err != nil {
		return err
	}
	defer arcFile.Close()

	_, err = arcFile.Seek(int64(arc.DataOffset), io.SeekCurrent)
	if err != nil {
		return err
	}

	// 	Создаем файлы и директории
	var outPath string
	for _, h := range headers {
		outPath = filepath.Join(outputDir, h.Path())

		err := filesystem.CreatePath(filepath.Dir(outPath))
		if err != nil {
			return err
		}

		if fi, ok := h.(*header.FileItem); ok {
			if err := arc.decompressFile(fi, arcFile, outPath); err != nil {
				return err
			}

			os.Chtimes(outPath, fi.AccTime, fi.ModTime)
		} else {
			di := h.(*header.DirItem)
			os.Chtimes(outPath, di.AccTime, di.ModTime)
		}
	}

	return nil
}

// Распаковывает файл
func (arc Arc) decompressFile(fi *header.FileItem, arcFile io.Reader, outputPath string) error {
	outFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	// Если размер файла равен 0, то пропускаем запись
	if fi.UncompressedSize == 0 {
		return nil
	}

	var totalRead header.Size

	for i := range uncompressedBuf {
		compressedBuf[i] = make([]byte, arc.maxCompLen)
	}

	for totalRead < fi.CompressedSize {
		remaining := int(fi.CompressedSize - totalRead)
		if n, err := arc.loadCompressedBuf(arcFile, remaining); err != nil {
			return err
		} else {
			totalRead += header.Size(n)
		}

		if err = arc.decompressBuffers(&(fi.CRC)); err != nil {
			return err
		}

		for i := range uncompressedBuf {
			if len(compressedBuf[i]) == 0 {
				break
			}
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
			log.Println("compressedBuf", i, "not needed")
			continue
		}

		if err := binary.Read(r, binary.LittleEndian, &bufferSize); err != nil {
			return 0, err
		}
		log.Println("Read compressedBuf length:", bufferSize)
		compressedBuf[i] = compressedBuf[i][:bufferSize]

		if n, err := io.ReadFull(r, compressedBuf[i]); err != nil {
			return 0, err
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

	for i, buf := range compressedBuf {
		*crc ^= crc32.Checksum(compressedBuf[i], crct)

		if len(buf) == 0 {
			log.Println("compressedBuf", i, "breaking foo-loop with zero len")
			break
		}

		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			var err error

			uncompressedBuf[i], err = c.Decompress(compressedBuf[i], arc.Compressor)
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
		return fmt.Errorf("decompressBuf: %v", err)
	}

	return nil
}
