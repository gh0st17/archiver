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
	fileBuf := bufio.NewReader(arcFile)
	var cBlockSize int64

	for totalRead < fi.CompressedSize {
		if err := binary.Read(arcFile, binary.LittleEndian, &cBlockSize); err != nil {
			return err
		}
		log.Println("Read block length:", cBlockSize)

		cBlockBytes := make([]byte, cBlockSize)
		remaining := fi.CompressedSize - totalRead
		if remaining < header.Size(len(cBlockBytes)) {
			cBlockBytes = cBlockBytes[:remaining]
		}

		n, err := io.ReadFull(fileBuf, cBlockBytes)
		if err != nil {
			return err
		}

		totalRead += header.Size(n)
		fi.CRC ^= crc32.Checksum(cBlockBytes, crct)

		decompData, err := c.DecompressBlock(cBlockBytes, arc.Compressor)
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
