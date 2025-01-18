package arc

import (
	"archiver/arc/header"
	c "archiver/compressor"
	"archiver/errtype"
	"bufio"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
)

// Распаковывает архив
func (arc Arc) Decompress(outputDir string, integ bool) error {
	headers, arcFile, err := arc.readHeaders()
	if err != nil {
		return errtype.ErrDecompress(ErrReadHeaders, err)
	}
	defer arcFile.Close()

	dirsSyms, files := arc.splitHeaders(headers)
	arc.restoreDirsSymsPaths(dirsSyms, outputDir)

	// 	Создаем файлы и директории
	var (
		outPath string
		dataPos int64
		skipLen int
	)

	for _, fi := range files {
		if err = fi.RestorePath(outputDir); err != nil {
			return errtype.ErrDecompress(ErrRestorePath(fi.PathOnDisk()), err)
		}

		skipLen = len(fi.PathOnDisk()) + 26
		if dataPos, err = arcFile.Seek(int64(skipLen), io.SeekCurrent); err != nil {
			return errtype.ErrDecompress(ErrSkipHeaders, err)
		}
		log.Println("Пропущенно", skipLen, "байт заголовка, читаю с позиции:", dataPos)

		outPath = filepath.Join(outputDir, fi.PathOnDisk())
		if _, err := os.Stat(outPath); err == nil && !arc.replaceAll {
			if arc.replaceInput(outPath, arcFile) {
				continue
			}
		}

		if integ { // --xinteg
			_, err = arc.checkCRC(fi.CRC(), arcFile)

			if err == ErrWrongCRC {
				fmt.Printf("Пропускаю поврежденный '%s'\n", fi.PathOnDisk())
				continue
			} else {
				arcFile.Seek(dataPos, io.SeekStart)
				log.Println("Файл цел, установлена позиция:", int(dataPos)+skipLen)
			}
		}

		if err = arc.decompressFile(fi, arcFile, outPath); err != nil {
			return errtype.ErrDecompress(ErrDecompressFile, err)
		}

		if dataPos, err = arcFile.Seek(4, io.SeekCurrent); err != nil {
			return errtype.ErrDecompress(ErrSkipCRC, err)
		}
		log.Println("Пропуск CRC, установлена позиция:", dataPos)

		if fi.IsDamaged() {
			fmt.Printf("%s: CRC сумма не совпадает\n", outPath)
		} else {
			fmt.Println(outPath)
		}

		fi.RestoreTime(outputDir)
	}

	// Сброс декомпрессоров перед новым использованием этой функции
	for i := 0; i < ncpu; i++ {
		decompressor[i] = nil
	}

	return nil
}

// Воссоздает директории из заголовков
func (Arc) restoreDirsSymsPaths(dirsSyms []header.Header, outputDir string) error {
	for _, h := range dirsSyms {
		if di, ok := h.(*header.DirItem); ok {
			if err := di.RestorePath(outputDir); err != nil {
				return errtype.ErrDecompress(ErrRestorePath(di.PathOnDisk()), err)
			}
			di.RestoreTime(outputDir)
		} else if si, ok := h.(*header.SymDirItem); ok {
			if err := si.RestorePath(outputDir); err != nil {
				return errtype.ErrDecompress(ErrRestorePath(si.PathOnDisk()), err)
			}
		}
	}

	return nil
}

// Обрабатывает диалог замены файла
func (arc *Arc) replaceInput(outPath string, arcFile io.ReadSeeker) bool {
	var input rune
	stdin := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("Файл '%s' существует, заменить? [(Д)а/(Н)ет/(В)се]: ", outPath)
		input, _, _ = stdin.ReadRune()

		switch input {
		case 'A', 'a', 'В', 'в':
			arc.replaceAll = true
		case 'Y', 'y', 'Д', 'д':
		case 'N', 'n', 'Н', 'н':
			arc.skipFileData(arcFile, true)
			return true
		default:
			stdin.ReadString('\n')
			continue
		}
		break
	}

	return false
}

