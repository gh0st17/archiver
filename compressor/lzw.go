package compressor

import (
	"compress/lzw"
	"io"
)

type LZW struct{}

func NewLZW() Compressor { return &LZW{} }

func (LZW) NewReader(r io.Reader) (io.ReadCloser, error) {
	return lzw.NewReader(r, lzw.MSB, 8), nil
}

func (LZW) NewWriter(w io.Writer) (io.WriteCloser, error) {
	return lzw.NewWriter(w, lzw.MSB, 8), nil
}
