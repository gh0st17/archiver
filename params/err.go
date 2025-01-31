package params

import (
	"archiver/compressor"
	"fmt"
)

var (
	ErrCompLevel            = fmt.Errorf("уровень сжатия должен быть в пределах от -2 до 9")
	ErrUnknownComp          = compressor.ErrUnknownComp
	ErrArchivePathInputPath = fmt.Errorf("имя архива и список файлов не указаны")
	ErrArchivePath          = fmt.Errorf("имя архива не указано")
	ErrSelfContains         = fmt.Errorf("путь к файлу не должен указывать на указаннный архив")
	ErrUnsupportedDict      = compressor.ErrUnsupportedDict
)
