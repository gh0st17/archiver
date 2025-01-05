package compressor

import (
	"bytes"
	"compress/flate"
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

type Compressor interface {
	NewReader(r io.Reader) (io.ReadCloser, error)  // Читатель для распаковки
	NewWriter(w io.Writer) (io.WriteCloser, error) // Писатель для сжатия
}

// Возвращает компрессор с уровнем сжатия по умолчанию
func NewComp(compType Type) (Compressor, error) {
	switch compType {
	case GZip:
		return NewGz(), nil
	case LempelZivWelch:
		return NewLZW(), nil
	case ZLib:
		return NewZlib(), nil
	case Nop:
		return NewNop(), nil
	default:
		return nil, fmt.Errorf("newcomp: неизвестный тип компрессора")
	}
}

// Возвращает компрессор с указанным уровнем сжатия
//
// Алгоритмы которые не поддерживают установку
// уровня сжатия не порождаются этой функцией
// даже если они есть в `Type`
func NewCompLevel(compType Type, level Level) (Compressor, error) {
	switch compType {
	case GZip:
		return NewGzLevel(level), nil
	case ZLib:
		return NewZlibLevel(level), nil
	default:
		return nil, fmt.Errorf("newcomplevel: неизвестный тип компрессора")
	}
}

// Сжимает данные в uBuf, возвращает сжатые данные
func Compress(uBuf []byte, c Compressor) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	cw, err := c.NewWriter(buf)
	if err != nil {
		return nil, err
	}

	// Записываем прочитанные данные в компрессор
	if _, err = cw.Write(uBuf); err != nil {
		cw.Close()
		return nil, err
	}

	if err := cw.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), err
}

// Распаковывает данные в cBuf, возвращает несжатые данные
func Decompress(cBuf []byte, c Compressor) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	cr, err := c.NewReader(bytes.NewReader(cBuf))
	if err != nil {
		return nil, err
	}
	defer cr.Close()

	// Распаковываем данные в buf
	if _, err = io.Copy(buf, cr); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
