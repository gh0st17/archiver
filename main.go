package main

import (
	"archiver/arc"
	"archiver/params"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

func main() {
	p := params.ParseParams()
	a, err := arc.NewArc(
		p.ArchivePath, p.InputPaths, p.CompType,
	)
	if err != nil {
		fmt.Println("arc:", err)
		os.Exit(1)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("Прерываю...")
		if len(p.InputPaths) > 0 {
			a.RemoveTmp()
		}
		os.Exit(0)
	}()

	switch {
	case len(p.InputPaths) > 0:
		err = a.Compress(p.InputPaths)
	case p.PrintStat:
		err = a.ViewStat()
	case p.PrintList:
		err = a.ViewList()
	case p.IntegTest:
		err = a.IntegrityTest()
	default:
		err = a.Decompress(p.OutputDir, p.XIntegTest)
	}

	if err != nil {
		fmt.Println("arc:", err)
		os.Exit(1)
	}

	if p.MemStat {
		printMemStat()
	}
}

func printMemStat() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	fmt.Printf("\nАллоцированная память: %8d KB\n", m.Alloc/1024)
	fmt.Printf("Всего аллокаций:       %8d KB\n", m.TotalAlloc/1024)
	fmt.Printf("Системная память:      %8d KB\n", m.Sys/1024)
	fmt.Printf("Количество сборок мусора: %d\n", m.NumGC)
}
