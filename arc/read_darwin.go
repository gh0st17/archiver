//go:build darwin
// +build darwin

package arc

import (
	"os"
	"syscall"
	"time"
)

// Возвращает временные метки доступа и изменения
func amTimes(info os.FileInfo) (time.Time, time.Time) {
	stat := info.Sys().(*syscall.Stat_t)
	atime := time.Unix(stat.Atimespec.Sec, stat.Atimespec.Nsec)
	mtime := time.Unix(stat.Mtimespec.Sec, stat.Mtimespec.Nsec)

	return atime, mtime
}
