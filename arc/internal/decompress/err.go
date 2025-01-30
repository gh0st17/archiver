package decompress

import "archiver/arc/internal/errors"

// Ошибки при распаковке
var (
	ErrReadHeaders   = errors.ErrReadHeaders
	ErrDecompressSym = errors.ErrDecompressSym
	ErrSkipCRC       = errors.ErrSkipCRC
	ErrCreateOutFile = errors.ErrCreateOutFile
	ErrSkipEofCrc    = errors.ErrSkipEofCrc
	ErrDecompress    = errors.ErrDecompress
	ErrWriteOutBuf   = errors.ErrWriteOutBuf
	ErrReadCompLen   = errors.ErrReadCompLen
	ErrReadCompBuf   = errors.ErrReadCompBuf
	ErrDecompInit    = errors.ErrDecompInit
	ErrReadDecomp    = errors.ErrReadDecomp
	ErrRestorePath   = errors.ErrRestorePath
	ErrBufSize       = errors.ErrBufSize
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
