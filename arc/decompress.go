package arc

import (
	"archiver/arc/header"
	c "archiver/compressor"
	"archiver/filesystem"
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

// Распаковывает архив
func Decompress(arcParams *Arc, outputDir string) error {
	f, err := os.Open(arcParams.ArchivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	r := bufio.NewReader(f)
	// 	Читаем заголовки
	headers, err := readHeaders(r)
	if err != nil {
		return err
	}

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

		if h.Type() == header.File {
			// Читаем данные
			fi := h.(*header.FileItem)

			data = make([]byte, fi.CompressedSize)
			if _, err := io.ReadFull(r, data); err != nil {
				return err
			}
			h.(*header.FileItem).SetData(data)

			wg.Add(1)
			go func(path string, fi *header.FileItem) { // Горутина для параллельной распаковки
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				if err := decompressFile(fi, path, arcParams.Compressor); err != nil {
					errChan <- err
					return
				}

				os.Chtimes(path, fi.AccTime, fi.ModTime)
			}(outPath, h.(*header.FileItem))
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
	buf := bytes.NewBuffer(fi.Bytes())

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

	fi.SetData(nil)

	return nil
}
