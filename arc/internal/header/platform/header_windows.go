//go:build windows
// +build windows

package platform

import (
	"os"

	"golang.org/x/sys/windows"
)

func getTerminalSize() (int, int, error) {
	handle := windows.Handle(os.Stdout.Fd())
	var info windows.ConsoleScreenBufferInfo
	err := windows.GetConsoleScreenBufferInfo(handle, &info)
	if err != nil {
		return 0, 0, err
	}
	return int(info.Window.Right - info.Window.Left), int(info.Window.Bottom - info.Window.Top), nil
}
