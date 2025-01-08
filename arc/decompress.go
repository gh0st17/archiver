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
		outPath          string
		skipLen, dataPos int64
		firstFileIdx     int
	)

	for {
		var (
			di *header.DirItem
			ok bool
		)
		if di, ok = headers[firstFileIdx].(*header.DirItem); !ok {
			break
		}

		outPath = filepath.Join(outputDir, di.Filepath)
		if err = filesystem.CreatePath(outPath); err != nil {
			return fmt.Errorf("decompress: can't create path '%s': %v", outPath, err)
		}
		if err = os.Chtimes(outPath, di.AccTime, di.ModTime); err != nil {
			return err
		}
		fmt.Println(outPath)
		firstFileIdx++
	}

	for _, h := range headers[firstFileIdx:] {
		fi := h.(*header.FileItem)
		outPath = filepath.Join(outputDir, fi.Filepath)
		if err = filesystem.CreatePath(filepath.Dir(outPath)); err != nil {
			return fmt.Errorf("decompress: can't create path '%s': %v", outPath, err)
		}

		skipLen = int64(len(fi.Filepath)) + 32
		if dataPos, err = arcFile.Seek(skipLen, io.SeekCurrent); err != nil {
			return err
		}
		log.Println("Skipped", skipLen, "bytes of file header, read from pos:", dataPos)

		if integ {
			arc.checkCRC(fi, arcFile)

			if fi.Damaged {
				fmt.Printf("Пропускаю повежденный '%s'\n", fi.Filepath)
				continue
			} else {
				arcFile.Seek(dataPos, io.SeekStart)
				log.Println("Set to pos:", dataPos+skipLen)
			}
		}

		if err = arc.decompressFile(fi, arcFile, outPath); err != nil {
			return fmt.Errorf("decompress: %v", err)
		}

		if dataPos, err = arcFile.Seek(4, io.SeekCurrent); err != nil {
			return err
		}
		log.Println("Skipped CRC, new arcFile pos:", dataPos)

		if err = os.Chtimes(outPath, fi.AccTime, fi.ModTime); err != nil {
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
		pos, _ := arcFile.Seek(8, io.SeekCurrent)
		log.Println("Empty size, set to pos:", pos)
		return nil
	}

	for i := 0; i < ncpu; i++ {
		if cap(compressedBuf[i]) < int(arc.maxCompLen) {
			compressedBuf[i] = make([]byte, arc.maxCompLen)
		}
	}

	var eof error
	for eof == nil {
		if _, eof = arc.loadCompressedBuf(arcFile); eof != nil {
			if eof != io.EOF && eof != io.ErrUnexpectedEOF {
				return fmt.Errorf("decompressFile: %v", eof)
			}
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
func (Arc) loadCompressedBuf(r io.ReadSeeker) (int, error) {
	var (
		read, n    int
		bufferSize int64
		err, eof   error
	)

	skip := func(i int) {
		compressedBuf[i] = compressedBuf[i][:0]
		log.Println("compressedBuf", i, "doesn't need, skipping")
	}

	pos, _ := r.Seek(0, io.SeekCurrent)
	log.Println("loadCompBuf: start reading from pos: ", pos)

	for i := 0; i < ncpu; i++ {
		if eof == io.EOF {
			skip(i)
			continue
		}

		if err = binary.Read(r, binary.LittleEndian, &bufferSize); err != nil {
			return 0, fmt.Errorf("loadCompressedBuf: can't binary read: %v", err)
		}

		if bufferSize == -1 {
			log.Println("Read EOF")
			skip(i)
			eof = io.EOF
			continue
		}
		log.Println("Read length of compressed data:", bufferSize)

		compressedBuf[i] = compressedBuf[i][:bufferSize]

		if n, eof = io.ReadFull(r, compressedBuf[i]); eof != nil {
			if eof != io.EOF && eof != io.ErrUnexpectedEOF {
				return 0, fmt.Errorf("loadCompressedBuf: can't read: %v", eof)
			}
		}

		read += n
	}

	return read, eof
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
