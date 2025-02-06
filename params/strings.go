package params

// Строки для справки

const (
	versionDesc = "Печать номера версии и выход"
	versionText = "github.com/gh0st17/archiver 1.0.4\n" +
		"Copyright (C) 2025\n" +
		"Лицензия MIT: THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY\n" +
		"OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO\n" +
		"THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR\n" +
		"PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR\n" +
		"COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER\n" +
		"LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE,\n" +
		"ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE\n" +
		"USE OR OTHER DEALINGS IN THE SOFTWARE.\n\n" +
		"Это свободное ПО: вы можете изменять и распространять его.\n" +
		"Нет НИКАКИХ ГАРАНТИЙ в пределах действующего законодательства.\n\n" +
		"Автор: Alexey Sorokin.\n"

	compExample   = "[Флаги] <путь до архива> <список директории, файлов для сжатия>"
	decompExample = "[-o <путь к директории для распаковки>] <путь до архива>"
	viewExample   = "[-l | -s] <путь до архива>"

	outputDirDesc = "Путь к директории для распаковки"
	dictPathDesc  = "Путь к файлу словаря\n" +
		"Файл словаря представляет собой набор часто встречающихся\n" +
		"фрагментов данных, которые можно использовать для улучшения\n" +
		"сжатия. При декомпрессии необходимо использовать тот же\n" +
		"словарь для восстановления данных.\n" +
		"Поддерживаетя только компрессорами Zlib и Flate."
	levelDesc = "Уровень сжатия от -2 до 9 (Не применяется для LZW)\n" +
		" -2 -- Использовать только сжатие по Хаффману\n" +
		" -1 -- Уровень сжатия по умолчанию (6)\n" +
		"  0 -- Без сжатия\n" +
		"1-9 -- Произвольная степень сжатия"
	compDesc      = "Тип компрессора: GZip, LZW, ZLib, Flate"
	helpDesc      = "Показать эту помощь"
	statDesc      = "Печать информации о сжатии и выход (игнорирует -l)"
	listDesc      = "Печать списка файлов и выход"
	integDesc     = "Проверка целостности данных в архиве"
	xIntegDesc    = "Распаковка с учетом проверки целостности данных в архиве"
	memStatDesc   = "Печать статистики использования ОЗУ после выполнения"
	relaceAllDesc = "Автоматически заменять файлы при распаковке без подтверждения"
	verboseDesc   = "Печатать обработанные файлы"
	logDesc       = "Печатать логи"

	zeroLevel = "Флаг '-L' со значением '0' игнорирует '-c'"
)
