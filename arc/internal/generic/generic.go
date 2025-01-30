package generic

import (
	"archiver/arc/internal/header"
	c "archiver/compressor"
	"archiver/errtype"
	"archiver/filesystem"
	"bytes"
	"hash/crc32"
	"io"
	"log"
	"runtime"
	"sync"
)

type RestoreParams struct {
	OutputDir string
	Integ     bool
	Ct        c.Type  // Тип компрессора
	Cl        c.Level // Уровень сжатия
	// Флаг замены файлов без подтверждения
	ReplaceAll bool
}

const bufferSize int = 1048576 // 1М

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

func BufferSize() int { return bufferSize }

func CRCTable() *crc32.Table         { return crct }
func Ncpu() int                      { return ncpu }
func CompBuffers() []*bytes.Buffer   { return compressedBuf }
func DecompBuffers() []*bytes.Buffer { return decompressedBuf }
func Compressors() []*c.Writer       { return compressor }
func Decompressors() []*c.Reader     { return decompressor }

func WriteBuffer() *bytes.Buffer { return writeBuf }
func WriteBufSize() int          { return writeBufSize }

func SetWriteBufSize(size int) {
	writeBufSize = size
	writeBuf = bytes.NewBuffer(make([]byte, 0, writeBufSize))
}

// Сбрасывает буфер данных для записи на диск
func FlushWriteBuffer(wg *sync.WaitGroup, w io.Writer) {
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

// Проверяет корректность размера буфера.
// Возвращает true если размер некорректный.
func CheckBufferSize(bufferSize int64) bool {
	return bufferSize < 0 || bufferSize>>1 > bufferSize
}

func InitCompressors(rp RestoreParams) (err error) {
	for i := 0; i < ncpu; i++ { // Инициализация компрессоров
		compressor[i], err = c.NewWriter(rp.Ct, compressedBuf[i], rp.Cl)
		if err != nil {
			return err
		}
	}

	return nil
}

func ResetDecomp() {
	for i := 0; i < ncpu; i++ {
		decompressor[i] = nil
	}
}

type ProcHeaderHandler = func(header.HeaderType, io.ReadSeekCloser) error

func ProcessHeaders(arcFile io.ReadSeekCloser, arcLenH int64, handler ProcHeaderHandler) error {
	var typ header.HeaderType

	arcFile.Seek(arcLenH, io.SeekStart) // Перемещаемся на начало заголовков

	for {
		err := filesystem.BinaryRead(arcFile, &typ) // Читаем тип заголовка
		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}

		if err := handler(typ, arcFile); err != nil {
			return err
		}
	}
}

func init() {
	for i := 0; i < ncpu; i++ {
		compressedBuf[i] = bytes.NewBuffer(nil)
		decompressedBuf[i] = bytes.NewBuffer(nil)
	}
}
