package compressor

import (
	"fmt"
)

var (
	ErrDecompCreate = fmt.Errorf("не могу создать новый декомпрессор")
	ErrCompCreate   = fmt.Errorf("не могу создать новый компрессор")
	ErrUnknownComp  = fmt.Errorf("неизвестный тип компрессора")
)
