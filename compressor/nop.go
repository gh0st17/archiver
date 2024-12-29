package compressor

import (
	"io"
)

type NopCompressor struct{}

func NewNop() Compressor { return &NopCompressor{} }

func (NopCompressor) NewReader(r io.Reader) (io.ReadCloser, error) {
	return io.NopCloser(r), nil
}

func (NopCompressor) NewWriter(w io.Writer) (io.WriteCloser, error) {
	return nopWriteCloser{Writer: w}, nil
}

type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error {
	return nil
}
