package arc

import (
	"archiver/arc/header"
	"fmt"
	"hash/crc32"
	"io"
	"log"
)

// Проверяет целостность данных в архиве
func (arc Arc) IntegrityTest() error {
	headers, err := arc.readHeaders()
	if err != nil {
		return fmt.Errorf("decompress: can't read headers: %v", err)
	}

	arcFile, err := arc.prepareArcFile()
	if err != nil {
		return err
	}
	defer arcFile.Close()

	for _, h := range headers {
		if fi, ok := h.(*header.FileItem); ok {
			if err = arc.checkFile(fi, arcFile); err != nil {
				return fmt.Errorf("integrity test: %v", err)
			}
		}
	}

	return nil
}

// Распаковывает файл проверяя CRC32 каждого блока сжатых данных
func (arc Arc) checkFile(fi *header.FileItem, arcFile io.ReadSeeker) error {
	// Если размер файла равен 0, то пропускаем
	if fi.UcSize() == 0 {
		return nil
	}

	skipLen := int64(len(fi.Path())) + 32
	if _, err := arcFile.Seek(skipLen, io.SeekCurrent); err != nil {
		return err
	}

	if _, err := arc.checkCRC(fi, arcFile); err != nil {
		return err
	}

	if fi.IsDamaged() {
		fmt.Println(fi.Path() + ": Файл поврежден")
	} else {
		fmt.Println(fi.Path() + ": OK")
		log.Println("CRC matched")
	}

	return nil
}

func (arc Arc) checkCRC(fi *header.FileItem, arcFile io.ReadSeeker) (header.Size, error) {
	var (
		totalRead header.Size
		n         int
		err, eof  error
		crc       uint32
	)

	for i := 0; i < ncpu; i++ {
		if cap(compressedBuf[i]) < int(arc.maxCompLen) {
			compressedBuf[i] = make([]byte, arc.maxCompLen)
		}
	}

	for eof == nil {
		if n, eof = arc.loadCompressedBuf(arcFile); eof != nil {
			if eof != io.EOF && eof != io.ErrUnexpectedEOF {
				return 0, fmt.Errorf("check CRC: %v", eof)
			}
		} else {
			totalRead += header.Size(n)
		}

		for i := 0; i < ncpu && len(compressedBuf[i]) > 0; i++ {
			crc ^= crc32.Checksum(compressedBuf[i], crct)
			compressedBuf[i] = compressedBuf[i][:cap(compressedBuf[i])]
		}
	}
	fi.SetDamaged(crc != fi.CRC())

	if _, err = arcFile.Seek(4, io.SeekCurrent); err != nil {
		return 0, err
	}

	return totalRead, nil
}
