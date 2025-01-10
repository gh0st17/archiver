package arc

import (
	"archiver/arc/header"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	fp "path/filepath"
)

// Собирает элементы из списка файлов
func fetchFile(filepath string) (h header.Header, err error) {
	file, err := os.OpenFile(filepath, os.O_RDONLY, 0444)
	if err != nil {
		return nil, err
	}

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	atime, mtime := amTimes(info)

	if info.IsDir() {
		di := header.NewDirItem(
			header.NewBase(fp.ToSlash(filepath), atime, mtime),
		)
		h = &di
	} else {
		fi := header.NewFileItem(
			header.NewBase(fp.ToSlash(filepath), atime, mtime),
			header.Size(info.Size()),
		)
		h = &fi
	}

	return h, nil
}

// Рекурсивно собирает элементы в директории
func fetchDir(path string) (headers []header.Header, err error) {
	err = fp.WalkDir(path, func(path string, _ os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		header, err := fetchFile(path)
		if err != nil {
			return err
		}

		headers = append(headers, header)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return headers, nil
}

// Читает заголовки из архива, определяет смещение данных
func (arc *Arc) readHeaders() (headers []header.Header, err error) {
	var (
		dirs  []header.DirItem
		files []header.FileItem
	)

	arcFile, err := os.OpenFile(arc.arcPath, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}
	defer arcFile.Close()

	arcFile.Seek(3, io.SeekCurrent) // Пропускаем магическое число и тип компрессора

	if dirs, err = arc.readDirsAndHeader(arcFile); err != nil {
		return nil, fmt.Errorf("read headers: %v", err)
	}

	arc.dataOffset, err = arcFile.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, err
	}
	log.Println("Data Offset:", arc.dataOffset)

	if files, err = arc.readFileHeaders(arcFile); err != nil {
		return nil, fmt.Errorf("read headers: %v", err)
	}

	for _, dir := range dirs {
		headers = append(headers, &dir)
	}
	for _, file := range files {
		headers = append(headers, &file)
	}

	return headers, nil
}

// Читает заголовки директории из архива
func (arc *Arc) readDirsAndHeader(arcFile io.Reader) (dirs []header.DirItem, err error) {
	// Читаем количество элементов
	var headersCount int64
	if err = binary.Read(arcFile, binary.LittleEndian, &headersCount); err != nil {
		return nil, err
	}
	dirs = make([]header.DirItem, headersCount)

	// Читаем заголовки
	for i := int64(0); i < headersCount; i++ {
		dirs[i].Read(arcFile)
	}

	return dirs, nil
}

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
			return nil, err
		}

		pos, _ := arcFile.Seek(0, io.SeekCurrent)
		log.Println("Read file data from pos:", pos)
		if dataSize, err = arc.readSize(arcFile); err != nil {
			return nil, err
		}
		fi.SetCSize(dataSize)

		if err = binary.Read(arcFile, binary.LittleEndian, &crc); err != nil {
			return nil, err
		}
		fi.SetCRC(crc)

		files = append(files, fi)
	}

	return files, nil
}

func (arc Arc) readSize(arcFile io.ReadSeeker) (read header.Size, err error) {
	var bufferSize int64

	for {
		if err = binary.Read(arcFile, binary.LittleEndian, &bufferSize); err != nil {
			return 0, fmt.Errorf("readSize: can't binary read: %v", err)
		}

		if bufferSize == -1 {
			break
		}

		if _, err = arcFile.Seek(bufferSize, io.SeekCurrent); err != nil {
			return 0, fmt.Errorf("readSize: can't seek: %v", err)
		}

		read += header.Size(bufferSize)
	}

	return read, nil
}
