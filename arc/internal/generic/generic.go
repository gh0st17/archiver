package generic

import (
	c "archiver/compressor"
	"archiver/errtype"
	"bytes"
	"hash/crc32"
	"io"
	"log"
	"runtime"
	"sync"
)

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

func InitCompressors(ct c.Type, cl c.Level) (err error) {
	for i := 0; i < ncpu; i++ { // Инициализация компрессоров
		compressor[i], err = c.NewWriter(ct, compressedBuf[i], cl)
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

func init() {
	for i := 0; i < ncpu; i++ {
		compressedBuf[i] = bytes.NewBuffer(nil)
		decompressedBuf[i] = bytes.NewBuffer(nil)
	}
}
