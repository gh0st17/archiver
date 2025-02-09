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
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"syscall"

	"github.com/gh0st17/archiver/arc/internal/generic"
	"github.com/gh0st17/archiver/arc/internal/userinput"
	c "github.com/gh0st17/archiver/compressor"
	"github.com/gh0st17/archiver/errtype"
	"github.com/gh0st17/archiver/filesystem"
	"github.com/gh0st17/archiver/params"
)

const (
	magicNumber uint16 = 0x5717
	headerLen   int64  = 3
)

// Структура параметров архива
type Arc struct {
	path    string // Путь к файлу архива
	verbose bool
	sigChan chan os.Signal
	generic.RestoreParams
}

var allowRemove atomic.Bool

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
	if p.PatternsStr != "" {
		arc.Patterns = strings.Split(p.PatternsStr, ":")
	}
	arc.verbose = p.Verbose

	arc.sigChan = make(chan os.Signal, 1)
	signal.Notify(arc.sigChan, os.Interrupt, syscall.SIGTERM)
	go arc.sigFunc()

	if len(p.InputPaths) > 0 {
		allowRemove.Store(true)
		arc.Ct = p.Ct
		arc.Cl = p.Cl
	} else {
		allowRemove.Store(false)
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

// Печать статистики использования памяти
func (Arc) PrintMemStat() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	fmt.Printf("\nАллоцированная память: %8d KB\n", m.Alloc/1024)
	fmt.Printf("Всего аллокаций:       %8d KB\n", m.TotalAlloc/1024)
	fmt.Printf("Системная память:      %8d KB\n", m.Sys/1024)
	fmt.Printf("Количество сборок мусора: %d\n", m.NumGC)
}

// Обработчик прерывания SIGINT
func (arc Arc) sigFunc() {
	<-arc.sigChan
	fmt.Println("Прерываю...")
	if allowRemove.Load() {
		arc.removeTmp()
	}
	os.Exit(0)
}

// Удаляет архив
func (arc Arc) removeTmp() {
	os.Remove(arc.path)
}

// Закрывает файл архива и удаляет его
func (arc Arc) closeRemove(arcFile io.Closer) {
	arcFile.Close()
	arc.removeTmp()
}

// Создает файл архива и пишет информацию об архиве
func (arc Arc) writeArcHeader() (arcFile *os.File, err error) {
	if _, err = os.Stat(arc.path); err == nil && !*arc.ReplaceAll {
		allowRemove.Store(false)
		if userinput.ReplacePrompt(arc.path, nil, nil) {
			os.Exit(0)
		}
		allowRemove.Store(true)
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
