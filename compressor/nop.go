package compressor

import "io"

// Сквозной читатель без сжатия
type nopReader struct{ reader io.ReadCloser }

func (nr nopReader) Read(p []byte) (int, error) {
	return nr.reader.Read(p)
}

func (nr nopReader) Close() error {
	return nr.reader.Close()
}

func (nr nopReader) Reset(io.Reader) error { return nil }

// Сквозной писатель без сжатия
type nopWriteCloser struct{ io.Writer }

func (nopWriteCloser) Close() error { return nil }

func (nopWriteCloser) Reset(io.Writer) {}
