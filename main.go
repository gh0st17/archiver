package main

import (
	"archiver/arc"
	"archiver/errtype"
	"archiver/params"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	p := params.ParseParams()
	a, err := arc.NewArc(p)
	if err != nil {
		errtype.ErrorHandler(err)
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
		p.PrintNopLevelIgnore()
		params.PrintPathsIgnore()
		err = a.Compress(p.InputPaths)
	case p.PrintStat:
		params.PrintStatIgnore()
		err = a.ViewStat()
	case p.PrintList:
		params.PrintListIgnore()
		err = a.ViewList()
	case p.IntegTest:
		params.PrintIntegIgnore()
		err = a.IntegrityTest()
	default:
		params.PrintDecompressIgnore()
		err = a.Decompress(p.OutputDir, p.XIntegTest)
	}

	if err != nil {
		errtype.ErrorHandler(err)
	}

	if p.MemStat {
		a.PrintMemStat()
	}
}
