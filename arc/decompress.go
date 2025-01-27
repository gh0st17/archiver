package arc

import (
	"archiver/arc/header"
	c "archiver/compressor"
	"archiver/errtype"
	"archiver/filesystem"
	"bufio"
	"bytes"
	"fmt"
	"hash/crc32"
	"io"
	"log"
	"os"
	fp "path/filepath"
	"sync"
)

// Распаковывает архив
func (arc Arc) Decompress() error {
	arcFile, err := os.OpenFile(arc.arcPath, os.O_RDONLY, 0644)
	if err != nil {
		return errtype.ErrDecompress(
			errtype.Join(ErrOpenArc, err),
		)
	}
	arcFile.Seek(3, io.SeekStart) // Пропускаем магическое число и тип компрессора

	writeBufSize = int(bufferSize) * ncpu
	writeBuf = bytes.NewBuffer(make([]byte, 0, writeBufSize))

	var typ header.HeaderType

	for err != io.EOF {
		err = filesystem.BinaryRead(arcFile, &typ)
		if err != io.EOF && err != nil {
			return errtype.ErrDecompress(
				errtype.Join(ErrReadHeaderType, err),
			)
		} else if err == io.EOF {
			continue
		}

		switch typ {
		case header.File:
			if err = arc.restoreFile(arcFile); err != nil && err != io.EOF {
				return errtype.ErrDecompress(
					errtype.Join(ErrDecompressFile, err),
				)
			}
		case header.Symlink:
			if err = arc.restoreSym(arcFile); err != nil && err != io.EOF {
				return errtype.ErrDecompress(
					errtype.Join(ErrDecompressSym, err),
				)
			}
		default:
			return errtype.ErrDecompress(ErrHeaderType)
		}
	}

	// Сброс декомпрессоров перед новым использованием этой функции
	for i := 0; i < ncpu; i++ {
		decompressor[i] = nil
	}

	return nil
}

func (arc *Arc) restoreFile(arcFile io.ReadSeeker) error {
	fi := &header.FileItem{}
	err := fi.Read(arcFile)
	if err != nil && err != io.EOF {
		return errtype.Join(ErrReadFileHeader, err)
	}

	if err = fi.RestorePath(arc.outputDir); err != nil {
		return errtype.Join(ErrRestorePath(fi.PathOnDisk()), err)
	}

	outPath := fp.Join(arc.outputDir, fi.PathOnDisk())
	if _, err = os.Stat(outPath); err == nil && !arc.replaceAll {
		if arc.replaceInput(outPath, arcFile) {
			return nil
		}
	}

	if arc.integ { // --xinteg
		_, err = arc.checkCRC(arcFile)
		if err == ErrWrongCRC {
			fmt.Printf("Пропускаю поврежденный '%s'\n", fi.PathOnDisk())
			return nil
		}
	}

	if err = arc.decompressFile(fi, arcFile, outPath); err != nil {
		return errtype.Join(ErrDecompressFile, err)
	}

	if fi.IsDamaged() {
		fmt.Printf("%s: CRC сумма не совпадает\n", outPath)
	} else {
		fmt.Println(outPath)
	}

	return fi.RestoreTime(arc.outputDir)
}

func (arc Arc) restoreSym(arcFile io.ReadSeeker) error {
	sym := &header.SymItem{}

	err := sym.Read(arcFile)
	if err != nil {
		return errtype.Join(ErrReadSymHeader, err)
	}

	if err = sym.RestorePath(arc.outputDir); err != nil {
		return errtype.Join(
			ErrRestorePath(fp.Join(arc.outputDir, sym.PathOnDisk())),
		)
	}

	fmt.Println(sym.PathInArc(), "->", sym.PathOnDisk())

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
		return errtype.Join(ErrCreateOutFile, err)
	}
	defer outFile.Close()

	// Если размер файла равен 0, то пропускаем запись
	if fi.UcSize() == 0 {
		if pos, err := arcFile.Seek(12, io.SeekCurrent); err != nil {
			return errtype.Join(ErrSkipEofCrc, err)
		} else {
			log.Println("Нулевой размер, перемещаю на позицию:", pos)
			return nil
		}
	}

	var (
		wrote, read int64
		calcCRC     uint32
		fileCRC     uint32
		eof         error
		wg          = sync.WaitGroup{}
	)

	outBuf := bufio.NewWriter(outFile)
	for eof != io.EOF {
		if read, eof = arc.loadCompressedBuf(arcFile, &calcCRC); eof != nil && eof != io.EOF {
			return errtype.Join(ErrReadCompressed, eof)
		}

		if read > 0 {
			if err = arc.decompressBuffers(); err != nil {
				return errtype.Join(ErrDecompress, err)
			}

			wg.Wait()

			for i := 0; i < ncpu && decompressedBuf[i].Len() > 0; i++ {
				if wrote, err = decompressedBuf[i].WriteTo(writeBuf); err != nil {
					return errtype.Join(ErrWriteOutBuf, err)
				}
				log.Println("В буфер записи записан блок размера:", wrote)
			}
		}

		if writeBuf.Len() >= writeBufSize || eof == io.EOF {
			wg.Add(1)
			go arc.flushWriteBuffer(&wg, outBuf)
		}
	}
	wg.Wait()

	err = filesystem.BinaryRead(arcFile, &fileCRC)
	if err != nil {
		return errtype.Join(ErrReadCRC, err)
	}
	fi.SetDamaged(calcCRC != fileCRC)

	outBuf.Flush()

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
		if err = filesystem.BinaryRead(arcBuf, &bufferSize); err != nil {
			return 0, errtype.Join(ErrReadCompLen, err)
		}

		if bufferSize == -1 {
			log.Println("Прочитан EOF")
			return read, io.EOF
		} else if arc.checkBufferSize(bufferSize) {
			return 0, errtype.Join(ErrBufSize(bufferSize), err)
		}

		if n, err = io.CopyN(compressedBuf[i], arcBuf, bufferSize); err != nil {
			return 0, errtype.Join(ErrReadCompBuf, err)
		}
		log.Println("Прочитан блок сжатых данных размера:", bufferSize)
		*crc ^= crc32.Checksum(compressedBuf[i].Bytes(), crct)
		read += n

		if decompressor[i] != nil {
			decompressor[i].Reset(compressedBuf[i])
		} else {
			if decompressor[i], err = c.NewReader(arc.ct, compressedBuf[i]); err != nil {
				return 0, errtype.Join(ErrDecompInit, err)
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
			_, err := decompressedBuf[i].ReadFrom(decompressor[i])
			if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
				errChan <- errtype.Join(ErrReadDecomp, err)
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
