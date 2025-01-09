package compressor

import (
	"compress/flate"
	"compress/gzip"
	"compress/lzw"
	"compress/zlib"
	"fmt"
	"io"
)

const BufferSize int64 = 524288 // 512K

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

// Возвращает нового читателя типа typ
func NewReader(typ Type, r io.Reader) Reader {
	reader, err := newReader(typ, r)
	if err != nil {
		if err == io.EOF {
			panic(fmt.Sprintf("compressor: new reader: %v", err))
		}

		panic(fmt.Sprint("не могу создать новый reader: ", err))
	}

	return Reader{
		reader: reader,
	}
}

// Выбирает читателя согласно typ
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
		panic("new reader: неизвестный тип компрессора")
	}
}

type Writer struct {
	writer io.WriteCloser
}

// Возвращает нового писателя типа typ
func NewWriter(typ Type, w io.Writer, l Level) Writer {
	writer, err := newWriter(typ, w, l)
	if err != nil {
		if err == io.EOF {
			panic(fmt.Sprintf("compressor: new writer: %v", err))
		}
		panic(fmt.Sprint("не могу создать новый writer:", err))
	}

	return Writer{
		writer: writer,
	}
}

// Выбирает писателя согласно typ
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

// Сжимает из p len(p) байт во внутренний writer
func (w Writer) Write(p []byte) (int, error) {
	n, err := w.writer.Write(p)
	if err != nil {
		return 0, fmt.Errorf("compressor write error: %v", err)
	}

	return n, nil
}

func (w Writer) Close() error { return w.writer.Close() }

// Распаковывает из внутреннего reader в p len(p) байт
func (r Reader) Read(p []byte) (int, error) {
	return r.reader.Read(p)
}

func (r Reader) Close() error { return r.reader.Close() }
