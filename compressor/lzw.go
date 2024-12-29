package compressor

import (
	"bufio"
	"compress/lzw"
	"io"
)

type LZW struct{}

func NewLZW() Compressor { return &LZW{} }

func (LZW) Read(r io.Reader, w io.Writer) error {
	lz := lzw.NewReader(r, lzw.MSB, 8)
	defer lz.Close()

	if _, err := io.Copy(w, lz); err != nil {
		return err
	}

	if err := lz.Close(); err != nil {
		return err
	}

	return nil
}

func (LZW) Write(w io.Writer, r io.Reader) error {
	lz := lzw.NewWriter(w, lzw.MSB, 8)

	reader := bufio.NewReader(r)
	if _, err := io.Copy(lz, reader); err != nil {
		return err
	}

	if err := lz.Close(); err != nil {
		return err
	}

	return nil
}
