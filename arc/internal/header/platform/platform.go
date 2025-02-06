package platform

// Возвращает размеры терминала
func GetTerminalSize() (int, int, error) {
	return getTerminalSize()
}