// Распаковывает файл
func (arc Arc) decompressFile(fi *header.FileItem, arcFile io.ReadSeeker, outPath string) error {
	outFile, err := os.Create(outPath)
	if err != nil {
		return errtype.ErrDecompress(ErrCreateOutFile, err)
	}
	defer outFile.Close()
	outBuf := bufio.NewWriter(outFile)

	// Если размер файла равен 0, то пропускаем запись
	if fi.UcSize() == 0 {
		if pos, err := arcFile.Seek(8, io.SeekCurrent); err != nil {
			return errtype.ErrDecompress(ErrSkipEOF, err)
		} else {
			log.Println("Нулевой размер, перемещаю на позицию:", pos)
			return nil
		}
	}

	var (
		wrote, read int64
		crc         = fi.CRC()
		eof         error
	)
	for eof != io.EOF {
		if read, eof = arc.loadCompressedBuf(arcFile, &crc); eof != nil && eof != io.EOF {
			return errtype.ErrDecompress(ErrReadCompressed, eof)
		}

		if read > 0 {
			if err = arc.decompressBuffers(); err != nil {
				return errtype.ErrDecompress(ErrDecompress, err)
			}

			for i := 0; i < ncpu && decompressedBuf[i].Len() > 0; i++ {
				if wrote, err = decompressedBuf[i].WriteTo(outBuf); err != nil {
					return errtype.ErrCompress(ErrWriteOutBuf, err)
				}
				log.Println("Записан буфер размера:", wrote)
				decompressedBuf[i].Reset()
			}
		}
		outBuf.Flush()
	}
	fi.SetDamaged(crc != 0)

	return nil
}

// Загружает данные в буферы сжатых данных
//
// Возвращает количество прочитанных байт и ошибку.
// Если err == io.EOF, то был прочитан признак конца файла,
// новых данных для файла не будет.
func (arc Arc) loadCompressedBuf(arcBuf io.Reader, crc *uint32) (read int64, err error) {
	var n, bufferSize int64

	for i := 0; i < ncpu; i++ {
		if err = binary.Read(arcBuf, binary.LittleEndian, &bufferSize); err != nil {
			return 0, errtype.ErrDecompress(ErrReadCompLen, err)
		}

		if bufferSize == -1 {
			log.Println("Прочитан EOF")
			return read, io.EOF
		} else if arc.checkBufferSize(bufferSize) {
			return 0, errtype.ErrDecompress(ErrBufSize(bufferSize), err)
		}

		compressedBuf[i].Reset()
		if n, err = io.CopyN(compressedBuf[i], arcBuf, bufferSize); err != nil {
			return 0, errtype.ErrDecompress(ErrReadCompBuf, err)
		}
		log.Println("Прочитан блок сжатых данных размера:", bufferSize)
		*crc ^= crc32.Checksum(compressedBuf[i].Bytes(), crct)
		read += n

		if decompressor[i] != nil {
			decompressor[i].Reset(compressedBuf[i])
		} else {
			if decompressor[i], err = c.NewReader(arc.ct, compressedBuf[i]); err != nil {
				return 0, errtype.ErrDecompress(ErrDecompInit, err)
			}
		}
	}

	return read, nil
}

// Распаковывает данные в буферах сжатых данных
func (arc Arc) decompressBuffers() error {
	var (
		errChan = make(chan error, ncpu)
		wg      sync.WaitGroup
	)

	for i := 0; i < ncpu && compressedBuf[i].Len() > 0; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			defer decompressor[i].Close()
			_, err := decompressor[i].WriteTo(decompressedBuf[i])
			if err != nil {
				errChan <- errtype.ErrDecompress(ErrReadDecomp, err)
			}
		}(i)
	}

	go func() {
		wg.Wait()
		close(errChan)
	}()

	for err := range errChan {
		return err
	}

	return nil
}
