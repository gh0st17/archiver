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
	message string
	err     error
	code    int
}

func (e Error) Error() string {
	var eMessage string
	switch {
	case e.err == nil:
		{
		}
	case errors.Is(e.err, gzip.ErrHeader) || errors.Is(e.err, zlib.ErrHeader):
		eMessage = "ошибка заголовка"
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
		return fmt.Sprintf("%s: %s", e.message, eMessage)
	} else {
		return fmt.Sprint(e.message)
	}
}

func ErrRuntime(message string, err error) error {
	return &Error{
		message: message,
		err:     err,
		code:    1,
	}
}

func ErrCompress(message string, err error) error {
	return &Error{
		message: message,
		err:     err,
		code:    2,
	}
}

func ErrDecompress(message string, err error) error {
	return &Error{
		message: message,
		err:     err,
		code:    3,
	}
}

func ErrIntegrity(message string, err error) error {
	return &Error{
		message: message,
		err:     err,
		code:    4,
	}
}

func HandleError(err Error) {
	fmt.Println(err)
	os.Exit(err.code)
}
