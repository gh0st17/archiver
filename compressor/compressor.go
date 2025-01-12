package compressor

import (
	"archiver/errtype"
	"compress/flate"
	"compress/gzip"
	"compress/lzw"
	"compress/zlib"
	"io"
)

const BufferSize int64 = 1048576 // 1М

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
func NewReader(typ Type, r io.Reader) (*Reader, error) {
	reader, err := newReader(typ, r)
	if err != nil {
		if err == io.EOF {
			return nil, errtype.ErrRuntime("читатель достиг EOF", err)
		}

		return nil, errtype.ErrRuntime("не могу создать новый декомпрессор", err)
	}

	return &Reader{
		reader: reader,
	}, nil
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
		return nil, errtype.ErrRuntime("неизвестный тип компрессора", nil)
	}
}

type WriteCloserReset interface {
	io.WriteCloser
	Reset(io.Writer)
}

// Адаптер для lzw.Writer
type lzwWriter struct {
	*lzw.Writer
}

func (lw lzwWriter) Reset(w io.Writer) {
	lw.Writer.Reset(w, lzw.MSB, 8)
}

type Writer struct {
	writer WriteCloserReset
}

func (w *Writer) Reset(iow io.Writer) { w.writer.Reset(iow) }

// Возвращает нового писателя типа typ
func NewWriter(typ Type, w io.Writer, l Level) (*Writer, error) {
	writer, err := newWriter(typ, w, l)
	if err != nil {
		if err == io.EOF {
			return nil, errtype.ErrRuntime("писатель достиг EOF", err)
		}
		return nil, errtype.ErrRuntime("не могу создать новый компрессор", err)
	}

	return &Writer{
		writer: writer,
	}, nil
}

// Выбирает писателя согласно typ
func newWriter(typ Type, w io.Writer, l Level) (WriteCloserReset, error) {
	switch typ {
	case GZip:
		return gzip.NewWriterLevel(w, int(l))
	case LempelZivWelch:
		return &lzwWriter{lzw.NewWriter(w, lzw.MSB, 8).(*lzw.Writer)}, nil
	case ZLib:
		return zlib.NewWriterLevel(w, int(l))
	case Nop:
		return nopWriteCloser{Writer: w}, nil
	default:
		return nil, errtype.ErrRuntime("неизвестный тип компрессора", nil)
	}
}

// Сжимает из p len(p) байт во внутренний writer
func (w Writer) Write(p []byte) (n int, err error) {
	n, err = w.writer.Write(p)
	if err != nil {
		return 0, err
	}

	return n, nil
}

func (w Writer) Close() error { return w.writer.Close() }

// Распаковывает из внутреннего reader в p len(p) байт
func (r Reader) Read(p []byte) (int, error) {
	return r.reader.Read(p)
}

func (r Reader) Close() error { return r.reader.Close() }
