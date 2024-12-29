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
		if err := arc.Compress(a, p.InputPaths); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	case p.PrintStat:
		if err := arc.ViewStat(a); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	case p.PrintList:
		if err := arc.ViewList(a); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	default:
		if err := arc.Decompress(a, p.OutputDir); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
}
