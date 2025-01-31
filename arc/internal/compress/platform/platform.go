// Пакет platform предоставляет набор кроссплатформенных функции
package platform

import (
	"os"
	"time"
)

// Возвращает временные метки доступа и изменения
func Timestamp(info os.FileInfo) (atime time.Time, mtime time.Time) {
	return amTimes(info)
}
