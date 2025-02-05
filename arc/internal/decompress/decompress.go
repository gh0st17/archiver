// Пакет decompress предоставляет функции для распаковки
//
// Основные функции:
//   - RestoreFile: Восстанавливает файл из архива
//   - RestoreSym: Восстанавливает символьную ссылку
package decompress

import (
	"archiver/arc/internal/generic"
	"archiver/arc/internal/header"
	"archiver/arc/internal/userinput"
	c "archiver/compressor"
	"archiver/errtype"
	"archiver/filesystem"
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	fp "path/filepath"
	"sync"
)

// Восстанавливает файл из архива.
//
// Читает заголовок файла, проверяет
// его целостность (CRC), определяет путь для восстановления,
// а затем либо декомпрессирует файл, либо пропускает его в
// случае повреждений. Также обрабатывает сценарии замены уже
// существующих файлов.
func RestoreFile(arcFile io.ReadSeeker, rp generic.RestoreParams, verbose bool) error {
	fi := &header.FileItem{}
	err := fi.Read(arcFile)
	if err != nil && err != io.EOF {
		return errtype.Join(ErrReadFileHeader, err)
	}

	if err = fi.RestorePath(rp.OutputDir); err != nil {
		return errtype.Join(ErrRestorePath(fi.PathOnDisk()), err)
	}

	outPath := fp.Join(rp.OutputDir, fi.PathOnDisk())
	if _, err = os.Stat(outPath); err == nil && !*rp.ReplaceAll {
		allFunc := func() {
			*rp.ReplaceAll = true
		}
		negFunc := func() {
			skipFileData(arcFile, true)
		}

		if userinput.ReplacePrompt(outPath, allFunc, negFunc) {
			return nil
		}
	}

	if rp.Integ { // --xinteg
		pos, _ := arcFile.Seek(0, io.SeekCurrent)
		if _, err = CheckCRC(arcFile, rp.Ct); err == ErrWrongCRC {
			fmt.Printf("Пропускаю поврежденный '%s'\n", fi.PathOnDisk())
			return nil
		} else if err != nil {
			return errtype.Join(ErrCheckCRC, err)
		}
		arcFile.Seek(pos, io.SeekStart)
	}

	if err = decompressFile(fi, arcFile, outPath, rp.Ct); err != nil {
		return err
	}

	if fi.IsDamaged() {
		fmt.Printf("%s: CRC сумма не совпадает\n", outPath)
	} else if verbose {
		fmt.Println(outPath)
	}

	if err = fi.RestoreTime(rp.OutputDir); err != nil {
		return errtype.Join(ErrRestoreTime, err)
	}

	return nil
}

// Восстанавливает символьную ссылку
func RestoreSym(arcFile io.Reader, rp generic.RestoreParams, verbose bool) error {
	sym := &header.SymItem{}

	err := sym.Read(arcFile)
	if err != nil {
		return errtype.Join(ErrReadSymHeader, err)
	}

	if err = sym.RestorePath(rp.OutputDir); err != nil {
		return errtype.Join(
			ErrRestorePath(fp.Join(rp.OutputDir, sym.PathOnDisk())),
		)
	}

	if verbose {
		fmt.Println(sym.PathInArc(), "->", sym.PathOnDisk())
	}

	return nil
}

