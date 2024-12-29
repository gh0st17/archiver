//go:build windows
// +build windows

package arc

import (
	"os"
	"syscall"
	"time"
)

// Возвращает временные метки доступа и изменения
func AMtimes(info os.FileInfo) (time.Time, time.Time) {
	stat := info.Sys().(*syscall.Win32FileAttributeData)
	atime := time.Unix(0, stat.LastAccessTime.Nanoseconds())
	mtime := time.Unix(0, stat.LastWriteTime.Nanoseconds())

	return atime, mtime
}
