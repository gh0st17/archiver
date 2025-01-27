package header

import (
	"fmt"
)

// Описание директории
type DirItem struct {
	basePaths
}

func NewDirItem(pathInArc string) *DirItem {
	return &DirItem{basePaths{pathInArc, pathInArc}}
}

// Реализация fmt.Stringer
func (di DirItem) String() string {
	filename := prefix(di.pathInArc, maxInArcWidth)

	return fmt.Sprintf(
		"%-*s", maxInArcWidth, filename,
	)
}
