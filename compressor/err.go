package compressor

import "fmt"

var (
	ErrNewReader       = fmt.Errorf("не могу создать новый декомпрессор")
	ErrNewWriter       = fmt.Errorf("не могу создать новый компрессор")
	ErrUnknownComp     = fmt.Errorf("неизвестный тип компрессора")
	ErrUnsupportedDict = func(ct Type) error {
		return fmt.Errorf("выбранный тип компрессора (%s) не поддерживает словарь", ct)
	}
)
