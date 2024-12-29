package params

import (
	"archiver/compressor"
	"flag"
	"fmt"
	"os"
)

type Params struct {
	InputPaths  []string
	OutputDir   string
	ArchivePath string
	Level       compressor.Level
	PrintStat   bool
	PrintList   bool
	CompType    compressor.Type
}

// Печатает справку
func PrintHelp() {
	fmt.Println("Сжатие:    ", os.Args[0], compExample)
	fmt.Println("Распаковка:", os.Args[0], decompExample)
	fmt.Printf("\nФлаги:\n")

	flag.PrintDefaults()
}

// Возвращает структуру Params с прочитанными
// входными аргументами программы
func ParseParams() *Params {
	var p Params

	flag.Usage = PrintHelp
	flag.StringVar(&p.OutputDir, "o", "", outputDirDesc)

	var level int
	flag.IntVar(&level, "L", -1, levelDesc)

	var IsLZW, IsZLib bool
	flag.BoolVar(&IsLZW, "lzw", false, lzwDesc)
	flag.BoolVar(&IsZLib, "zlib", false, zlibDesc)

	var help bool
	flag.BoolVar(&help, "help", false, helpDesc)
	flag.BoolVar(&p.PrintStat, "s", false, statDesc)
	flag.BoolVar(&p.PrintList, "l", false, listDesc)
	flag.Parse()

	if help {
		PrintHelp()
		os.Exit(0)
	}

	checkCompLevel(&p, level)
	checkCompType(IsLZW, IsZLib, &p)

	if (p.PrintList || p.PrintStat) && len(flag.Args()) == 0 {
		printError(archivePathError)
	}

	checkPaths(&p)

	return &p
}

// Проверяет параметр уровня сжатия
func checkCompLevel(p *Params, level int) {
	p.Level = compressor.Level(level)
	if p.Level < -2 && p.Level > 9 {
		printError(compTypeError)
	}
}

// Проверяет параметр типа компрессора
func checkCompType(IsLZW bool, IsZLib bool, p *Params) {
	if IsLZW && IsZLib {
		printError(compTypeError)
	} else if IsLZW {
		p.CompType = compressor.LempelZivWelch
	} else if IsZLib {
		p.CompType = compressor.ZLib
	}
}

// Проверяет пути к файлам и архиву
func checkPaths(p *Params) {
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
