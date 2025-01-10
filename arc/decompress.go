package arc

import (
	"archiver/arc/header"
	c "archiver/compressor"
	"archiver/errtype"
	"archiver/filesystem"
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
)

func (arc Arc) prepareArcFile() (arcFile *os.File, err error) {
	arcFile, err = os.Open(arc.arcPath)
	if err != nil {
		return nil, errtype.ErrDecompress(
			"ошибка при открытии архива", err,
		)
	}

	pos, err := arcFile.Seek(arc.dataOffset, io.SeekStart)
	if err != nil {
		return nil, errtype.ErrDecompress(
			"ошибка установки в позицию смещения данных", err,
		)
	}
	log.Println("Позиция установлена:", pos)

	return arcFile, nil
}

// Распаковывает архив
func (arc Arc) Decompress(outputDir string, integ bool) error {
	headers, err := arc.readHeaders()
	if err != nil {
		return errtype.ErrDecompress("ошибка чтения заголовоков", err)
	}

	arcFile, err := arc.prepareArcFile()
	if err != nil {
		return err
	}
	defer arcFile.Close()

	// 	Создаем файлы и директории
	var (
		outPath          string
		skipLen, dataPos int64
		firstFileIdx     int
	)

	if firstFileIdx, err = arc.findFileIdx(headers, outputDir); err != nil {
		return err
	}

	if firstFileIdx >= len(headers) { // В архиве нет файлов
		return nil
	}

	for _, h := range headers[firstFileIdx:] {
		fi := h.(*header.FileItem)
		outPath = filepath.Join(outputDir, fi.Path())
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
			return err
		}

		if dataPos, err = arcFile.Seek(4, io.SeekCurrent); err != nil {
			return err
		}
		log.Println("Skipped CRC, new arcFile pos:", dataPos)

		if err = os.Chtimes(outPath, fi.Atim(), fi.Mtim()); err != nil {
			return err
		}

		if fi.IsDamaged() {
			fmt.Printf("%s: CRC сумма не совпадает\n", outPath)
		} else {
			fmt.Println(outPath)
		}
	}

	return nil
}

// Возвращает первый индекс, указывающий на файл.
// Воссоздает пути для распаковки если они не существуют.
func (Arc) findFileIdx(headers []header.Header, outputDir string) (int, error) {
	var (
		di           *header.DirItem
		ok           bool
		firstFileIdx int
	)

	for {
		if di, ok = headers[firstFileIdx].(*header.DirItem); !ok {
			break
		}

		outPath := filepath.Join(outputDir, di.Path())
		if _, err := os.Stat(outPath); errors.Is(err, os.ErrNotExist) {
			if err = filesystem.CreatePath(outPath); err != nil {
				return 0, errtype.ErrDecompress(
					fmt.Sprintf("не могу создать путь до директории '%s'", outPath), err,
				)
			}
			if err = os.Chtimes(outPath, di.Atim(), di.Mtim()); err != nil {
				return 0, errtype.ErrDecompress(
					fmt.Sprintf(
						"не могу установить аттрибуты времени для директории '%s'", outPath,
					), err,
				)
			}
		}

		fmt.Println(outPath)
		firstFileIdx++
		if firstFileIdx >= len(headers) {
			break
		}
	}

	return firstFileIdx, nil
}

// Обрабатывает вопрос замены файла
func (arc Arc) replaceInput(outPath string, arcFile io.ReadSeeker) bool {
	var input rune
	stdin := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("Файл '%s' существует, заменить? [(Д)а/(Н)ет/(В)се]: ", outPath)
		input, _, _ = stdin.ReadRune()

		switch input {
		case 'A', 'a', 'В', 'в':
			arc.replaceAll = true
			return false
		case 'Y', 'y', 'Д', 'д':
		case 'N', 'n', 'Н', 'н':
			arc.skipFile(arcFile)
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
		var pos int64
		if pos, err = arcFile.Seek(8, io.SeekCurrent); err != nil {
			return errtype.ErrDecompress("ошибка пропуска признака EOF", err)
		}
		log.Println("Нулевой размер, перемещаю на позицию:", pos)
		return nil
	}

	var (
		n   int64
		crc = fi.CRC()
	)
	for n != -1 {
		pos, _ := arcFile.Seek(0, io.SeekCurrent)
		log.Println("Чтение блоков с позиции:", pos)
		if n, err = arc.loadCompressedBuf(arcFile); err != nil {
			return errtype.ErrDecompress("ошибка чтения сжатых блоков", err)
		}

		for i := 0; i < ncpu && compressedBuf[i].Len() > 0; i++ {
			crc ^= crc32.Checksum(compressedBuf[i].Bytes(), crct)
		}

		if err = arc.decompressBuffers(); err != nil {
			return errtype.ErrDecompress("ошибка распаковки буферов", err)
		}

		for i := 0; i < ncpu && decompressedBuf[i].Len() > 0; i++ {
			decompressedBuf[i].WriteTo(outFile)
		}
	}
	fi.SetDamaged(crc != 0)

	return nil
}

// Загружает данные в буферы сжатых данных
func (Arc) loadCompressedBuf(r io.Reader) (read int64, err error) {
	var n, bufferSize int64

	for i := 0; i < ncpu; i++ {
		if err = binary.Read(r, binary.LittleEndian, &bufferSize); err != nil {
			return 0, errtype.ErrDecompress("не могу прочитать размер блок сжатых данных", err)
		}

		if bufferSize == -1 {
			log.Println("Прочитан EOF")
			return -1, nil
		}
		log.Println("Прочитан блок сжатых данных:", bufferSize)

		lim := io.LimitReader(r, bufferSize)
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

			decompressor := c.NewReader(arc.ct, compressedBuf[i])
			defer decompressor.Close()
			_, err := decompressedBuf[i].ReadFrom(decompressor)
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
