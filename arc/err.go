package arc

import (
	"archiver/arc/internal/errors"
	c "archiver/compressor"
)

var errIsDir = errors.ErrIsDir
var errNotArc = errors.ErrNotArc
var ErrUnknownComp = c.ErrUnknownComp

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
