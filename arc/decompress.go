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
	"os"
	"path/filepath"
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
func (arc Arc) decompressFile(fi *header.FileItem, arcFile io.Reader, outputPath string) error {
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
	var totalRead header.Size
	crct := crc32.MakeTable(crc32.Koopman)
	buf := bufio.NewReader(arcFile)
	var blockSize int64

	for totalRead < fi.CompressedSize {
		if err := binary.Read(arcFile, binary.LittleEndian, &blockSize); err != nil {
			return err
		}

		blockBytes := make([]byte, blockSize)
		remaining := fi.CompressedSize - totalRead
		if remaining < header.Size(len(blockBytes)) {
			blockBytes = blockBytes[:remaining]
		}

		n, err := io.ReadFull(buf, blockBytes)
		if err != nil {
			return err
		}

		totalRead += header.Size(n)
		fi.CRC ^= crc32.Checksum(blockBytes, crct)

		blockBuffer := bytes.NewBuffer(blockBytes)
		decompData, err := c.DecompressBlock(blockBuffer, arc.Compressor)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return err
		}

		f.Write(decompData)
	}

	if fi.CRC != 0 {
		fmt.Println(outputPath + ": Файл поврежден\n")
	} else {
		fmt.Println(outputPath)
	}

	return nil
}
