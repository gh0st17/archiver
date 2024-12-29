package compressor

import (
	"bufio"
	"compress/gzip"
	"io"
)

type GZ struct {
	compLevel Level
}

func NewGz() Compressor                 { return &GZ{DefaultCompression} }
func NewGzLevel(level Level) Compressor { return &GZ{level} }

func (GZ) Read(r io.Reader, w io.Writer) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gz.Close()

	if _, err = io.Copy(w, gz); err != nil {
		return err
	}

	if err := gz.Close(); err != nil {
		return err
	}

	return nil
}

func (gz GZ) Write(w io.Writer, r io.Reader) error {
	gzw, err := gzip.NewWriterLevel(w, int(gz.compLevel))
	if err != nil {
		return err
	}

	reader := bufio.NewReader(r)
	if _, err = io.Copy(gzw, reader); err != nil {
		return err
	}

	if err := gzw.Close(); err != nil {
		return err
	}

	return nil
}
