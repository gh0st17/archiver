package arc_test

import (
	"archiver/arc"
	"archiver/compressor"
	"archiver/filesystem"
	"archiver/params"
	"crypto/sha1"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"testing"
)

const (
	prefix      = "../"
	testPath    = "testdata"
	arcName     = "archive.arc"
	archivePath = prefix + testPath + "/" + arcName
	outPath     = prefix + testPath + "/out"
)

func archivateAll(t *testing.T, ct compressor.Type, rootEnts []os.DirEntry) {
	if len(rootEnts) == 0 {
		t.Skip("No entries in testdata for test")
	}

	params := params.Params{
		ArchivePath: archivePath,
		OutputDir:   outPath,
		CompType:    ct,
		Level:       -1,
	}

	for _, rootEnt := range rootEnts {
		path := filepath.Join(prefix, testPath, rootEnt.Name())
		params.InputPaths = append(params.InputPaths, path)
	}

	log.SetOutput(io.Discard)
	archive, err := arc.NewArc(&params)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Testing", ct, "compress")

	if err = archive.Compress(params.InputPaths); err != nil {
		t.Fatal(err)
	}

	t.Log("Testing", ct, "decompress")

	if err = archive.Decompress(params.OutputDir); err != nil {
		t.Fatal(err)
	}

	t.Log("Comparing SHA-1 hashsum before and after", ct, "compress")

	if checkSHA1(t, rootEnts) {
		t.Log("All files are matched")
	}
}

func clearArcOut(t *testing.T) {
	if err := os.RemoveAll(outPath); err != nil {
		t.Error("error deleting outPath:", err)
	} else {
		t.Log("outPath deleted")
	}

	if err := os.Remove(archivePath); err != nil {
		t.Error("error deleting archivePath:", err)
	} else {
		t.Log("archivePath deleted")
	}
}

func fetchRootDir() ([]os.DirEntry, error) {
	rootEntries, err := os.ReadDir(prefix + testPath)
	if err != nil {
		return nil, err
	}

	return rootEntries, nil
}

func fetchDir(path string, t *testing.T) []string {
	var files []string

	err := filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			t.Fatal("error during fetch files path:", err)
		}

		if d.IsDir() {
			return nil
		}

		files = append(files, path)

		return nil
	})
	if err != nil {
		t.Fatal("error after fetch files path:", err)
	}

	return files
}

func checkSHA1(t *testing.T, rootEnts []os.DirEntry) bool {
	var (
		err             error
		files           []string
		outFilepath     string
		inSHA1, outSHA1 []byte
	)

	for _, rootEnt := range rootEnts {
		if !rootEnt.IsDir() {
			continue
		}

		files = fetchDir(filepath.Join(prefix, testPath, rootEnt.Name()), t)

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
			}
		}
	}

	return true
}

const chunkSize = 10 * 1024 * 1024

func hashFileSHA1(filePath string) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть файл: %w", err)
	}
	defer file.Close()

	hash := sha1.New()
	buffer := make([]byte, chunkSize)

	for {
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("ошибка при чтении файла: %w", err)
		}
		if n == 0 { // Конец файла
			break
		}

		_, writeErr := hash.Write(buffer[:n])
		if writeErr != nil {
			return nil, fmt.Errorf("ошибка при вычислении хеша: %w", writeErr)
		}
	}

	return hash.Sum(nil), nil
}

func compareSHA1(buf1, buf2 []byte) bool {
	for i := range buf1 {
		if buf1[i] != buf2[i] {
			return false
		}
	}

	return true
}

func archivateRootEnt(t *testing.T, ct compressor.Type) {
	rootEnts, err := fetchRootDir()
	if err != nil {
		t.Fatal("can't fetch root entries:", err)
	}

	params := params.Params{
		ArchivePath: archivePath,
		InputPaths:  make([]string, 1),
		OutputDir:   outPath,
		CompType:    ct,
		Level:       -1,
	}

	log.SetOutput(io.Discard)

	var archive *arc.Arc
	for _, rootEnt := range rootEnts {
		path := filepath.Join(prefix, testPath, rootEnt.Name())
		params.InputPaths[0] = path

		archive, err = arc.NewArc(&params)
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("Testing %s compress '%s'", ct, rootEnt.Name())

		if err = archive.Compress(params.InputPaths); err != nil {
			t.Fatal(err)
		}

		t.Logf("Testing %s decompress '%s'", ct, rootEnt.Name())

		if err = archive.Decompress(params.OutputDir); err != nil {
			t.Fatal(err)
		}

		t.Log("Comparing SHA-1 hashsum before and after", ct, "compress")

		if checkSHA1(t, []os.DirEntry{rootEnt}) {
			t.Log("All files are matched")
		}
	}
}

func TestArchiveAll(t *testing.T) {
	clearArcOut(t)
	for ct := compressor.Type(0); ct < 4; ct++ {
		t.Log("Testing archivate with", ct, "algorithm")
		rootEnts, err := fetchRootDir()
		if err != nil {
			t.Fatal("can't fetch root entries:", err)
		}

		archivateAll(t, ct, rootEnts)
		clearArcOut(t)
	}
}

func TestArchiveRootEnt(t *testing.T) {
	clearArcOut(t)
	for ct := compressor.Type(0); ct < 4; ct++ {
		t.Log("Testing archivate per directory with", ct, "algorithm")
		archivateRootEnt(t, ct)
		clearArcOut(t)
	}
}
