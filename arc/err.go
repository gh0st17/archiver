package arc

import (
	"errors"
	"fmt"
)

func errIsDir(path string) error {
	return fmt.Errorf("'%s' это директория", path)
}

func errNotArc(path string) error {
	return fmt.Errorf("'%s' не архив Arc", path)
}

var ErrUnknownComp = errors.New("неизвестный тип компрессора")

// Ошибки при сжатии
var (
	ErrCompressorInit    = errors.New("ошибка иницализации компрессора")
	ErrWriteDirHeaders   = errors.New("ошибка записи заголовков директории")
	ErrWriteFileHeader   = errors.New("ошибка записи заголовка файла")
	ErrCompressFile      = errors.New("ошибка сжатия файла")
	ErrReadUncompressed  = errors.New("ошибка чтения несжатых блоков")
	ErrCompress          = errors.New("ошибка сжатия буфферов")
	ErrWriteBufLen       = errors.New("ошибка записи длины блока")
	ErrReadCompressBuf   = errors.New("ошибка чтения из буфера сжатых данных")
	ErrReadUncompressBuf = errors.New("ошибка чтения в несжатый буфер")
	ErrWriteEOF          = errors.New("ошибка записи EOF (-1)")
	ErrWriteCRC          = errors.New("ошибка записи CRC")
	ErrWriteCompressor   = errors.New("ошибка записи в компрессор")
	ErrCloseCompressor   = errors.New("ошибка закрытия компрессора")
	ErrFetchDirs         = errors.New("не могу получить директории")

	ErrPathLength = func(path string) error {
		return fmt.Errorf("длина пути к '%s' первышает максимально допустимую (1023)", path)
	}

	ErrOpenFileCompress = func(path string) error {
		return fmt.Errorf("не могу открыть входной файл '%s' для сжатия", path)
	}
)

// Ошибки при распаковке
var (
	ErrReadHeaders    = errors.New("ошибка чтения заголовоков")
	ErrSkipHeaders    = errors.New("ошибка пропуска заголовков")
	ErrDecompressFile = errors.New("ошибка распаковки файла")
	ErrSkipCRC        = errors.New("ошибка пропуска CRC")
	ErrCreateOutFile  = errors.New("не могу создать файл")
	ErrSkipEOF        = errors.New("ошибка пропуска признака EOF")
	ErrReadCompressed = errors.New("ошибка чтения сжатых блоков")
	ErrDecompress     = errors.New("ошибка распаковки буферов")
	ErrWriteOutBuf    = errors.New("ошибка записи в выходной буфер")
	ErrReadCompLen    = errors.New("ошибка чтения размера блока")
	ErrReadCompBuf    = errors.New("ошибка чтения блока")
	ErrDecompInit     = errors.New("ошибка иницализации декомпрессора")
	ErrReadDecomp     = errors.New("ошибка чтения декомпрессора")

	ErrRestorePath = func(path string) error {
		return fmt.Errorf("не могу создать путь для '%s'", path)
	}

	ErrBufSize = func(bufferSize int64) error {
		return fmt.Errorf("некорректный размер (%d) блока сжатых данных", bufferSize)
	}
)

// Ошибки проверки целостности
var (
	ErrCheckFile = errors.New("ошибка проверки файла")
	ErrCheckCRC  = errors.New("ошибка проверки CRC")
	ErrWrongCRC  = errors.New("CRC сумма не совпадает")
)

// Ошибки функции чтения
var (
	ErrOpenArc          = errors.New("не могу открыть файл архива")
	ErrReadDirsSyms     = errors.New("ошибка чтения заголовка директории/ссылки")
	ErrReadFileHeader   = errors.New("ошибка чтения заголовка файла")
	ErrReadHeadersCount = errors.New("ошибка чтения числа заголовков")
	ErrReadCompSize     = errors.New("ошибка чтения размера сжатых данных")
	ErrReadCRC          = errors.New("ошибка чтения CRC")
	ErrSkipData         = errors.New("ошибка пропуска блока сжатых данных")
)

// Ошибки функции записи
var (
	ErrCreateArc         = errors.New("не могу создать файл архива")
	ErrMagic             = errors.New("ошибка записи сигнатуры")
	ErrWriteCompType     = errors.New("ошибка записи типа компрессора")
	ErrWriteHeadersCount = errors.New("ошибка записи количества заголовков")
	ErrWriteHeaderIO     = errors.New("ошибка записи заголовка директории")
)
