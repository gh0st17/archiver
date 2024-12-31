package main

import (
	"archiver/arc"
	"archiver/params"
	"fmt"
	"os"
)

func main() {
	p := params.ParseParams()
	a, err := arc.NewArc(p)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	switch {
	case len(p.InputPaths) > 0:
		if err := a.Compress(p.InputPaths); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	case p.PrintStat:
		if err := a.ViewStat(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	case p.PrintList:
		if err := a.ViewList(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	default:
		if err := a.Decompress(p.OutputDir); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
}
