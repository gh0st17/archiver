package arc

import (
	"archiver/arc/internal/decompress"
	c "archiver/compressor"
	"archiver/errtype"
	"archiver/filesystem"
	"archiver/params"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

const (
	magicNumber  uint16 = 0x5717
	arcHeaderLen int64  = 3
)

// Структура параметров архива
type Arc struct {
	arcPath   string // Путь к файлу архива
	outputDir string
	integ     bool
	ct        c.Type  // Тип компрессора
	cl        c.Level // Уровень сжатия
	// Флаг замены файлов без подтверждения
	replaceAll bool
}

// Возвращает новый [Arc] из входных параметров программы
func NewArc(p params.Params) (arc *Arc, err error) {
	arc = &Arc{
		arcPath:    p.ArcPath,
		replaceAll: p.ReplaceAll,
	}

	if filesystem.DirExists(arc.arcPath) {
		return nil, errIsDir(filepath.Base(arc.arcPath))
	}

	if len(p.InputPaths) > 0 {
		arc.ct = p.Ct
		arc.cl = p.Cl
	} else {
		arcFile, err := os.Open(arc.arcPath)
		if err != nil {
			return nil, errtype.Join(ErrOpenArc, err)
		}
		defer arcFile.Close()

		var magic uint16
		if err = filesystem.BinaryRead(arcFile, &magic); err != nil {
			return nil, errtype.Join(ErrReadMagic, err)
		}
		if magic != magicNumber {
			return nil, errNotArc(arcFile.Name())
		}

		var compType byte
		if err = filesystem.BinaryRead(arcFile, &compType); err != nil {
			return nil, err
		}

		if compType <= byte(c.ZLib) {
			arc.ct = c.Type(compType)
		} else {
			return nil, ErrUnknownComp
		}

		arc.integ = p.XIntegTest
		arc.outputDir = p.OutputDir
	}

	return arc, nil
}

func (arc Arc) ToRestoreParams() decompress.RestoreParams {
	return decompress.RestoreParams{
		OutputDir:  arc.outputDir,
		Integ:      arc.integ,
		Ct:         arc.ct,
		Cl:         arc.cl,
		ReplaceAll: arc.replaceAll,
	}
}

// Удаляет архив
func (arc Arc) RemoveTmp() {
	os.Remove(arc.arcPath)
}

// Закрывает файл архива и удаляет его
func (arc Arc) closeRemove(arcFile io.Closer) {
	arcFile.Close()
	arc.RemoveTmp()
}

// Печать статистики использования памяти
func (Arc) PrintMemStat() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	fmt.Printf("\nАллоцированная память: %8d KB\n", m.Alloc/1024)
	fmt.Printf("Всего аллокаций:       %8d KB\n", m.TotalAlloc/1024)
	fmt.Printf("Системная память:      %8d KB\n", m.Sys/1024)
	fmt.Printf("Количество сборок мусора: %d\n", m.NumGC)
}
