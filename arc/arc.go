// Пакет arc предоставляет функции для работы с архивом.
// Позволяет создавать и распаковывать архивы,
// печатать содержимое архива, выполнять проверку целостности
//
// Основные функции:
//   - NewArc: Создает новую структуру [Arc]
//   - Compress: Создает файл архива
//   - Decompress: Выполняет распаковку архива
//   - IntegrityTest: Проверяет целостность данных в архиве
//   - ViewStat: Печатает подробную информацию об архиве
//   - ViewList: Печатает список файлов в архиве
package arc

import (
	"archiver/arc/internal/generic"
	"archiver/arc/internal/userinput"
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
	magicNumber uint16 = 0x5717
	headerLen   int64  = 3
)

// Структура параметров архива
type Arc struct {
	path    string // Путь к файлу архива
	verbose bool
	generic.RestoreParams
}

// Возвращает новый [Arc] из входных параметров программы
func NewArc(p params.Params) (arc *Arc, err error) {
	arc = &Arc{
		path: p.ArcPath,
	}

	if filesystem.DirExists(arc.path) {
		return nil, ErrIsDir(filepath.Base(arc.path))
	}

	arc.ReplaceAll = &p.ReplaceAll
	arc.DictPath = p.DictPath
	arc.verbose = p.Verbose

	if len(p.InputPaths) > 0 {
		arc.Ct = p.Ct
		arc.Cl = p.Cl
	} else {
		arcFile, err := os.Open(arc.path)
		if err != nil {
			return nil, errtype.Join(ErrOpenArc, err)
		}
		defer arcFile.Close()

		var magic uint16
		if err = filesystem.BinaryRead(arcFile, &magic); err != nil {
			return nil, errtype.Join(ErrReadMagic, err)
		}
		if magic != magicNumber {
			return nil, ErrNotArc(arcFile.Name())
		}

		var compType byte
		if err = filesystem.BinaryRead(arcFile, &compType); err != nil {
			return nil, err
		}

		if compType <= byte(c.Flate) {
			arc.Ct = c.Type(compType)
		} else {
			return nil, ErrUnknownComp
		}

		arc.Integ = p.XIntegTest
		arc.OutputDir = p.OutputDir
	}

	return arc, nil
}

// Удаляет архив
func (arc Arc) RemoveTmp() {
	os.Remove(arc.path)
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

// Закрывает файл архива и удаляет его
func (arc Arc) closeRemove(arcFile io.Closer) {
	arcFile.Close()
	arc.RemoveTmp()
}

// Создает файл архива и пишет информацию об архиве
func (arc Arc) writeArcHeader() (arcFile *os.File, err error) {
	if _, err = os.Stat(arc.path); err == nil && !*arc.ReplaceAll {
		if userinput.ReplacePrompt(arc.path, nil, nil) {
			os.Exit(0)
		}
	}

	// Создаем файл
	arcFile, err = os.Create(arc.path)
	if err != nil {
		return nil, errtype.Join(ErrCreateArc, err)
	}

	// Пишем магическое число
	if err = filesystem.BinaryWrite(arcFile, magicNumber); err != nil {
		return nil, errtype.Join(ErrWriteMagic, err)
	}

	// Пишем тип компрессора
	if err = filesystem.BinaryWrite(arcFile, arc.Ct); err != nil {
		return nil, errtype.Join(ErrWriteCompType, err)
	}

	return arcFile, nil
}
