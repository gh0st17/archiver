package arc

import "github.com/gh0st17/archiver/arc/internal/errors"

// Общие ошибки
var (
	ErrTerminalWidth = errors.ErrTerminalWidth
	ErrCloseFile     = errors.ErrCloseFile
	ErrSeek          = errors.ErrSeek
)

// Ошибки при открытии архива
var (
	ErrIsDir       = errors.ErrIsDir
	ErrNotArc      = errors.ErrNotArc
	ErrUnknownComp = errors.ErrUnknownComp
)

// Ошибки при сжатии
var (
	ErrCompressorInit  = errors.ErrCompressorInit
	ErrWriteArcHeaders = errors.ErrWriteArcHeaders
)

// Ошибки при распаковке
var (
	ErrReadHeaders    = errors.ErrReadHeaders
	ErrDecompressFile = errors.ErrDecompressFile
	ErrDecompressSym  = errors.ErrDecompressSym
)

// Ошибки проверки целостности
var (
	ErrCheckFile = errors.ErrCheckFile
	ErrCheckCRC  = errors.ErrCheckCRC
	ErrWrongCRC  = errors.ErrWrongCRC
)

// Ошибки функции чтения
var (
	ErrOpenArc        = errors.ErrOpenArc
	ErrReadMagic      = errors.ErrReadMagic
	ErrReadFileHeader = errors.ErrReadFileHeader
	ErrReadSymHeader  = errors.ErrReadSymHeader
	ErrReadHeaderType = errors.ErrReadHeaderType
	ErrHeaderType     = errors.ErrHeaderType
)

// Ошибки функции записи
var (
	ErrCreateArc     = errors.ErrCreateArc
	ErrWriteMagic    = errors.ErrWriteMagic
	ErrWriteCompType = errors.ErrWriteCompType
)
