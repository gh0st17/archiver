package compressor

import (
	"compress/zlib"
	"io"
)

type Zlib struct {
	compLevel Level
}

func NewZlib() Compressor                 { return &Zlib{DefaultCompression} }
func NewZlibLevel(level Level) Compressor { return &Zlib{level} }

func (Zlib) NewReader(r io.Reader) (io.ReadCloser, error) {
	return zlib.NewReader(r)
}

func (z Zlib) NewWriter(w io.Writer) (io.WriteCloser, error) {
	return zlib.NewWriterLevel(w, int(z.compLevel))
}
