package arc

import (
	c "archiver/compressor"
	"archiver/filesystem"
	"archiver/params"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
)

const magicNumber uint16 = 0x5717

// Структура параметров архива
type Arc struct {
	ArchivePath string
	CompType    c.Type
	Compressor  c.Compressor
	DataOffset  int64
}

// Возвращает новый Arc из входных параметров программы
func NewArc(params *params.Params) (*Arc, error) {
	arc := &Arc{
		ArchivePath: params.ArchivePath,
	}

	if filesystem.DirExists(arc.ArchivePath) {
		return nil, fmt.Errorf("'%s' это директория", filepath.Base(arc.ArchivePath))
	}

	if len(params.InputPaths) > 0 {
		arc.CompType = params.CompType
	} else {
		f, err := os.Open(arc.ArchivePath)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		info, err := f.Stat()
		if err != nil {
			return nil, err
		}

		var magic uint16
		if err = binary.Read(f, binary.LittleEndian, &magic); err != nil {
			return nil, err
		}
		if magic != magicNumber {
			return nil, fmt.Errorf("'%s' не архив Arc", info.Name())
		}

		var compType byte
		if err = binary.Read(f, binary.LittleEndian, &compType); err != nil {
			return nil, err
		}
		arc.CompType = c.Type(compType)
	}

	var err error
	arc.Compressor, err = selectCompressor(arc.CompType, params.Level)
	if err != nil {
		return nil, err
	}

	return arc, nil
}

// Выбирает оптимальный способ порождения компрессора
func selectCompressor(compType c.Type, level c.Level) (c.Compressor, error) {
	switch compType {
	case c.GZip, c.ZLib:
		return c.NewCompLevel(compType, level)
	default:
		return c.NewComp(compType)
	}
}
