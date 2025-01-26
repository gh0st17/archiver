package arc

import (
	"archiver/arc/header"
	c "archiver/compressor"
	"fmt"
)

func errIsDir(path string) error {
	return fmt.Errorf("'%s' это директория", path)
}

func errNotArc(path string) error {
	return fmt.Errorf("'%s' не архив Arc", path)
}

var ErrUnknownComp = c.ErrUnknownComp

// Ошибки при сжатии
var (
	ErrNoEntries         = fmt.Errorf("нет элементов для сжатия")
	ErrCompressorInit    = fmt.Errorf("ошибка иницализации компрессора")
	ErrWriteDirHeaders   = fmt.Errorf("ошибка записи заголовков директории")
	ErrWriteFileHeader   = fmt.Errorf("ошибка записи заголовка файла")
	ErrCompressFile      = fmt.Errorf("ошибка сжатия файла")
	ErrReadUncompressed  = fmt.Errorf("ошибка чтения несжатых блоков")
	ErrCompress          = fmt.Errorf("ошибка сжатия буфферов")
	ErrWriteBufLen       = fmt.Errorf("ошибка записи длины блока")
	ErrReadCompressBuf   = fmt.Errorf("ошибка чтения из буфера сжатых данных")
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
	ErrSkipHeaders    = fmt.Errorf("ошибка пропуска заголовков")
	ErrDecompressFile = fmt.Errorf("ошибка распаковки файла")
	ErrSkipCRC        = fmt.Errorf("ошибка пропуска CRC")
	ErrCreateOutFile  = fmt.Errorf("не могу создать файл")
	ErrSkipEOF        = fmt.Errorf("ошибка пропуска признака EOF")
	ErrReadCompressed = fmt.Errorf("ошибка чтения сжатых блоков")
	ErrDecompress     = fmt.Errorf("ошибка распаковки буферов")
	ErrWriteOutBuf    = fmt.Errorf("ошибка записи в буфер выхода")
	ErrReadCompLen    = fmt.Errorf("ошибка чтения размера блока")
	ErrReadCompBuf    = fmt.Errorf("ошибка чтения блока")
	ErrDecompInit     = fmt.Errorf("ошибка иницализации декомпрессора")
	ErrReadDecomp     = fmt.Errorf("ошибка чтения декомпрессора")

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
	ErrOpenArc          = fmt.Errorf("не могу открыть файл архива")
	ErrReadMagic        = fmt.Errorf("ошибка чтения сигнатуры")
	ErrReadDirsSyms     = fmt.Errorf("ошибка чтения заголовка директории/ссылки")
	ErrReadFileHeader   = fmt.Errorf("ошибка чтения заголовка файла")
	ErrReadHeadersCount = fmt.Errorf("ошибка чтения числа заголовков")
	ErrReadCompSize     = fmt.Errorf("ошибка чтения размера сжатых данных")
	ErrReadCRC          = fmt.Errorf("ошибка чтения CRC")
	ErrSkipData         = fmt.Errorf("ошибка пропуска блока сжатых данных")
	ErrReadHeaderType   = fmt.Errorf("ошибка чтения типа")
	ErrHeaderType       = fmt.Errorf("неизвестный тип")
)

// Ошибки функции записи
var (
	ErrCreateArc         = fmt.Errorf("не могу создать файл архива")
	ErrWriteMagic        = fmt.Errorf("ошибка записи сигнатуры")
	ErrWriteCompType     = fmt.Errorf("ошибка записи типа компрессора")
	ErrWriteHeadersCount = fmt.Errorf("ошибка записи количества заголовков")
	ErrWriteHeaderIO     = fmt.Errorf("ошибка записи заголовка директории")
	ErrFlushWrBuf        = fmt.Errorf("ошибка сброса буфера записи на диск")
)
