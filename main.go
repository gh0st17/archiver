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
		errtype.HandleError(*err.(*errtype.Error))
	}

	if p.MemStat {
		a.PrintMemStat()
	}
}
