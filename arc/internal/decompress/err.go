package decompress

import "github.com/gh0st17/archiver/arc/internal/errors"

// Ошибки при распаковке
var (
	ErrReadHeaders   = errors.ErrReadHeaders
	ErrDecompressSym = errors.ErrDecompressSym
	ErrSeek          = errors.ErrSeek
	ErrCreateOutFile = errors.ErrCreateOutFile
	ErrDecompress    = errors.ErrDecompress
	ErrWriteOutBuf   = errors.ErrWriteOutBuf
	ErrReadCompLen   = errors.ErrReadCompLen
	ErrReadCompBuf   = errors.ErrReadCompBuf
	ErrDecompInit    = errors.ErrDecompInit
	ErrReadDecomp    = errors.ErrReadDecomp
	ErrRestorePath   = errors.ErrRestorePath
	ErrRestoreTime   = errors.ErrRestoreTime
	ErrBufSize       = errors.ErrBufSize
	ErrCheckCRC      = errors.ErrCheckCRC
)

// Ошибки функции чтения
var (
	ErrReadCompressed = errors.ErrReadCompressed
	ErrReadFileHeader = errors.ErrReadFileHeader
	ErrReadSymHeader  = errors.ErrReadSymHeader
	ErrReadCRC        = errors.ErrReadCRC
	ErrSkipData       = errors.ErrSkipData
	ErrReadHeaderType = errors.ErrReadHeaderType
	ErrHeaderType     = errors.ErrHeaderType
	ErrWrongCRC       = errors.ErrWrongCRC
)
