//go:build !windows
// +build !windows

package platform

import (
	"os"

	"golang.org/x/term"
)

func getTerminalSize() (int, int, error) {
	return term.GetSize(int(os.Stdin.Fd()))
}
