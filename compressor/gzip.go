package compressor

import (
	"compress/gzip"
	"io"
)

type GZ struct {
	compLevel Level
}

func NewGz() Compressor                 { return &GZ{DefaultCompression} }
func NewGzLevel(level Level) Compressor { return &GZ{level} }

func (GZ) NewReader(r io.Reader) (io.ReadCloser, error) {
	return gzip.NewReader(r)
}

func (gz GZ) NewWriter(w io.Writer) (io.WriteCloser, error) {
	return gzip.NewWriterLevel(w, int(gz.compLevel))
}
