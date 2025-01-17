package arc

import (
	"archiver/arc/header"
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
	// Полином CRC32
	crct = crc32.MakeTable(crc32.Koopman)
	// Количество доступных процессоров
	ncpu = runtime.NumCPU()
	// Буффер для сжатых данных
	compressedBuf = make([]*bytes.Buffer, ncpu)
	// Буфер для несжатых данных
	decompressedBuf = make([]*bytes.Buffer, ncpu)
	compressor      = make([]*c.Writer, ncpu)
	decompressor    = make([]*c.Reader, ncpu)
)

// Структура параметров архива
type Arc struct {
	arcPath string  // Путь к файлу архива
	ct      c.Type  // Тип компрессора
	cl      c.Level // Уровень сжатия
	// Флаг замены файлов без подтверждения
	replaceAll bool
}

// Возвращает новый Arc из входных параметров программы
func NewArc(p params.Params) (arc *Arc, err error) {
	arc = &Arc{
		arcPath:    p.ArcPath,
		replaceAll: p.ReplaceAll,
	}

	if filesystem.DirExists(arc.arcPath) {
		return nil, errtype.ErrRuntime(errIsDir(filepath.Base(arc.arcPath)), nil)
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
			return nil, errtype.ErrRuntime(errNotArc(info.Name()), nil)
		}

		var compType byte
		if err = binary.Read(arcFile, binary.LittleEndian, &compType); err != nil {
			return nil, err
		}

		if compType <= byte(c.ZLib) {
			arc.ct = c.Type(compType)
		} else {
			return nil, errtype.ErrRuntime(ErrUnknownComp, nil)
		}
	}

	return arc, nil
}

// Удаляет архив
func (arc Arc) RemoveTmp() {
	os.Remove(arc.arcPath)
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

// Разделяет заголовки на директории и файлы
func (Arc) splitHeaders(headers []header.Header) ([]*header.DirItem, []*header.FileItem) {
	var (
		dirs  []*header.DirItem
		files []*header.FileItem
	)

	for _, h := range headers {
		if len(h.Path()) > 1023 {
			fmt.Printf(
				"Длина пути к '%s' первышает максимально допустимую (1023)\n",
				filepath.Base(h.Path()),
			)
			continue
		}
		if d, ok := h.(*header.DirItem); ok {
			dirs = append(dirs, d)
		} else {
			files = append(files, h.(*header.FileItem))
		}
	}

	return dirs, files
}

// Проверяет корректность размера буфера.
// Возвращает true если размер некорректный.
func (Arc) checkBufferSize(bufferSize int64) bool {
	return bufferSize < 0 || bufferSize>>1 > c.BufferSize
}

func init() {
	for i := 0; i < ncpu; i++ {
		compressedBuf[i] = bytes.NewBuffer(make([]byte, 0, c.BufferSize))
		decompressedBuf[i] = bytes.NewBuffer(make([]byte, 0, c.BufferSize))
	}
}
