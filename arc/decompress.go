package arc

import (
	"archiver/arc/header"
	c "archiver/compressor"
	"archiver/errtype"
	"archiver/filesystem"
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
		return errtype.ErrDecompress("ошибка чтения заголовоков", err)
	}
	defer arcFile.Close()

	// 	Создаем файлы и директории
	var (
		outPath          string
		skipLen, dataPos int64
	)

	for _, h := range headers {
		outPath = filepath.Join(outputDir, h.Path())
		if di, ok := h.(*header.DirItem); ok {
			if err = filesystem.CreatePath(outPath); err != nil {
				return errtype.ErrDecompress(
					fmt.Sprintf("не могу создать путь до директории '%s'", outPath), err,
				)
			}
			if err = os.Chtimes(outPath, di.Atim(), di.Mtim()); err != nil {
				return errtype.ErrDecompress(
					fmt.Sprintf(
						"не могу установить аттрибуты времени для директории '%s'", outPath,
					), err,
				)
			}

			continue
		}

		fi := h.(*header.FileItem)
		if err = filesystem.CreatePath(filepath.Dir(outPath)); err != nil {
			return errtype.ErrDecompress(
				fmt.Sprintf("не могу создать путь до файла '%s'", outPath), err,
			)
		}

		skipLen = int64(len(fi.Path())) + 32
		if dataPos, err = arcFile.Seek(skipLen, io.SeekCurrent); err != nil {
			return err
		}
		log.Println("Пропушенно", skipLen, "байт заголовка, читаю с позиции", dataPos)

		if _, err := os.Stat(outPath); err == nil && !arc.replaceAll {
			if arc.replaceInput(outPath, arcFile) {
				continue
			}
		}

		if integ { // --xinteg
			arc.checkCRC(fi, arcFile)

			if fi.IsDamaged() {
				fmt.Printf("Пропускаю повежденный '%s'\n", fi.Path())
				continue
			} else {
				arcFile.Seek(dataPos, io.SeekStart)
				log.Println("Set to pos:", dataPos+skipLen)
			}
		}

		if err = arc.decompressFile(fi, arcFile, outPath); err != nil {
			return errtype.ErrDecompress("ошибка распаковки файла", err)
		}

		if dataPos, err = arcFile.Seek(4, io.SeekCurrent); err != nil {
			return errtype.ErrDecompress("ошибка пропуска CRC", err)
		}
		log.Println("Skipped CRC, new arcFile pos:", dataPos)

		if err = os.Chtimes(outPath, fi.Atim(), fi.Mtim()); err != nil {
			return errtype.ErrDecompress(
				fmt.Sprintf(
					"не могу установить аттрибуты времени для  '%s'", outPath,
				), err,
			)
		}

		if fi.IsDamaged() {
			fmt.Printf("%s: CRC сумма не совпадает\n", outPath)
		} else {
			fmt.Println(outPath)
		}
	}

	return nil
}

// Обрабатывает вопрос замены файла
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
		return errtype.ErrDecompress("не могу создать файл для распаковки", err)
	}
	defer outFile.Close()

	// Если размер файла равен 0, то пропускаем запись
	if fi.UcSize() == 0 {
		if pos, err := arcFile.Seek(8, io.SeekCurrent); err != nil {
			return errtype.ErrDecompress("ошибка пропуска признака EOF", err)
		} else {
			log.Println("Нулевой размер, перемещаю на позицию:", pos)
			return nil
		}
	}

	var (
		read, wrote int64
		crc         = fi.CRC()
	)
	for read != -1 {
		pos, _ := arcFile.Seek(0, io.SeekCurrent)
		log.Println("Чтение блоков с позиции:", pos)
		if read, err = arc.loadCompressedBuf(arcFile); err != nil {
			return errtype.ErrDecompress("ошибка чтения сжатых блоков", err)
		}
		log.Println("Загружено сжатых буферов размера:", read)

		for i := 0; i < ncpu && compressedBuf[i].Len() > 0; i++ {
			crc ^= crc32.Checksum(compressedBuf[i].Bytes(), crct)
		}

		if err = arc.decompressBuffers(); err != nil {
			return errtype.ErrDecompress("ошибка распаковки буферов", err)
		}

		for i := 0; i < ncpu && decompressedBuf[i].Len() > 0; i++ {
			if _, err = decompressedBuf[i].WriteTo(writeBuf); err != nil {
				return errtype.ErrCompress("ошибка записи в буфера", err)
			}
		}

		if (writeBuf.Len() >= int(c.BufferSize)) || read == -1 {
			if wrote, err = writeBuf.WriteTo(outFile); err != nil {
				return errtype.ErrCompress("ошибка записи буфера в файл архива", err)
			}
			log.Println("Записан буфер размера:", wrote)
		}
	}
	fi.SetDamaged(crc != 0)

	return nil
}

// Загружает данные в буферы сжатых данных
func (Arc) loadCompressedBuf(arcFile io.Reader) (read int64, err error) {
	var n, bufferSize int64

	for i := 0; i < ncpu; i++ {
		if err = binary.Read(arcFile, binary.LittleEndian, &bufferSize); err != nil {
			return 0, errtype.ErrDecompress("не могу прочитать размер блока сжатых данных", err)
		}

		if bufferSize == -1 {
			log.Println("Прочитан EOF")
			return -1, nil
		}
		log.Println("Прочитан блок сжатых данных размера:", bufferSize)

		lim := io.LimitReader(arcFile, bufferSize)
		if n, err = compressedBuf[i].ReadFrom(lim); err != nil {
			return 0, errtype.ErrDecompress("не могу прочитать блок сжатых данных", err)
		}

		read += n
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

			decompressor, err := c.NewReader(arc.ct, compressedBuf[i])
			if err != nil {
				errChan <- errtype.ErrCompress("ошибка иницализации декомпрессора", err)
				return
			}
			defer decompressor.Close()
			_, err = decompressedBuf[i].ReadFrom(decompressor)
			if err != nil {
				errChan <- errtype.ErrDecompress("ошибка чтения декомпрессора", err)
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
