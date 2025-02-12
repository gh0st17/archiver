// Пакет generic предоставляет глобальные переменные,
// константы и функции для работы с ними
package generic

import (
	"bytes"
	"hash/crc32"
	"io"
	"log"
	"os"
	"runtime"

	"github.com/gh0st17/archiver/arc/internal/header"
	c "github.com/gh0st17/archiver/compressor"
	"github.com/gh0st17/archiver/errtype"
	"github.com/gh0st17/archiver/filesystem"
)

type RestoreParams struct {
	OutputDir string
	DictPath  string
	Patterns  []string
	Integ     bool
	Ct        c.Type  // Тип компрессора
	Cl        c.Level // Уровень сжатия
	// Флаг замены файлов без подтверждения
	ReplaceAll *bool
}

// Базовый размер буфера
// для операции ввода вывода
const BufferSize int = 262144 // 256K

var (
	// Полином CRC32
	crct = crc32.MakeTable(crc32.Koopman)
	// Количество доступных процессоров
	ncpu = runtime.NumCPU()
	// Буффер для сжатых данных
	compressedBufs = make([]*bytes.Buffer, ncpu)
	// Буфер для несжатых данных
	decompressedBufs = make([]*bytes.Buffer, ncpu)
	compressors      = make([]*c.Writer, ncpu)
	decompressors    = make([]*c.Reader, ncpu)
	writeBuf         *bytes.Buffer
	dict             []byte
)

func Ncpu() int                      { return ncpu }
func CompBuffers() []*bytes.Buffer   { return compressedBufs }
func DecompBuffers() []*bytes.Buffer { return decompressedBufs }
func Compressors() []*c.Writer       { return compressors }
func Decompressors() []*c.Reader     { return decompressors }

func WriteBuffer() *bytes.Buffer { return writeBuf }
func Dict() []byte               { return dict }

func Checksum(data []byte) uint32 { return crc32.Checksum(data, crct) }

// Сбрасывает буфер данных для записи в w
func FlushWriteBuffer(w io.Writer) {
	if writeBuf.Len() == 0 {
		return
	}

	wrote, err := writeBuf.WriteTo(w)
	if err != nil {
		errtype.ErrorHandler(errtype.Join(ErrFlushWrBuf, err))
	}
	log.Println("Буфер записи сброшен в писателя:", wrote)
}

// Проверяет корректность размера буфера.
// Возвращает true если размер некорректный.
func CheckBufferSize(bufferSize int64) bool {
	return bufferSize < 0 || bufferSize>>1 > bufferSize
}

// Инициализирует компрессоры
func InitCompressors(rp RestoreParams) (err error) {
	if err = LoadDict(rp); err != nil {
		return err
	}

	for i := 0; i < ncpu; i++ { // Инициализация компрессоров
		compressors[i], err = c.NewWriterDict(rp.Ct, dict, compressedBufs[i], rp.Cl)
		if err != nil {
			return err
		}
	}

	return nil
}

// Загружает файл словаря в байтовый срез
func LoadDict(rp RestoreParams) (err error) {
	if rp.DictPath != "" {
		if dict, err = os.ReadFile(rp.DictPath); err != nil {
			return errtype.Join(ErrReadDict, err)
		}
	}

	return nil
}

// Сбрасывает декомпрессоры
func ResetDecomp() {
	for i := 0; i < ncpu; i++ {
		decompressors[i] = nil
	}
}

// Прототип функции-обработчика заголовков
type ProcHeaderHandler = func(header.HeaderType, io.ReadSeeker) error

// Универсальная функция обработки заголовков из arcFile
func ProcessHeaders(arcFile io.ReadSeeker, handler ProcHeaderHandler) error {
	var typ header.HeaderType

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
		compressedBufs[i] = bytes.NewBuffer(nil)
		decompressedBufs[i] = bytes.NewBuffer(nil)
	}

	writeBuf = bytes.NewBuffer(nil)
}
