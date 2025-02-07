package arc_test

import (
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"

	"github.com/gh0st17/archiver/filesystem"
)

func checkMD5(t *testing.T, path string) bool {
	var (
		files []string
		sem   = make(chan struct{}, ncpu)
		wg    = sync.WaitGroup{}
	)

	if filesystem.DirExists(path) {
		files = fetchDir(path, t)
	} else {
		files = append(files, path)
	}

	for i := range files {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int) {
			defer func() {
				wg.Done()
				<-sem
			}()

			checkFileMD5(t, files[i])
		}(i)
		if t.Failed() {
			break
		}
	}

	wg.Wait()
	return !t.Failed()
}

func checkFileMD5(t *testing.T, inFilepath string) {
	outFilepath := filepath.Join(outPath, filesystem.Clean(inFilepath))

	inMD5, err := hashFileMD5(inFilepath)
	if err != nil {
		t.Fatal("inMD5:", err)
	}

	outMD5, err := hashFileMD5(outFilepath)
	if err != nil {
		t.Fatal("outMD5:", err)
	}

	if slices.Compare(inMD5, outMD5) == 0 {
		t.Logf("%s matched '%s'", outMD5, inFilepath)
	} else {
		t.Errorf(
			"Mismatched '%s':\nexpected %s got %s",
			inFilepath, inMD5, outMD5,
		)
		t.FailNow()
	}
}

func hashFileMD5(filePath string) (MD5hash, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть файл: %v", err)
	}
	defer file.Close()

	hash := md5.New()

	for {
		n, err := io.CopyN(hash, file, chunkSize)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return nil, fmt.Errorf("ошибка при чтении файла: %v", err)
		}
		if n == 0 {
			break
		}
	}

	return MD5hash(hash.Sum(nil)), nil
}

type MD5hash []byte

func (s MD5hash) String() string {
	var buf bytes.Buffer

	for _, b := range s {
		buf.WriteString(fmt.Sprintf("%02x", b))
	}

	return buf.String()
}

func testSymlink(path string) bool {
	if info, err := os.Lstat(path); err != nil && errors.Is(err, os.ErrNotExist) {
		return false
	} else if err != nil {
		panic(err)
	} else {
		return info.Mode()&os.ModeSymlink != 0
	}
}
