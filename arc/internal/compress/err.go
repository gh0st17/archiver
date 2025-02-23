package compress

import "github.com/gh0st17/archiver/arc/internal/errors"

// Ошибки при сжатии
var (
	ErrNoEntries         = errors.ErrNoEntries
	ErrWriteFileHeader   = errors.ErrWriteFileHeader
	ErrWriteSymHeader    = errors.ErrWriteSymHeader
	ErrCompressFile      = errors.ErrCompressFile
	ErrReadUncompressed  = errors.ErrReadUncompressed
	ErrCompress          = errors.ErrCompress
	ErrWriteBufLen       = errors.ErrWriteBufLen
	ErrWriteCompressBuf  = errors.ErrWriteCompressBuf
	ErrReadUncompressBuf = errors.ErrReadUncompressBuf
	ErrWriteEOF          = errors.ErrWriteEOF
	ErrWriteCRC          = errors.ErrWriteCRC
	ErrWriteCompressor   = errors.ErrWriteCompressor
	ErrCloseCompressor   = errors.ErrCloseCompressor
	ErrFetchDirs         = errors.ErrFetchDirs

	ErrLongPath = errors.ErrLongPath

	ErrOpenFileCompress = errors.ErrOpenFileCompress
)
