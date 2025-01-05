package arc

import (
	c "archiver/compressor"
	"archiver/filesystem"
	"archiver/params"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const magicNumber uint16 = 0x5717

var (
	tmpPath         string
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

func randomString() string {
	const charset = "abcdefghijklmnopqrstuvwxyz"
	var result strings.Builder
	for i := 0; i < 5; i++ {
		randomChar := charset[rand.Intn(len(charset))]
		result.WriteByte(randomChar)
	}
	return result.String()
}

func init() {
	tmpPath = filepath.Join(os.TempDir(), "arctmp"+randomString())
}
