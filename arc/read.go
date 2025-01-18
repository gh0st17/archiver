package arc

import (
	"archiver/arc/header"
	"archiver/errtype"
	"archiver/filesystem"
	"encoding/binary"
	"io"
	"log"
	"os"
	fp "path/filepath"
)

// Проверяет чем является path, директорией или файлом,
// возвращает интерфейс заголовка, указывающий на
// соответствующий тип
func fetchPath(path string) (h header.Header, err error) {
	file, err := os.OpenFile(path, os.O_RDONLY, 0444)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	atime, mtime := amTimes(info)

	b, err := header.NewBase(fp.ToSlash(path), atime, mtime)
	if err != nil {
		return nil, err
	}

	if info.IsDir() {
		h = header.NewDirItem(b)
	} else {
		h = header.NewFileItem(b, header.Size(info.Size()))
	}

	return h, nil
}

// Рекурсивно собирает элементы в директории
func fetchDir(path string) (headers []header.Header, err error) {
	var visited = map[string]struct{}{}

	err = fp.WalkDir(path, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		// Проверка на символическую ссылку
		if info.Mode()&os.ModeSymlink != 0 {
			// Получение назначения ссылки
			dst, err := os.Readlink(path)
			if err != nil {
				return err
			}

			// Указывает на абсолютный или отностительный путь?
			if !fp.IsAbs(dst) {
				dst = filesystem.Clean(fp.Join(fp.Dir(path), dst))
			}

			if _, seen := visited[dst]; seen {
				return nil // Этот пусть уже посещен
			}
			visited[dst] = struct{}{}

			if info, err = os.Stat(dst); err != nil {
				return err
			}

			if info.IsDir() { // Ссылка указывает на директорию
				lheaders, err := fetchDir(dst)
				if err != nil {
					return err
				}
				headers = append(headers, lheaders...)
			} else { // Ссылка указывает на файл
				lheader, err := fetchPath(dst)
				if err != nil {
					return err
				}
				headers = append(headers, lheader)
			}
		} else {
			// Обычный путь
			header, err := fetchPath(path)
			if err != nil {
				return err
			}
			headers = append(headers, header)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return headers, nil
}

// Собирает элементы файловой системы в заголовки
func (Arc) fetchHeaders(paths []string) (headers []header.Header, err error) {
	var (
		dirHeaders []header.Header
		header     header.Header
	)

	for _, path := range paths { // Получение списка файлов и директории
		// Добавление директории в заголовок
		// и ее рекурсивный обход
		if filesystem.DirExists(path) {
			if dirHeaders, err = fetchDir(path); err == nil {
				headers = append(headers, dirHeaders...)
			} else {
				return nil, errtype.ErrCompress(ErrFetchDirs, err)
			}
			continue
		}

		if header, err = fetchPath(path); err == nil { // Добавалние файла в заголовок
			headers = append(headers, header)
		} else {
			return nil, errtype.ErrCompress(ErrFetchDirs, err)
		}
	}
	return headers, nil
}

// Читает заголовки из архива, определяет смещение данных
func (arc *Arc) readHeaders() (headers []header.Header, arcFile *os.File, err error) {
	var (
		dirs  []header.DirItem
		files []header.FileItem
	)

	arcFile, err = os.OpenFile(arc.arcPath, os.O_RDONLY, 0644)
	if err != nil {
		return nil, nil, errtype.ErrRuntime(ErrOpenArc, err)
	}

	arcFile.Seek(3, io.SeekCurrent) // Пропускаем магическое число и тип компрессора

	if dirs, err = arc.readDirHeaders(arcFile); err != nil {
		return nil, nil, errtype.ErrRuntime(ErrReadHeaders, err)
	}

	dataPos, _ := arcFile.Seek(0, io.SeekCurrent)
	if files, err = arc.readFileHeaders(arcFile); err != nil {
		return nil, nil, errtype.ErrRuntime(ErrReadHeaders, err)
	}
	arcFile.Seek(dataPos, io.SeekStart)

	for _, dir := range dirs {
		headers = append(headers, &dir)
	}
	for _, file := range files {
		headers = append(headers, &file)
	}

	return headers, arcFile, nil
}

// Читает заголовки директории из архива
func (arc *Arc) readDirHeaders(arcFile io.Reader) (dirs []header.DirItem, err error) {
	var headersCount int64 // Количество заголовков директории
	if err = binary.Read(arcFile, binary.LittleEndian, &headersCount); err != nil {
		return nil, errtype.ErrRuntime(ErrReadHeadersCount, err)
	}
	dirs = make([]header.DirItem, headersCount)

	// Читаем заголовки директории
	for i := int64(0); i < headersCount; i++ {
		if err = dirs[i].Read(arcFile); err != nil {
			return nil, errtype.ErrRuntime(ErrReadDirHeaders, err)
		}
	}

	return dirs, nil
}

// Читает и возвращает заголовки файлов
func (arc Arc) readFileHeaders(arcFile io.ReadSeeker) ([]header.FileItem, error) {
	var (
		files    []header.FileItem
		dataSize header.Size
		err      error
		crc      uint32
	)

	for { // One cycle is one file
		var fi header.FileItem

		if err = fi.Read(arcFile); err != nil {
			if err == io.EOF {
				break
			}
			return nil, errtype.ErrRuntime(ErrReadFileHeader, err)
		}

		pos, _ := arcFile.Seek(0, io.SeekCurrent)
		log.Println("Читаю размер сжатых данных с позиции:", pos)
		if dataSize, err = arc.skipFileData(arcFile, false); err != nil {
			return nil, errtype.ErrRuntime(ErrReadCompLen, err)
		}
		fi.SetCSize(dataSize)

		if err = binary.Read(arcFile, binary.LittleEndian, &crc); err != nil {
			return nil, errtype.ErrRuntime(ErrReadCRC, err)
		}
		fi.SetCRC(crc)

		files = append(files, fi)
	}

	return files, nil
}

// Пропускает файл в дескрипторе файла архива
func (arc Arc) skipFileData(arcFile io.ReadSeeker, skipCRC bool) (read header.Size, err error) {
	var bufferSize int64

	for {
		if err = binary.Read(arcFile, binary.LittleEndian, &bufferSize); err != nil {
			return 0, errtype.ErrDecompress(ErrReadCompLen, err)
		}

		if bufferSize == -1 {
			break
		} else if arc.checkBufferSize(bufferSize) {
			return 0, errtype.ErrDecompress(ErrBufSize(bufferSize), err)
		}

		read += header.Size(bufferSize)

		if _, err = arcFile.Seek(bufferSize, io.SeekCurrent); err != nil {
			return 0, errtype.ErrDecompress(ErrSkipData, err)
		}
	}

	if skipCRC {
		if _, err = arcFile.Seek(4, io.SeekCurrent); err != nil {
			return 0, errtype.ErrDecompress(ErrSkipCRC, err)
		}
	}

	return read, nil
}
