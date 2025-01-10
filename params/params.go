package params

import (
	"archiver/compressor"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type Params struct {
	InputPaths  []string
	OutputDir   string
	ArchivePath string
	CompType    compressor.Type
	Level       compressor.Level
	PrintStat,
	PrintList,
	IntegTest,
	XIntegTest,
	MemStat,
	ReplaceAll bool
}

// Печатает справку
func PrintHelp() {
	program := filepath.Base(os.Args[0])

	fmt.Println("Сжатие:    ", program, compExample)
	fmt.Println("Распаковка:", program, decompExample)
	fmt.Println("Просмотр:  ", program, viewExample)
	fmt.Printf("\nФлаги:\n")

	flag.PrintDefaults()
}

// Возвращает структуру Params с прочитанными
// входными аргументами программы
func ParseParams() (p Params) {
	flag.Usage = PrintHelp
	flag.StringVar(&p.OutputDir, "o", "", outputDirDesc)

	var level int
	flag.IntVar(&level, "L", -1, levelDesc)

	var compType string
	flag.StringVar(&compType, "c", "gzip", compDesc)

	var help bool
	flag.BoolVar(&help, "help", false, helpDesc)
	flag.BoolVar(&p.PrintStat, "s", false, statDesc)
	flag.BoolVar(&p.PrintList, "l", false, listDesc)
	flag.BoolVar(&p.IntegTest, "integ", false, integDesc)
	flag.BoolVar(&p.XIntegTest, "xinteg", false, xIntegDesc)
	flag.BoolVar(&p.MemStat, "mstat", false, memStatDesc)
	flag.BoolVar(&p.ReplaceAll, "f", false, relaceAllDesc)
	logging := flag.Bool("log", false, logDesc)
	version := flag.Bool("V", false, versionDesc)
	flag.Parse()

	if !*logging {
		log.SetOutput(io.Discard)
	}

	if *version {
		fmt.Print(versionText)
		os.Exit(0)
	}
	if help {
		PrintHelp()
		os.Exit(0)
	}

	p.checkCompType(compType)
	p.checkCompLevel(level)

	if (p.PrintList || p.PrintStat) && len(flag.Args()) == 0 {
		printError(archivePathError)
	}

	p.checkPaths()

	return p
}

// Проверяет параметр уровня сжатия
func (p *Params) checkCompLevel(level int) {
	p.Level = compressor.Level(level)
	if p.Level < -2 && p.Level > 9 {
		printError(compLevelError)
	} else if p.Level == 0 {
		p.CompType = compressor.Nop
	}
}

// Проверяет параметр типа компрессора
func (p *Params) checkCompType(compType string) {
	compType = strings.ToLower(compType)

	switch compType {
	case "gzip":
		p.CompType = compressor.GZip
	case "lzw":
		p.CompType = compressor.LempelZivWelch
	case "zlib":
		p.CompType = compressor.ZLib
	default:
		printError(compTypeError)
	}
}

// Проверяет пути к файлам и архиву
func (p *Params) checkPaths() {
	if len(flag.Args()) == 0 {
		printError(archivePathInputPathError)
	}

	pathsLen := len(flag.Args()[1:])
	argsLen := len(flag.Args())

	if pathsLen > 0 {
		p.InputPaths = append(p.InputPaths, flag.Args()[1:]...)
		p.ArchivePath = flag.Arg(0)
	} else if argsLen == 1 {
		p.ArchivePath = flag.Arg(0)
	}

	if isContain(p.InputPaths, p.ArchivePath) {
		printError(containsError)
	}
}

// Выводит сообщение об ошибке
func printError(message string) {
	fmt.Printf("%s\n\n", message)
	PrintHelp()
	os.Exit(1)
}

// Проверяет, содержится ли строка в массиве строк
func isContain(slice []string, str string) bool {
	for _, v := range slice {
		if v == str {
			return true
		}
	}
	return false
}
