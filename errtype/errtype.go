package errtype

import (
	"compress/gzip"
	"compress/zlib"
	"errors"
	"fmt"
	"io"
	"os"
)

type Error struct {
	errMessage error // Ошибка (этой) программы
	err        error // Встроенная ошибка (библотеки и т.д.)
	code       int   // Код завершения после вывода ошибки
}

// Перевод и форматирование встроенных ошибок
func (e Error) Error() string {
	var eMessage string
	switch {
	case e.err == nil:
		{
		}
	case errors.Is(e.err, gzip.ErrHeader) || errors.Is(e.err, zlib.ErrHeader):
		eMessage = "ошибка заголовка сжатых данных"
	case errors.Is(e.err, gzip.ErrChecksum) || errors.Is(e.err, zlib.ErrChecksum):
		eMessage = "неверная контрольная сумма"
	case errors.Is(e.err, os.ErrPermission):
		eMessage = fmt.Sprint("нет доступа:", e.err)
	case errors.Is(e.err, os.ErrExist):
		eMessage = "файл уже существует"
	case errors.Is(e.err, os.ErrNotExist):
		eMessage = "файл не существует"
	case errors.Is(e.err, io.EOF):
		eMessage = "достигнут конец файла"
	case errors.Is(e.err, io.ErrUnexpectedEOF):
		eMessage = "неожиданный конец файла"
	default:
		eMessage = e.err.Error()
	}

	if e.err != nil {
		return fmt.Sprintf("%s: %s", e.errMessage, eMessage)
	} else {
		return fmt.Sprint(e.errMessage)
	}
}

// Возвращает встроенную ошибку
func (e Error) Err() error { return e.err }

// Возвращает общую ошибку времени выполнения
func ErrRuntime(errMessage error, err error) error {
	return &Error{
		errMessage: errMessage,
		err:        err,
		code:       1,
	}
}

// Возвращает ошибки при сжатии
func ErrCompress(errMessage error, err error) error {
	return &Error{
		errMessage: errMessage,
		err:        err,
		code:       2,
	}
}

// Возвращает ошибки при распаковке
func ErrDecompress(errMessage error, err error) error {
	return &Error{
		errMessage: errMessage,
		err:        err,
		code:       3,
	}
}

// Возвращает ошибки при проверке целостности
func ErrIntegrity(errMessage error, err error) error {
	return &Error{
		errMessage: errMessage,
		err:        err,
		code:       4,
	}
}

// Обработчик ошибок
func HandleError(err Error) {
	fmt.Println(err)
	os.Exit(err.code)
}
