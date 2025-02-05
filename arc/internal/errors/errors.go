// Пакет errors предоставляет переменные и функции
// для описания внутренних ошибок
package errors

import (
	"fmt"

	"github.com/gh0st17/archiver/arc/internal/header"
	c "github.com/gh0st17/archiver/compressor"
)

func ErrIsDir(path string) error {
	return fmt.Errorf("'%s' это директория", path)
}

func ErrNotArc(path string) error {
	return fmt.Errorf("'%s' не архив Arc", path)
}

// Общие ошибки
var (
	ErrUnknownComp   = c.ErrUnknownComp
	ErrTerminalWidth = func(needExtra int) error {
		return fmt.Errorf(
			`недостаточная ширина терминала для вывода статистики, `+
				`увеличьте ширину терминала на %d столбцов`,
			needExtra,
		)
	}
	ErrCloseFile = fmt.Errorf("ошибка закрытия файла")
	ErrSeek      = fmt.Errorf("ошибка чтения/установки позиции")
)

// Ошибки при сжатии
var (
	ErrNoEntries         = fmt.Errorf("нет элементов для сжатия")
	ErrCompressorInit    = fmt.Errorf("ошибка иницализации компрессора")
	ErrWriteArcHeaders   = fmt.Errorf("ошибка записи заголовка архива")
	ErrWriteFileHeader   = fmt.Errorf("ошибка записи заголовка файла")
	ErrWriteSymHeader    = fmt.Errorf("ошибка записи заголовка символической ссылки")
	ErrCompressFile      = fmt.Errorf("ошибка сжатия файла")
	ErrReadUncompressed  = fmt.Errorf("ошибка чтения несжатых блоков")
	ErrCompress          = fmt.Errorf("ошибка сжатия буфферов")
	ErrWriteBufLen       = fmt.Errorf("ошибка записи длины блока")
	ErrWriteCompressBuf  = fmt.Errorf("ошибка чтения из буфера сжатых данных")
	ErrReadUncompressBuf = fmt.Errorf("ошибка чтения в несжатый буфер")
	ErrWriteEOF          = fmt.Errorf("ошибка записи EOF (-1)")
	ErrWriteCRC          = fmt.Errorf("ошибка записи CRC")
	ErrWriteCompressor   = fmt.Errorf("ошибка записи в компрессор")
	ErrCloseCompressor   = fmt.Errorf("ошибка закрытия компрессора")
	ErrFetchDirs         = fmt.Errorf("не могу получить директории")

	ErrLongPath = header.ErrLongPath

	ErrOpenFileCompress = func(path string) error {
		return fmt.Errorf("не могу открыть входной файл '%s' для сжатия", path)
	}
)

// Ошибки при распаковке
var (
	ErrReadHeaders    = fmt.Errorf("ошибка чтения заголовоков")
	ErrDecompressFile = fmt.Errorf("ошибка распаковки файла")
	ErrDecompressSym  = fmt.Errorf("ошибка распаковки символьной ссылки")
	ErrCreateOutFile  = fmt.Errorf("не могу создать файл")
	ErrDecompress     = fmt.Errorf("ошибка распаковки буферов")
	ErrWriteOutBuf    = fmt.Errorf("ошибка записи в буфер выхода")
	ErrReadCompLen    = fmt.Errorf("ошибка чтения размера блока")
	ErrReadCompBuf    = fmt.Errorf("ошибка чтения блока")
	ErrDecompInit     = fmt.Errorf("ошибка иницализации декомпрессора")
	ErrReadDecomp     = fmt.Errorf("ошибка чтения декомпрессора")
	ErrRestoreTime    = fmt.Errorf("ошибка восставновления времени")

	ErrRestorePath = func(path string) error {
		return fmt.Errorf("не могу создать путь для '%s'", path)
	}

	ErrBufSize = func(bufferSize int64) error {
		return fmt.Errorf("некорректный размер (%d) блока сжатых данных", bufferSize)
	}
)

// Ошибки проверки целостности
var (
	ErrCheckFile = fmt.Errorf("ошибка проверки файла")
	ErrCheckCRC  = fmt.Errorf("ошибка проверки CRC")
	ErrWrongCRC  = fmt.Errorf("CRC сумма не совпадает")
)

// Ошибки функции чтения
var (
	ErrOpenArc        = fmt.Errorf("не могу открыть файл архива")
	ErrReadMagic      = fmt.Errorf("ошибка чтения сигнатуры")
	ErrReadCompressed = fmt.Errorf("ошибка чтения сжатых блоков")
	ErrReadFileHeader = fmt.Errorf("ошибка чтения заголовка файла")
	ErrReadSymHeader  = fmt.Errorf("ошибка чтения заголовка символьной ссылки")
	ErrReadCompSize   = fmt.Errorf("ошибка чтения размера сжатых данных")
	ErrReadCRC        = fmt.Errorf("ошибка чтения CRC")
	ErrSkipData       = fmt.Errorf("ошибка пропуска блока сжатых данных")
	ErrReadHeaderType = fmt.Errorf("ошибка чтения типа")
	ErrHeaderType     = fmt.Errorf("неизвестный тип")
	ErrReadDict       = fmt.Errorf("ошибка чтения словаря")
)

// Ошибки функции записи
var (
	ErrCreateArc     = fmt.Errorf("не могу создать файл архива")
	ErrWriteMagic    = fmt.Errorf("ошибка записи сигнатуры")
	ErrWriteCompType = fmt.Errorf("ошибка записи типа компрессора")
	ErrFlushWrBuf    = fmt.Errorf("ошибка сброса буфера записи на диск")
)
