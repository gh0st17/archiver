package compressor

import (
	"compress/flate"
	"fmt"
	"io"
)

type Type byte

const (
	GZip Type = iota
	LempelZivWelch
	ZLib
)

// Реализация fmt.Stringer
func (ct Type) String() string {
	return [...]string{"GZip", "LZW", "ZLib"}[ct]
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
	Read(r io.Reader, w io.Writer) error

	// Сжимает данные из `r` в `w`
	Write(w io.Writer, r io.Reader) error
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
	default:
		return nil, fmt.Errorf("неизвестный тип компрессора")
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
		return nil, fmt.Errorf("неизвестный тип компрессора")
	}
}
