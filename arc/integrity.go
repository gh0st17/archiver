package arc

import (
	"archiver/arc/header"
	"archiver/errtype"
	"fmt"
	"io"
)

// Проверяет целостность данных в архиве
func (arc Arc) IntegrityTest() error {
	headers, arcFile, err := arc.readHeaders()
	if err != nil {
		return errtype.ErrIntegrity(
			errtype.Join(ErrReadHeaders, err),
		)
	}
	defer arcFile.Close()

	for _, h := range headers {
		if fi, ok := h.(*header.FileItem); ok {
			if err = arc.checkFile(fi, arcFile); err != nil {
				return errtype.ErrIntegrity(
					errtype.Join(ErrCheckFile, err),
				)
			}
		}
	}

	return nil
}

// Распаковывает файл с проверкой CRC каждого
// блока сжатых данных
func (arc Arc) checkFile(fi *header.FileItem, arcFile io.ReadSeeker) error {
	var err error

	skipLen := len(fi.PathOnDisk()) + 26
	if _, err = arcFile.Seek(int64(skipLen), io.SeekCurrent); err != nil {
		return errtype.Join(ErrSkipHeaders, err)
	}

	if _, err = arc.checkCRC(fi.CRC(), arcFile); err == ErrWrongCRC {
		fmt.Println(fi.PathOnDisk() + ": Файл поврежден")
	} else if err != nil {
		return errtype.Join(ErrCheckCRC, err)
	} else {
		fmt.Println(fi.PathOnDisk() + ": OK")
	}

	return nil
}

// Считывает данные сжатого файла из arcFile,
// проверяет контрольную сумму и возвращает
// количество прочитанных байт
func (arc Arc) checkCRC(crc uint32, arcFile io.ReadSeeker) (read header.Size, err error) {
	var (
		n   int64
		eof error
	)

	for eof != io.EOF {
		if n, eof = arc.loadCompressedBuf(arcFile, &crc); eof != nil && eof != io.EOF {
			return 0, errtype.Join(ErrReadCompressed, eof)
		}

		read += header.Size(n)

		for i := 0; i < ncpu && compressedBuf[i].Len() > 0; i++ {
			compressedBuf[i].Reset()
		}
	}

	if _, err = arcFile.Seek(4, io.SeekCurrent); err != nil {
		return 0, errtype.Join(ErrSkipCRC, err)
	}

	if crc != 0 {
		return read, ErrWrongCRC
	}

	return read, nil
}
