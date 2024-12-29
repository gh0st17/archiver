package compressor

import (
	"bufio"
	"compress/zlib"
	"io"
)

type Zlib struct {
	compLevel Level
}

func NewZlib() Compressor                 { return &Zlib{DefaultCompression} }
func NewZlibLevel(level Level) Compressor { return &Zlib{level} }

func (Zlib) Read(r io.Reader, w io.Writer) error {
	zl, err := zlib.NewReader(r)
	if err != nil {
		return err
	}
	defer zl.Close()

	if _, err = io.Copy(w, zl); err != nil {
		return err
	}

	if err := zl.Close(); err != nil {
		return err
	}

	return nil
}

func (z Zlib) Write(w io.Writer, r io.Reader) error {
	zl, err := zlib.NewWriterLevel(w, int(z.compLevel))
	if err != nil {
		return err
	}

	reader := bufio.NewReader(r)
	if _, err = io.Copy(zl, reader); err != nil {
		return err
	}

	if err := zl.Close(); err != nil {
		return err
	}

	return nil
}
