package arc

import (
	c "archiver/compressor"
	"archiver/filesystem"
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"os"
	"path/filepath"
	"runtime"
)

const magicNumber uint16 = 0x5717

var (
	crct            = crc32.MakeTable(crc32.Koopman)
	ncpu            = runtime.NumCPU()
	compressedBuf   = make([]*bytes.Buffer, ncpu)
	decompressedBuf = make([]*bytes.Buffer, ncpu)
)

// Структура параметров архива
type Arc struct {
	arcPath    string
	ct         c.Type
	dataOffset int64
	replaceAll bool
}

// Возвращает новый Arc из входных параметров программы
func NewArc(arcPath string, inPaths []string, ct c.Type, replaceAll bool) (*Arc, error) {
	arc := &Arc{
		arcPath:    arcPath,
		replaceAll: replaceAll,
	}

	if filesystem.DirExists(arc.arcPath) {
		return nil, fmt.Errorf("'%s' это директория", filepath.Base(arc.arcPath))
	}

	if len(inPaths) > 0 {
		arc.ct = ct
	} else {
		arcFile, err := os.Open(arc.arcPath)
		if err != nil {
			return nil, err
		}
		defer arcFile.Close()

		info, err := arcFile.Stat()
		if err != nil {
			return nil, err
		}

		var magic uint16
		if err = binary.Read(arcFile, binary.LittleEndian, &magic); err != nil {
			return nil, err
		}
		if magic != magicNumber {
			return nil, fmt.Errorf("'%s' не архив Arc", info.Name())
		}

		var compType byte
		if err = binary.Read(arcFile, binary.LittleEndian, &compType); err != nil {
			return nil, err
		}

		if compType <= byte(c.ZLib) {
			arc.ct = c.Type(compType)
		} else {
			return nil, fmt.Errorf("неизвестный тип компрессора")
		}
	}

	return arc, nil
}

func (arc Arc) RemoveTmp() {
	os.Remove(arc.arcPath)
}

func init() {
	for i := 0; i < ncpu; i++ {
		compressedBuf[i] = bytes.NewBuffer(nil)
		decompressedBuf[i] = bytes.NewBuffer(nil)
	}
}
