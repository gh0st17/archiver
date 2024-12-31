package arc

import (
	"archiver/arc/header"
	c "archiver/compressor"
	"archiver/filesystem"
	"bufio"
	"bytes"
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

	r := bufio.NewReader(f)
	r.Discard(int(arc.DataOffset)) // Перепещаемся к началу данных

	// 	Создаем файлы и директории
	var outPath string
	for _, h := range headers {
		outPath = filepath.Join(outputDir, h.Path())

		err := filesystem.CreatePath(filepath.Dir(outPath))
		if err != nil {
			return err
		}

		if fi, ok := h.(*header.FileItem); ok {
			if err := arc.decompressFile(fi, r, outPath); err != nil {
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
	fmt.Print(outputPath)
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
	var totalRead int
	buffer := make([]byte, c.BufferSize)
	crct := crc32.MakeTable(crc32.Koopman)

	for totalRead < int(fi.CompressedSize) {
		remaining := int(fi.CompressedSize) - totalRead
		if remaining < len(buffer) {
			// Ограничиваем размер буфера если остаток меньше его размера
			buffer = buffer[:remaining]
		}

		n, err := io.ReadFull(arcFile, buffer)
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				// Если достигли конца файла, просто продолжаем
				// Уменьшаем буфер до фактического количества прочитанных байт
				buffer = buffer[:n]
			} else {
				return err
			}
		}

		totalRead += n
		fi.CRC ^= crc32.Checksum(buffer[:n], crct)
		buf := bufio.NewReader(bytes.NewReader(buffer[:n]))
		decompData, err := c.DecompressBlock(buf, arc.Compressor)
		if err != nil {
			fmt.Println(": Ошибка распаковки:", err)
			return nil
		}

		f.Write(decompData)
	}

	if fi.CRC != 0 {
		fmt.Println(": Файл", outputPath, "поврежден")
	} else {
		fmt.Println()
	}

	return nil
}
