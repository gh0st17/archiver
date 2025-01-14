package arc

import (
	c "archiver/compressor"
	"archiver/errtype"
	"archiver/filesystem"
	"archiver/params"
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
	compressor      = make([]*c.Writer, ncpu)
	decompressor    = make([]*c.Reader, ncpu)
	writeBuf        *bytes.Buffer
)

// Структура параметров архива
type Arc struct {
	arcPath    string
	ct         c.Type
	cl         c.Level
	replaceAll bool
}

// Возвращает новый Arc из входных параметров программы
func NewArc(p params.Params) (arc *Arc, err error) {
	arc = &Arc{
		arcPath:    p.ArcPath,
		replaceAll: p.ReplaceAll,
	}

	if filesystem.DirExists(arc.arcPath) {
		return nil, errtype.ErrRuntime(
			fmt.Sprintf("'%s' это директория", filepath.Base(arc.arcPath)), nil,
		)
	}

	if len(p.InputPaths) > 0 {
		arc.ct = p.Ct
		arc.cl = p.Cl
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
			return nil, errtype.ErrRuntime(
				fmt.Sprintf("'%s' не архив Arc", info.Name()), nil,
			)
		}

		var compType byte
		if err = binary.Read(arcFile, binary.LittleEndian, &compType); err != nil {
			return nil, err
		}

		if compType <= byte(c.ZLib) {
			arc.ct = c.Type(compType)
		} else {
			return nil, errtype.ErrRuntime("неизвестный тип компрессора", nil)
		}
	}

	return arc, nil
}

func (arc Arc) RemoveTmp() {
	os.Remove(arc.arcPath)
}

func init() {
	for i := 0; i < ncpu; i++ {
		compressedBuf[i] = bytes.NewBuffer(make([]byte, 0, c.BufferSize))
		decompressedBuf[i] = bytes.NewBuffer(make([]byte, 0, c.BufferSize))
	}

	writeBuf = bytes.NewBuffer(make([]byte, 0, int(c.BufferSize)*ncpu))
}

func (Arc) PrintMemStat() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	fmt.Printf("\nАллоцированная память: %8d KB\n", m.Alloc/1024)
	fmt.Printf("Всего аллокаций:       %8d KB\n", m.TotalAlloc/1024)
	fmt.Printf("Системная память:      %8d KB\n", m.Sys/1024)
	fmt.Printf("Количество сборок мусора: %d\n", m.NumGC)
}
