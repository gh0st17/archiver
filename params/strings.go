package params

// Строки для справки

const (
	versionDesc = "Печать номера версии и выход"
	versionText = `archiver 1.0.1
Copyright (C) 2025
Лицензия MIT: THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY
OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO 
THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR
PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE,
ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE
USE OR OTHER DEALINGS IN THE SOFTWARE.

Это свободное ПО: вы можете изменять и распространять его.
Нет НИКАКИХ ГАРАНТИЙ в пределах действующего законодательства.

Автор: Alexey Sorokin.
`

	compExample   = "[Флаги] <путь до архива> <список директории, файлов для сжатия>"
	decompExample = "[-o <путь к директории для распаковки>] <путь до архива>"
	viewExample   = "[-l | -s] <путь до архива>"

	outputDirDesc = "Путь к директории для распаковки"
	levelDesc     = `Уровень сжатия от -2 до 9 (Не применяется для LZW)
 -2 -- HuffmanOnly
 -1 -- DefaultCompression
  0 -- Без сжатия
1-9 -- Произвольная степень сжатия`
	compDesc      = "Тип компрессора: GZip, LZW, ZLib"
	helpDesc      = "Показать эту помощь"
	statDesc      = "Печать информации о сжатии и выход (игнорирует -l)"
	listDesc      = "Печать списка файлов и выход"
	integDesc     = "Проверка целостности данных в архиве"
	xIntegDesc    = "Распаковка с учетом проверки целостности данных в архиве"
	memStatDesc   = "Печать статистики использования ОЗУ после выполнения"
	relaceAllDesc = "Автоматически заменять существующие файлы"
	logDesc       = "Печатать логи"

	compLevelError            = "Уровень сжатия должен быть в пределах от -2 до 9"
	compTypeError             = "Неизвестный тип компрессора"
	archivePathInputPathError = "Имя архива и список файлов не указаны"
	archivePathError          = "Имя архива не указано"
	containsError             = "Путь к файлу не должен указывать на указаннный архив"
)
