package header

import (
	"fmt"
)

// Описание директории
type DirItem struct {
	Base
}

func NewDirItem(base *Base) *DirItem { return &DirItem{*base} }

// Реализация fmt.Stringer
func (di DirItem) String() string {
	filename := prefix(di.pathOnDisk, maxInArcWidth)
	mtime := di.mtim.Format(dateFormat)

	return fmt.Sprintf(
		"%-*s  %s", maxInArcWidth, filename, mtime,
	)
}