// Распаковывает файл
func decompressFile(fi *header.FileItem, arcFile io.ReadSeeker, outPath string, ct c.Type) error {
	outFile, err := os.Create(outPath)
	if err != nil {
		return errtype.Join(ErrCreateOutFile, err)
	}
	defer outFile.Close()

	// Если размер файла равен 0, то пропускаем запись
	if fi.UcSize() == 0 {
		if pos, err := arcFile.Seek(12, io.SeekCurrent); err != nil {
			return errtype.Join(ErrSeek, err)
		} else {
			log.Println("Нулевой размер, перемещаю на позицию:", pos)
			return nil
		}
	}

	var (
		ncpu             = generic.Ncpu()
		decompressedBufs = generic.DecompBuffers()
		writeBuf         = generic.WriteBuffer()
		writeBufSize     = generic.BufferSize * ncpu

		wrote, read int64
		calcCRC     uint32
		fileCRC     uint32
		eof         error
		wg          = sync.WaitGroup{}
	)

	outBuf := bufio.NewWriter(outFile)
	for eof != io.EOF {
		read, eof = loadCompressedBuf(arcFile, &calcCRC, ct, false)
		if eof != nil && eof != io.EOF {
			return errtype.Join(ErrReadCompressed, eof)
		}

		if read > 0 {
			if err = decompressBuffers(); err != nil {
				return errtype.Join(ErrDecompress, err)
			}

			wg.Wait()

			for i := 0; i < ncpu && decompressedBufs[i].Len() > 0; i++ {
				if wrote, err = decompressedBufs[i].WriteTo(writeBuf); err != nil {
					return errtype.Join(ErrWriteOutBuf, err)
				}
				log.Println("В буфер записи записан блок размера:", wrote)
			}
		}

		if writeBuf.Len() >= writeBufSize || eof == io.EOF {
			wg.Add(1)
			go func() {
				defer wg.Done()
				generic.FlushWriteBuffer(outBuf)
			}()
		}
	}
	wg.Wait()

	if err = filesystem.BinaryRead(arcFile, &fileCRC); err != nil {
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
//
// Для определения длины файла без распаковки используется
// countOnly == true, благодаря чему инициализация или сброс
// декомпрессоров пропускается
func loadCompressedBuf(arcBuf io.Reader, crc *uint32, ct c.Type, countOnly bool) (read int64, err error) {
	var (
		ncpu           = generic.Ncpu()
		compressedBufs = generic.CompBuffers()
		decompressors  = generic.Decompressors()
		dict           = generic.Dict()

		n, bufferSize int64
	)

	for i := 0; i < ncpu; i++ {
		if err = filesystem.BinaryRead(arcBuf, &bufferSize); err != nil {
			return 0, errtype.Join(ErrReadCompLen, err)
		}

		if bufferSize == -1 {
			log.Println("Прочитан EOF")
			return read, io.EOF
		} else if generic.CheckBufferSize(bufferSize) {
			return 0, errtype.Join(ErrBufSize(bufferSize), err)
		}

		if n, err = io.CopyN(compressedBufs[i], arcBuf, bufferSize); err != nil {
			return 0, errtype.Join(ErrReadCompBuf, err)
		}
		log.Println("Прочитан блок сжатых данных размера:", bufferSize)
		*crc ^= generic.Checksum(compressedBufs[i].Bytes())
		read += n

		if countOnly {
			continue
		}

		if decompressors[i] != nil {
			decompressors[i].Reset(compressedBufs[i])
		} else {
			if decompressors[i], err = c.NewReaderDict(ct, dict, compressedBufs[i]); err != nil {
				return 0, errtype.Join(ErrDecompInit, err)
			}
		}
	}

	return read, nil
}

// Распаковывает данные в буферах сжатых данных
func decompressBuffers() error {
	var (
		ncpu             = generic.Ncpu()
		compressedBufs   = generic.CompBuffers()
		decompressedBufs = generic.DecompBuffers()
		decompressors    = generic.Decompressors()

		errChan = make(chan error, ncpu)
		wg      sync.WaitGroup
	)

	for i := 0; i < ncpu && compressedBufs[i].Len() > 0; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			defer decompressors[i].Close()
			_, err := decompressedBufs[i].ReadFrom(decompressors[i])
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

// Считывает данные сжатого файла из arcFile, проверяет
// контрольную сумму и возвращает количество прочитанных байт
func CheckCRC(arcFile io.Reader, ct c.Type) (read header.Size, err error) {
	var (
		ncpu           = generic.Ncpu()
		compressedBufs = generic.CompBuffers()

		n       int64
		eof     error
		calcCRC uint32
		fileCRC uint32
	)

	for eof != io.EOF {
		if n, eof = loadCompressedBuf(arcFile, &calcCRC, ct, true); eof != nil && eof != io.EOF {
			return 0, errtype.Join(ErrReadCompressed, eof)
		}

		read += header.Size(n)

		for i := 0; i < ncpu && compressedBufs[i].Len() > 0; i++ {
			compressedBufs[i].Reset()
		}
	}

	if err = filesystem.BinaryRead(arcFile, &fileCRC); err != nil {
		return 0, errtype.Join(ErrReadCRC, err)
	}

	if calcCRC != fileCRC {
		return read, ErrWrongCRC
	}

	return read, nil
}
