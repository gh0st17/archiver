package compressor

import (
	"io"
)

type NopCompressor struct{}

type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error {
	return nil
}
