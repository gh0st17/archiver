package compressor

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/lzw"
	"compress/zlib"
	"fmt"
	"io"
)

const BufferSize int64 = 1 * 1024 * 1024

type Type byte

const (
	Nop Type = iota
	GZip
	LempelZivWelch
	ZLib
)

// Реализация fmt.Stringer
func (ct Type) String() string {
	return [...]string{"Nop", "GZip", "LZW", "ZLib"}[ct]
}

type Level int

const (
	HuffmanOnly        Level = flate.HuffmanOnly
	DefaultCompression Level = flate.DefaultCompression
	NoCompression      Level = flate.NoCompression
	BestSpeed          Level = flate.BestSpeed
	BestCompression    Level = flate.BestCompression
)

type Reader struct {
	reader io.ReadCloser
}

func NewReader(typ Type, r io.Reader) Reader {
	reader, err := newReader(typ, r)
	if err != nil {
		panic(fmt.Sprint("не могу создать новый Reader: ", err))
	}

	return Reader{
		reader: reader,
	}
}

type Writer struct {
	writer io.WriteCloser
}

func NewWriter(typ Type, l Level, w io.Writer) Writer {
	writer, err := newWriter(typ, w, l)
	if err != nil {
		panic(fmt.Sprint("не могу создать новый Reader:", err))
	}

	return Writer{
		writer: writer,
	}
}

func newReader(typ Type, r io.Reader) (io.ReadCloser, error) {
	switch typ {
	case GZip:
		return gzip.NewReader(r)
	case LempelZivWelch:
		return lzw.NewReader(r, lzw.MSB, 8), nil
	case ZLib:
		return zlib.NewReader(r)
	case Nop:
		return io.NopCloser(r), nil
	default:
		panic("newReader: неизвестный тип компрессора")
	}
}

func newWriter(typ Type, w io.Writer, l Level) (io.WriteCloser, error) {
	switch typ {
	case GZip:
		return gzip.NewWriterLevel(w, int(l))
	case LempelZivWelch:
		return lzw.NewWriter(w, lzw.MSB, 8), nil
	case ZLib:
		return zlib.NewWriterLevel(w, int(l))
	case Nop:
		return nopWriteCloser{Writer: w}, nil
	default:
		panic("newWriter: неизвестный тип компрессора")
	}
}

func (w Writer) Write(p []byte) (int, error) {
	n, err := w.writer.Write(p)
	if err != nil {
		w.writer.Close()
		return 0, fmt.Errorf("compressor write error: %v", err)
	}

	if err := w.writer.Close(); err != nil {
		return 0, fmt.Errorf("compressor close error: %v", err)
	}

	return n, nil
}

func (r Reader) Read(p *[]byte) (int, error) {
	defer r.reader.Close()

	buf := bytes.NewBuffer(nil)
	n, err := io.Copy(buf, r.reader)
	if err != nil {
		return 0, err
	}

	*p = buf.Bytes()

	return int(n), nil
}
