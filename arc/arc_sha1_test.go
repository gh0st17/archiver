package arc_test

import (
	"archiver/filesystem"
	"bytes"
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func checkSHA1(t *testing.T, path string) bool {
	var (
		err             error
		files           []string
		outFilepath     string
		inSHA1, outSHA1 sha1hash
	)

	if filesystem.DirExists(path) {
		files = fetchDir(path, t)
	} else {
		files = append(files, path)
	}

	for _, inFilepath := range files {
		outFilepath = filepath.Join(outPath, filesystem.CleanPath(inFilepath))

		inSHA1, err = hashFileSHA1(inFilepath)
		if err != nil {
			t.Fatal("inSHA1:", err)
		}

		outSHA1, err = hashFileSHA1(outFilepath)
		if err != nil {
			t.Fatal("outSHA1:", err)
		}

		if compareSHA1(inSHA1, outSHA1) == false {
			t.Errorf("SHA1 sum mismatch %v for '%s'\n", inSHA1, inFilepath)
			return false
		} else {
			t.Logf("%s matched '%s'", inSHA1, inFilepath)
		}
	}

	return true
}

const chunkSize = 10 * 1024 * 1024

func hashFileSHA1(filePath string) (sha1hash, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть файл: %v", err)
	}
	defer file.Close()

	hash := sha1.New()
	buffer := make([]byte, chunkSize)

	for {
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("ошибка при чтении файла: %v", err)
		}
		if n == 0 {
			break
		}

		_, writeErr := hash.Write(buffer[:n])
		if writeErr != nil {
			return nil, fmt.Errorf("ошибка при вычислении хеша: %v", writeErr)
		}
	}

	return sha1hash(hash.Sum(nil)), nil
}

func compareSHA1(buf1, buf2 sha1hash) bool {
	for i := range buf1 {
		if buf1[i] != buf2[i] {
			return false
		}
	}

	return true
}

type sha1hash []byte

func (s sha1hash) String() string {
	var buf bytes.Buffer
	buf.WriteString("0x")

	for _, b := range s {
		buf.WriteString(fmt.Sprintf("%02x", b))
	}

	return buf.String()
}
