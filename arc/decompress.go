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
	"path/filepath"
	"runtime"
	"sync"
)

// Распаковывает архив
func (arc Arc) Decompress(outputDir string) error {
	headers, err := arc.readHeaders()
	if err != nil {
		return err
	}

	f, err := os.Open(arc.ArchivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Seek(int64(arc.DataOffset), io.SeekCurrent)
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
			if err := arc.decompressFile(fi, f, outPath); err != nil {
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
func (arc Arc) decompressFile(fi *header.FileItem, arcFile io.ReadSeeker, outputPath string) error {
	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	// Если размер файла равен 0, то пропускаем запись
	if fi.UncompressedSize == 0 {
		return nil
	}

	// Записываем данные в буфер
	fileBuf := bufio.NewReader(arcFile)

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

	for totalRead < fi.CompressedSize {
		n, err := fillBlocksToDecompress(&compBytes, fileBuf, int(fi.CompressedSize-totalRead))
		if err != nil {
			return err
		}
		totalRead += header.Size(n)

		errChan := make(chan error, ncpu)
		for i, comp := range compBytes {
			fi.CRC ^= crc32.Checksum(compBytes[i], crct)

			if len(comp) == 0 {
				log.Println("comp", i, "breaking foo-loop with zero len")
				break
			}

			wg.Add(1)
			go func(i int) {
				defer wg.Done()

				unCompBytes[i], err = c.DecompressBlock(compBytes[i], arc.Compressor)
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

		for i := range unCompBytes {
			if len(compBytes[i]) == 0 {
				break
			}
			f.Write(unCompBytes[i])
		}

		clearBuffers(compBytes)
		clearBuffers(unCompBytes)
	}

	if fi.CRC != 0 {
		fmt.Println(outputPath + ": Файл поврежден")
	} else {
		fmt.Println(outputPath)
		log.Println("CRC:", fi.CRC)
	}

	return nil
}

func fillBlocksToDecompress(blocks *[][]byte, r io.Reader, remaining int) (int, error) {
	var read int
	var blockSize int64
	buf := bufio.NewReader(r)

	for i := range *blocks {
		if remaining == 0 {
			(*blocks)[i] = (*blocks)[i][:0]
			log.Println("block", i, "now have len:", len((*blocks)[i]))
			continue
		}

		if err := binary.Read(buf, binary.LittleEndian, &blockSize); err != nil {
			return 0, err
		}
		log.Println("Read block length:", blockSize)
		(*blocks)[i] = (*blocks)[i][:blockSize]

		if n, err := io.ReadFull(buf, (*blocks)[i]); err != nil {
			return 0, err
		} else {
			read += n
			remaining -= n
		}
	}

	return read, nil
}
