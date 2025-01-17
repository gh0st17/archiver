package compressor

import (
	"errors"
)

var (
	ErrDecompCreate = errors.New("не могу создать новый декомпрессор")
	ErrCompCreate   = errors.New("не могу создать новый компрессор")
	ErrUnknownComp  = errors.New("неизвестный тип компрессора")
)
