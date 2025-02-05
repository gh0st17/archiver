package main

import (
	"github.com/gh0st17/archiver/arc"
	"github.com/gh0st17/archiver/errtype"
	"github.com/gh0st17/archiver/params"
)

func main() {
	p, err := params.ParseParams()
	if err != nil {
		errtype.ErrorHandler(errtype.ErrArgument(err))
	}

	a, err := arc.NewArc(*p)
	if err != nil {
		errtype.ErrorHandler(err)
	}

	switch {
	case len(p.InputPaths) > 0:
		p.PrintNopLevelIgnore()
		params.PrintCompressIgnore()
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
		err = a.Decompress()
	}

	if err != nil {
		errtype.ErrorHandler(err)
	}

	if p.MemStat {
		a.PrintMemStat()
	}
}
