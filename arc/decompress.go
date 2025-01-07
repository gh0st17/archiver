package arc

import (
	"archiver/arc/header"
	c "archiver/compressor"
	"archiver/filesystem"
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

func (arc Arc) prepareArcFile() (*os.File, error) {
	arcFile, err := os.Open(arc.ArchivePath)
	if err != nil {
		return nil, fmt.Errorf("prepareArcFile: can't open archive: %v", err)
	}

	pos, err := arcFile.Seek(arc.DataOffset, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("prepareArcFile: can't seek: %v", err)
	}
	log.Println("Prepare: pos set to ", pos)

	return arcFile, nil
}

// Распаковывает архив
func (arc Arc) Decompress(outputDir string, integ bool) error {
	headers, err := arc.readHeaders()
	if err != nil {
		return fmt.Errorf("decompress: can't read headers: %v", err)
	}

	arcFile, err := arc.prepareArcFile()
	if err != nil {
		return err
	}
	defer arcFile.Close()

	// 	Создаем файлы и директории
	var (
		outPath      string
		atime, mtime time.Time
		skipLen, pos int64
	)

	for _, h := range headers {
		outPath = filepath.Join(outputDir, h.Path())

		if fi, ok := h.(*header.FileItem); ok {
			skipLen = int64(len(fi.Filepath)) + 32
			if pos, err = arcFile.Seek(skipLen, io.SeekCurrent); err != nil {
				return err
			}
			log.Println("Skipped", skipLen, "bytes of file header, read from pos:", pos)

			if integ {
				arc.checkCRC(fi, arcFile)
			}

			if fi.Damaged {
				fmt.Printf("Пропускаю повежденный '%s'\n", fi.Filepath)
				continue
			}

			if !fi.Damaged && integ {
				arcFile.Seek(pos, io.SeekStart)
				log.Println("Set to pos:", pos+skipLen)
			}

			if err = filesystem.CreatePath(filepath.Dir(outPath)); err != nil {
				return fmt.Errorf(
					"decompress: can't create path '%s': %v",
					outPath,
					err,
				)
			}

			if err = arc.decompressFile(fi, arcFile, outPath); err != nil {
				return fmt.Errorf("decompress: %v", err)
			}

			if pos, err = arcFile.Seek(4, io.SeekCurrent); err != nil {
				return err
			}
			log.Println("Skipped CRC, new arcFile pos:", pos)

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

		fmt.Println(outPath)
	}

	return nil
}

// Распаковывает файл
func (arc Arc) decompressFile(fi *header.FileItem, arcFile io.ReadSeeker, outputPath string) error {
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("decompressFile: %v", err)
	}
	defer outFile.Close()

	// Если размер файла равен 0, то пропускаем запись
	if fi.UncompressedSize == 0 {
		return nil
	}

	if cap(uncompressedBuf[0]) == 0 {
		for i := range uncompressedBuf {
			compressedBuf[i] = make([]byte, arc.maxCompLen)
			uncompressedBuf[i] = make([]byte, c.BufferSize)
		}
	}

	var eof bool
	for !eof {
		if _, eof, err = arc.loadCompressedBuf(arcFile); err != nil {
			return fmt.Errorf("decompressFile: %v", err)
		}

		if err = arc.decompressBuffers(&(fi.CRC)); err != nil {
			return fmt.Errorf("decompressFile: %v", err)
		}

		for i := 0; i < ncpu && len(compressedBuf[i]) > 0; i++ {
			outFile.Write(uncompressedBuf[i])

			compressedBuf[i] = compressedBuf[i][:cap(compressedBuf[i])]
		}
	}

	return nil
}

// Загружает данные в буферы сжатых данных
func (Arc) loadCompressedBuf(r io.ReadSeeker) (int, bool, error) {
	var (
		read, n    int
		bufferSize int64
		eof        bool
		err        error
	)

	skip := func(i int) {
		compressedBuf[i] = compressedBuf[i][:0]
		log.Println("compressedBuf", i, "doesn't need, skipping")
	}

	pos, _ := r.Seek(0, io.SeekCurrent)
	log.Println("loadCompBuf: start reading from pos: ", pos)

	for i := range compressedBuf {
		if eof {
			skip(i)
			continue
		}

		if err = binary.Read(r, binary.LittleEndian, &bufferSize); err != nil {
			return 0, false, fmt.Errorf("loadCompressedBuf: can't binary read: %v", err)
		}

		if bufferSize == -1 {
			log.Println("Read EOF")
			skip(i)
			eof = true
			continue
		}
		log.Println("Read length of compressed data:", bufferSize)

		compressedBuf[i] = compressedBuf[i][:bufferSize]

		if n, err = io.ReadFull(r, compressedBuf[i]); err != nil {
			return 0, false, fmt.Errorf("loadCompressedBuf: can't read: %v", err)
		} else {
			read += n
		}
	}

	return read, eof, nil
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
