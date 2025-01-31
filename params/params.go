package params

import (
	c "archiver/compressor"
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
	InputPaths []string // Пути для архивирования
	OutputDir  string   // Путь к директории для распаковки
	ArcPath    string   // Путь к файлу архива
	DictPath   string   // Путь к словарю
	Ct         c.Type   // Тип компрессора
	Cl         c.Level  // Уровень сжатия
	PrintStat  bool     // Флаг вывода информации об архиве
	PrintList  bool     // Флаг вывода списка содержимого
	IntegTest  bool     // Флаг проверки целостности
	XIntegTest bool     // Флаг распаковки с учетом целостности
	// Флаг вывода статистики использования ОЗУ после выполнения
	MemStat bool
	// Флаг замены всех файлов при распаковке без подтверждения
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
func ParseParams() (p *Params, err error) {
	p = &Params{}
	flag.Usage = PrintHelp
	flag.StringVar(&p.OutputDir, "o", "", outputDirDesc)
	flag.StringVar(&p.DictPath, "dict", "", dictPathDesc)

	var level int
	flag.IntVar(&level, "L", -1, levelDesc)

	var compType string
	flag.StringVar(&compType, "c", "gzip", compDesc)

	flag.BoolVar(&p.PrintStat, "s", false, statDesc)
	flag.BoolVar(&p.PrintList, "l", false, listDesc)
	flag.BoolVar(&p.IntegTest, "integ", false, integDesc)
	flag.BoolVar(&p.XIntegTest, "xinteg", false, xIntegDesc)
	flag.BoolVar(&p.MemStat, "mstat", false, memStatDesc)
	flag.BoolVar(&p.ReplaceAll, "f", false, relaceAllDesc)

	logging := flag.Bool("log", false, logDesc)
	version := flag.Bool("V", false, versionDesc)
	help := flag.Bool("help", false, helpDesc)

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
		return nil, ErrArchivePath
	}

	if err = p.checkPaths(); err != nil {
		return nil, err
	}
	if len(p.InputPaths) > 0 {
		if err = p.checkCompType(compType); err != nil {
			return nil, err
		}
		if err = p.checkCompLevel(level); err != nil {
			return nil, err
		}
	}

	if err = p.checkDict(); err != nil {
		return nil, err
	}

	return p, nil
}

// Явный вывод какие флаги игнорирует флаг
// '-L' со значением '0'
func (p Params) PrintNopLevelIgnore() {
	if p.Cl == c.Level(0) {
		flag.Visit(func(f *flag.Flag) {
			if f.Name == "c" {
				fmt.Println(zeroLevel)
			}
		})
	}
}

// Флаги которые могут быть проигнорированы
// другими флагами
var ignores = [...]string{
	"f", "o", "xinteg", "dict", "integ", "l", "s", "c", "L",
}

// Явный вывод какие флаги игнорирует режим сжатия
func PrintCompressIgnore() {
	printIgnore("Сжатие файлов", append(ignores[:3], ignores[4:7]...))
}

// Явный вывод какие флаги игнорирует флаг '-s'
func PrintStatIgnore() {
	printIgnore("Наличие флага 's'", append(ignores[:6], ignores[7:]...))
}

// Явный вывод какие флаги игнорирует флаг '-l'
func PrintListIgnore() {
	printIgnore("Наличие флага 'l'", append(ignores[:5], ignores[7:]...))
}

// Явный вывод какие флаги игнорирует флаг '--integ'
func PrintIntegIgnore() {
	printIgnore("Наличие флага 'integ'", append(ignores[:4], ignores[7:]...))
}

// Явный вывод какие флаги игнорирует распаковка архива
func PrintDecompressIgnore() {
	printIgnore("Распаковка архива", ignores[4:])
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
func (p *Params) checkCompLevel(level int) error {
	p.Cl = c.Level(level)
	if p.Cl < -2 || p.Cl > 9 {
		return ErrCompLevel
	} else if p.Cl == 0 {
		p.Ct = c.Nop
	}

	return nil
}

// Проверяет параметр типа компрессора
func (p *Params) checkCompType(compType string) error {
	compType = strings.ToLower(compType)

	switch compType {
	case "gzip":
		p.Ct = c.GZip
	case "lzw":
		p.Ct = c.LempelZivWelch
	case "zlib":
		p.Ct = c.ZLib
	case "flate":
		p.Ct = c.Flate
	default:
		return ErrUnknownComp
	}

	return nil
}

// Проверяет пути к файлам и архиву
func (p *Params) checkPaths() error {
	if len(flag.Args()) == 0 {
		return ErrArcInPath
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
		return ErrSelfContains
	}

	return nil
}

func (p Params) checkDict() error {
	if len(p.InputPaths) == 0 || p.DictPath == "" {
		return nil
	}

	switch p.Ct {
	case c.GZip, c.LempelZivWelch, c.Nop:
		return ErrUnsupportedDict(p.Ct)
	}

	return nil
}
