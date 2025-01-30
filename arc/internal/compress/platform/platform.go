package platform

import (
	"os"
	"time"
)

func Timestamp(info os.FileInfo) (atime time.Time, mtime time.Time) {
	return amTimes(info)
}
