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
	"runtime"
	"sync"
)

// Распаковывает архив
func Decompress(arc *Arc, outputDir string) error {
	headers, err := readHeaders(arc)
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

	var (
		outPath string
		data    []byte
		sem     = make(chan struct{}, runtime.NumCPU())
		errChan = make(chan error, len(headers))
		wg      sync.WaitGroup
	)

	// 	Создаем файлы и директории
	for _, h := range headers {
		outPath = filepath.Join(outputDir, h.Path())

		err := filesystem.CreatePath(filepath.Dir(outPath))
		if err != nil {
			return err
		}

		if fi, ok := h.(*header.FileItem); ok {
			// Читаем данные
			data = make([]byte, fi.CompressedSize)
			if _, err := io.ReadFull(r, data); err != nil {
				return err
			}
			h.(*header.FileItem).Data = data

			wg.Add(1)
			go func(outPath string, fi *header.FileItem) { // Горутина для параллельной распаковки
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				if err := decompressFile(fi, outPath, arc.Compressor); err != nil {
					errChan <- err
					return
				}

				os.Chtimes(outPath, fi.AccTime, fi.ModTime)
			}(outPath, fi)
		} else {
			di := h.(*header.DirItem)
			os.Chtimes(outPath, di.AccTime, di.ModTime)
		}

	}

	go func() {
		wg.Wait()
		close(errChan)
	}()

	for err := range errChan { // Обработка первой ошибки из горутины
		return err
	}

	return nil
}

// Распаковывает файл
func decompressFile(fi *header.FileItem, outputPath string, c c.Compressor) error {
	crct := crc32.MakeTable(crc32.Koopman)
	if crc := crc32.Checksum(fi.Data, crct); crc != fi.CRC {
		fmt.Println(
			outputPath,
			"Контрольная сумма не совпадает, файл поврежден.",
			"Пропускаю...",
		)
		return nil
	}

	fmt.Println(outputPath)
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
	buf := bytes.NewBuffer(fi.Data)

	cr, err := c.NewReader(buf)
	if err != nil {
		return err
	}

	if _, err = io.Copy(f, cr); err != nil {
		return err
	}

	if err := cr.Close(); err != nil {
		return err
	}

	fi.Data = nil

	return nil
}
