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

type Type byte // Тип компрессора

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

type Level int // Уровень сжатия

const (
	HuffmanOnly        Level = flate.HuffmanOnly
	DefaultCompression Level = flate.DefaultCompression
	NoCompression      Level = flate.NoCompression
	BestSpeed          Level = flate.BestSpeed
	BestCompression    Level = flate.BestCompression
)

type ReadCloserReset interface {
	io.ReadCloser
	Reset(io.Reader) error
}

// Адаптер для lzw.Reader
type lzwReader struct {
	*lzw.Reader
}

func (lr *lzwReader) Reset(r io.Reader) error {
	lr.Reader.Reset(r, lzw.MSB, 8)
	return nil
}

// Адаптер для zlib.Reader
type zlibReader struct {
	reader io.ReadCloser
}

func (zr *zlibReader) Read(p []byte) (int, error) {
	return zr.reader.Read(p)
}

func (zr *zlibReader) Close() error {
	return zr.reader.Close()
}

func (zr *zlibReader) Reset(r io.Reader) error {
	return zr.reader.(zlib.Resetter).Reset(r, nil)
}

type Reader struct {
	reader ReadCloserReset
}

// Возвращает нового читателя типа typ
func NewReader(typ Type, r io.Reader) (*Reader, error) {
	reader, err := newReader(typ, r)
	if err != nil {
		if err == io.EOF {
			return nil, err
		}

		return nil, errtype.Join(ErrDecompCreate, err)
	}

	return &Reader{
		reader: reader,
	}, nil
}

// Выбирает читателя согласно typ
func newReader(typ Type, r io.Reader) (ReadCloserReset, error) {
	switch typ {
	case GZip:
		return gzip.NewReader(r)
	case LempelZivWelch:
		return &lzwReader{lzw.NewReader(r, lzw.MSB, 8).(*lzw.Reader)}, nil
	case ZLib:
		z, err := zlib.NewReader(r)
		if err != nil {
			return nil, err
		}
		return &zlibReader{z}, nil
	case Nop:
		return &nopReader{io.NopCloser(r)}, nil
	default:
		return nil, ErrUnknownComp
	}
}

// Копирует буфер [Reader.reader] в w
func (rd *Reader) WriteTo(w io.Writer) (int64, error) {
	n, err := io.Copy(w, rd.reader)
	if err != nil && err != io.EOF {
		return n, err
	}

	return n, nil
}

func (rd *Reader) Close() error { return rd.reader.Close() }

func (rd *Reader) Reset(r io.Reader) error { return rd.reader.Reset(r) }

type WriteCloseResetter interface {
	io.WriteCloser
	Reset(io.Writer)
}

// Адаптер для lzw.Writer
type lzwWriter struct {
	*lzw.Writer
}

func (lw *lzwWriter) Reset(w io.Writer) {
	lw.Writer.Reset(w, lzw.MSB, 8)
}

type Writer struct {
	writer WriteCloseResetter
}

// Возвращает нового писателя типа typ
func NewWriter(typ Type, w io.Writer, l Level) (*Writer, error) {
	writer, err := newWriter(typ, w, l)
	if err != nil {
		if err == io.EOF {
			return nil, err
		}
		return nil, errtype.Join(ErrCompCreate, err)
	}

	return &Writer{
		writer: writer,
	}, nil
}

// Выбирает писателя согласно typ
func newWriter(typ Type, w io.Writer, l Level) (WriteCloseResetter, error) {
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
		return nil, ErrUnknownComp
	}
}

// Сжимает из p len(p) байт во внутренний writer
func (wr *Writer) Write(p []byte) (n int, err error) {
	n, err = wr.writer.Write(p)
	if err != nil {
		return 0, err
	}

	return n, nil
}

func (wr *Writer) Close() error { return wr.writer.Close() }

func (wr *Writer) Reset(w io.Writer) { wr.writer.Reset(w) }
