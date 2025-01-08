package arc

import (
	c "archiver/compressor"
	"archiver/filesystem"
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
	uncompressedBuf = make([][]byte, ncpu)
	compressedBuf   = make([][]byte, ncpu)
)

// Структура параметров архива
type Arc struct {
	ArchivePath string
	CompType    c.Type
	DataOffset  int64
	maxCompLen  int64
}

// Возвращает новый Arc из входных параметров программы
func NewArc(arcPath string, inPaths []string, ct c.Type) (*Arc, error) {
	arc := &Arc{
		ArchivePath: arcPath,
	}

	if filesystem.DirExists(arc.ArchivePath) {
		return nil, fmt.Errorf("'%s' это директория", filepath.Base(arc.ArchivePath))
	}

	if len(inPaths) > 0 {
		arc.CompType = ct
	} else {
		arcFile, err := os.Open(arc.ArchivePath)
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
			arc.CompType = c.Type(compType)
		} else {
			return nil, fmt.Errorf("неизвестный тип компрессора")
		}
	}

	return arc, nil
}

func (arc Arc) RemoveTmp() {
	os.Remove(arc.ArchivePath)
}
