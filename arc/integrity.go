package arc

import (
	"archiver/arc/header"
	"fmt"
	"hash/crc32"
	"io"
)

// Проверяет целостность данных в архиве
func (arc Arc) IntegrityTest() error {
	headers, err := arc.readHeaders()
	if err != nil {
		return fmt.Errorf("integrity test: can't read headers: %v", err)
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

// Распаковывает файл с проверкой CRC каждого блока сжатых данных
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
	}

	return nil
}

// Считывает данные сжатого файла из arcFile,
// проверяет контрольную сумму и возвращает
// количество прочитанных байт
func (arc Arc) checkCRC(fi *header.FileItem, arcFile io.ReadSeeker) (read header.Size, err error) {
	var (
		n   int64
		crc = fi.CRC()
	)

	for n != -1 {
		if n, err = arc.loadCompressedBuf(arcFile); err != nil {
			return 0, fmt.Errorf("check CRC: %v", err)
		}

		read += header.Size(n)

		for i := 0; i < ncpu && compressedBuf[i].Len() > 0; i++ {
			crc ^= crc32.Checksum(compressedBuf[i].Bytes(), crct)
			compressedBuf[i].Reset()
		}
	}
	fi.SetDamaged(crc != 0)

	if _, err = arcFile.Seek(4, io.SeekCurrent); err != nil {
		return 0, err
	}

	return read, nil
}
