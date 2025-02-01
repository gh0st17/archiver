// Пакет userinput предоставляет функции для внутренней
// обработки пользовательского ввода
package userinput

import (
	"bufio"
	"fmt"
	"os"
	"unicode"

	"golang.org/x/term"
)

// Обрабатывает диалог замены файла
//
// allFunc и negFunc принимают функции для действия в случае
// выбора всех файлов и негативном ответе
//
// Если ответ был негативный функция возвращает true
func ReplacePrompt(outPath string, allFunc func(), negFunc func()) bool {
	if IsNonInteractive() {
		return false
	}

	actions := func() string {
		if allFunc == nil {
			return "(Д)а/(Н)ет"
		} else {
			return "(Д)а/(Н)ет/(В)се"
		}
	}()

	var (
		result       bool
		needContinue bool
		input        rune
		stdin        = bufio.NewReader(os.Stdin)
	)

	for {
		fmt.Printf("Файл '%s' существует, заменить? [%s]: ", outPath, actions)
		input, _, _ = stdin.ReadRune()
		input = unicode.ToLower(input)

		if allFunc == nil {
			result, needContinue = yesNoSwitch(input, negFunc)
		} else {
			result, needContinue = allSwitch(input, allFunc, negFunc)
		}

		if needContinue {
			if input != '\n' {
				stdin.ReadString('\n')
			}
			continue
		}

		return result
	}
}

// Обрабатывает диалог замены с возможностью применить
// выбор пользователя для всех файлов
func allSwitch(input rune, allFunc func(), negFunc func()) (bool, bool) {
	switch input {
	case 'a', 'в':
		if allFunc != nil {
			allFunc()
			return false, false
		}
	default:
		return yesNoSwitch(input, negFunc)
	}

	return false, false
}

// Обрабатывает диалог замены формата "Да/Нет"
func yesNoSwitch(input rune, negFunc func()) (bool, bool) {
	switch input {
	case 'y', 'д':
	case 'n', 'н':
		if negFunc != nil {
			negFunc()
		}
		return true, false
	default:
		return false, true
	}

	return false, false
}

// Проверяет является ли стандратный ввод терминалом
func IsNonInteractive() bool {
	fi, _ := os.Stdin.Stat()
	return (fi.Mode()&os.ModeCharDevice) == 0 || !term.IsTerminal(int(os.Stdin.Fd()))
}
