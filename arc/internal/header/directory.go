package header

import "fmt"

// Описание директории
type DirItem struct {
	basePaths
}

// Создает заголовок директории [header.DirItem]
func NewDirItem(pathInArc string) *DirItem {
	return &DirItem{basePaths{pathInArc, pathInArc}}
}

// Реализация fmt.Stringer
func (di DirItem) String() string {
	filename := prefix(di.pathInArc, nameWidth)

	return fmt.Sprintf(
		"%-*s", nameWidth, filename,
	)
}
