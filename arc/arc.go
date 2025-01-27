package arc

import (
	c "archiver/compressor"
	"archiver/errtype"
	"archiver/filesystem"
	"archiver/params"
	"bytes"
	"fmt"
	"hash/crc32"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

const (
	magicNumber uint16 = 0x5717
	bufferSize  int64  = 1048576 // 1М
)

var (
	// Полином CRC32
	crct = crc32.MakeTable(crc32.Koopman)
	// Количество доступных процессоров
	ncpu = runtime.NumCPU()
	// Буффер для сжатых данных
	compressedBuf = make([]*bytes.Buffer, ncpu)
	// Буфер для несжатых данных
	decompressedBuf = make([]*bytes.Buffer, ncpu)
	compressor      = make([]*c.Writer, ncpu)
	decompressor    = make([]*c.Reader, ncpu)
	writeBuf        *bytes.Buffer
	writeBufSize    int
)

// Структура параметров архива
type Arc struct {
	arcPath   string // Путь к файлу архива
	outputDir string
	integ     bool
	ct        c.Type  // Тип компрессора
	cl        c.Level // Уровень сжатия
	// Флаг замены файлов без подтверждения
	replaceAll bool
}

// Возвращает новый Arc из входных параметров программы
func NewArc(p params.Params) (arc *Arc, err error) {
	arc = &Arc{
		arcPath:    p.ArcPath,
		replaceAll: p.ReplaceAll,
	}

	if filesystem.DirExists(arc.arcPath) {
		return nil, errIsDir(filepath.Base(arc.arcPath))
	}

	if len(p.InputPaths) > 0 {
		arc.ct = p.Ct
		arc.cl = p.Cl
	} else {
		arcFile, err := os.Open(arc.arcPath)
		if err != nil {
			return nil, errtype.Join(ErrOpenArc, err)
		}
		defer arcFile.Close()

		var magic uint16
		if err = filesystem.BinaryRead(arcFile, &magic); err != nil {
			return nil, errtype.Join(ErrReadMagic, err)
		}
		if magic != magicNumber {
			return nil, errNotArc(arcFile.Name())
		}

		var compType byte
		if err = filesystem.BinaryRead(arcFile, &compType); err != nil {
			return nil, err
		}

		if compType <= byte(c.ZLib) {
			arc.ct = c.Type(compType)
		} else {
			return nil, ErrUnknownComp
		}

		arc.integ = p.XIntegTest
		arc.outputDir = p.OutputDir
	}

	return arc, nil
}

// Удаляет архив
func (arc Arc) RemoveTmp() {
	os.Remove(arc.arcPath)
}

// Закрывает файл архива и удаляет его
func (arc Arc) closeRemove(arcFile io.Closer) {
	arcFile.Close()
	arc.RemoveTmp()
}

// Печать статистики использования памяти
func (Arc) PrintMemStat() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	fmt.Printf("\nАллоцированная память: %8d KB\n", m.Alloc/1024)
	fmt.Printf("Всего аллокаций:       %8d KB\n", m.TotalAlloc/1024)
	fmt.Printf("Системная память:      %8d KB\n", m.Sys/1024)
	fmt.Printf("Количество сборок мусора: %d\n", m.NumGC)
}

// Проверяет корректность размера буфера.
// Возвращает true если размер некорректный.
func (Arc) checkBufferSize(bufferSize int64) bool {
	return bufferSize < 0 || bufferSize>>1 > bufferSize
}

// Сбрасывает буфер данных для записи на диск
func (Arc) flushWriteBuffer(wg *sync.WaitGroup, w io.Writer) {
	defer wg.Done()

	if writeBuf.Len() == 0 {
		return
	}

	wrote, err := writeBuf.WriteTo(w)
	if err != nil {
		errtype.ErrorHandler(errtype.Join(ErrFlushWrBuf, err))
	}
	log.Println("Буфер записи сброшен на диск:", wrote)
}

func init() {
	for i := 0; i < ncpu; i++ {
		compressedBuf[i] = bytes.NewBuffer(nil)
		decompressedBuf[i] = bytes.NewBuffer(nil)
	}
}
