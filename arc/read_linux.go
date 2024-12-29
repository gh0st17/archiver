//go:build linux
// +build linux

package arc

import (
	"os"
	"syscall"
	"time"
)

// Возвращает временные метки доступа и изменения
func AMtimes(info os.FileInfo) (time.Time, time.Time) {
	stat := info.Sys().(*syscall.Stat_t)
	atime := time.Unix(stat.Atim.Sec, stat.Atim.Nsec)
	mtime := time.Unix(stat.Mtim.Sec, stat.Mtim.Nsec)

	return atime, mtime
}
