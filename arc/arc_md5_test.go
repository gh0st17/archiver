package arc_test

import (
	"archiver/filesystem"
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func checkMD5(t *testing.T, path string) bool {
	var (
		err           error
		files         []string
		outFilepath   string
		inMD5, outMD5 MD5hash
	)

	if filesystem.DirExists(path) {
		files = fetchDir(path, t)
	} else {
		files = append(files, path)
	}

	for _, inFilepath := range files {
		outFilepath = filepath.Join(outPath, filesystem.Clean(inFilepath))

		inMD5, err = hashFileMD5(inFilepath)
		if err != nil {
			t.Fatal("inMD5:", err)
		}

		outMD5, err = hashFileMD5(outFilepath)
		if err != nil {
			t.Fatal("outMD5:", err)
		}

		if compareMD5(inMD5, outMD5) == false {
			t.Errorf("%s mismatched '%s'", inMD5, inFilepath)
			t.Fail()
			return false
		} else {
			t.Logf("%s matched '%s'", inMD5, inFilepath)
		}
	}

	return true
}

const chunkSize = 10 * 1024 * 1024

func hashFileMD5(filePath string) (MD5hash, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть файл: %v", err)
	}
	defer file.Close()

	hash := md5.New()
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

	return MD5hash(hash.Sum(nil)), nil
}

func compareMD5(buf1, buf2 MD5hash) bool {
	for i := range buf1 {
		if buf1[i] != buf2[i] {
			return false
		}
	}

	return true
}

type MD5hash []byte

func (s MD5hash) String() string {
	var buf bytes.Buffer
	buf.WriteString("0x")

	for _, b := range s {
		buf.WriteString(fmt.Sprintf("%02x", b))
	}

	return buf.String()
}
