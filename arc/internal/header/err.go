package header

import (
	"fmt"
	"path/filepath"
)

var (
	ErrPathLength = func(length int64) error {
		return fmt.Errorf("некорректная длина (%d) пути элемента", length)
	}

	ErrLongPath = func(path string) error {
		return fmt.Errorf(
			"длина пути к '%s' первышает максимально допустимую (1023)",
			filepath.Base(path),
		)
	}
)
