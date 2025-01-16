//go:build linux
// +build linux

package arc

import (
	"os"
	"syscall"
	"time"
)

// Возвращает временные метки доступа и изменения
func amTimes(info os.FileInfo) (atime time.Time, mtime time.Time) {
	stat := info.Sys().(*syscall.Stat_t)
	atime = time.Unix(int64(stat.Atim.Sec), int64(stat.Atim.Nsec))
	mtime = time.Unix(int64(stat.Mtim.Sec), int64(stat.Mtim.Nsec))

	return atime, mtime
}
