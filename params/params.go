package params

import (
	"archiver/compressor"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type Params struct {
	InputPaths []string
	OutputDir  string
	ArcPath    string
	Ct         compressor.Type
	Cl         compressor.Level
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

	help := flag.Bool("help", false, helpDesc)
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
	if *help {
		PrintHelp()
		os.Exit(0)
	}

	if (p.PrintList || p.PrintStat) && len(flag.Args()) == 0 {
		printError(archivePathError)
	}

	p.checkPaths()
	if len(p.InputPaths) > 0 {
		p.checkCompType(compType)
		p.checkCompLevel(level)
	}

	return p
}

// Явный вывод какие флаги игнорирует флаг
// '-L' со значением '0'
func (p Params) PrintNopLevelIgnore() {
	if p.Cl == compressor.Level(0) {
		flag.Visit(func(f *flag.Flag) {
			if f.Name == "c" {
				fmt.Println(zeroLevel)
			}
		})
	}
}

var ignores = []string{
	"f", "o", "xinteg", "integ", "l", "s", "c", "L",
}

// Явный вывод какие флаги игнорирует
// наличие путей после имени архива
func PrintPathsIgnore() {
	printIgnore("Наличие путей после имени архива", ignores[:6])
}

// Явный вывод какие флаги игнорирует флаг '-s'
func PrintStatIgnore() {
	printIgnore("Наличие флага 's'", append(ignores[:5], ignores[6:]...))
}

// Явный вывод какие флаги игнорирует флаг '-l'
func PrintListIgnore() {
	printIgnore("Наличие флага 'l'", append(ignores[:4], ignores[5:]...))
}

// Явный вывод какие флаги игнорирует флаг '--integ'
func PrintIntegIgnore() {
	printIgnore("Наличие флага 'integ'", append(ignores[:3], ignores[4:]...))
}

// Явный вывод какие флаги игнорирует флаг
// отсутствие путей после имени архива
func PrintDecompressIgnore() {
	printIgnore("Отсутствие путей после имени архива", ignores[3:])
}

// Общий шаблон вывода информации о том какие
// флаги будут проигнорированы
func printIgnore(fstr string, ignores []string) {
	isVisit := false
	flag.Visit(func(f *flag.Flag) {
		if slices.Contains(ignores, f.Name) && !isVisit {
			fmt.Printf("%s игнорирует флаги: '%s'", fstr, ignores[0])
			for _, ignore := range ignores[1:] {
				fmt.Printf(", '%s'", ignore)
			}
			fmt.Println()
			isVisit = true
		}
	})
}

// Проверяет параметр уровня сжатия
func (p *Params) checkCompLevel(level int) {
	p.Cl = compressor.Level(level)
	if p.Cl < -2 || p.Cl > 9 {
		printError(compLevelError)
	} else if p.Cl == 0 {
		p.Ct = compressor.Nop
	}
}

// Проверяет параметр типа компрессора
func (p *Params) checkCompType(compType string) {
	compType = strings.ToLower(compType)

	switch compType {
	case "gzip":
		p.Ct = compressor.GZip
	case "lzw":
		p.Ct = compressor.LempelZivWelch
	case "zlib":
		p.Ct = compressor.ZLib
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
		p.ArcPath = flag.Arg(0)
	} else if argsLen == 1 {
		p.ArcPath = flag.Arg(0)
	}

	if slices.Contains(p.InputPaths, p.ArcPath) {
		printError(containsError)
	}
}

// Выводит сообщение об ошибке
func printError(message string) {
	fmt.Printf("%s\n\n", message)
	PrintHelp()
	os.Exit(1)
}
