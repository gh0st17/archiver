package arc_test

import (
	"archiver/arc"
	"archiver/compressor"
	p "archiver/params"
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

var (
	params p.Params = p.Params{
		ArchivePath: archivePath,
		OutputDir:   outPath,
		Level:       -1,
	}
	rootEnts []os.DirEntry
	err      error
)

func baseTesting(t *testing.T, path string) {
	archive, err := arc.NewArc(&params)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Testing %s compress '%s'", params.CompType, path)
	if err = archive.Compress(params.InputPaths); err != nil {
		t.Fatal(err)
	}

	t.Logf("Testing %s decompress '%s'", params.CompType, path)
	if err = archive.Decompress(params.OutputDir, false); err != nil {
		t.Fatal(err)
	}
}

func archivateAll(t *testing.T) {
	defer func() {
		if t.Failed() {
			// Действия, которые нужно выполнить при провале теста
			t.Log("Test failed, performing cleanup...")
		}
	}()

	for _, rootEnt := range rootEnts {
		path := filepath.Join(prefix, testPath, rootEnt.Name())
		params.InputPaths = append(params.InputPaths, path)
	}

	baseTesting(t, "All Files")

	t.Log("Comparing MD-5 hashsum in/out files")
	for _, rootEnt := range rootEnts {
		checkMD5(t, filepath.Join(prefix, testPath, rootEnt.Name()))
	}
}

func archivateRootEnt(t *testing.T) {
	params.InputPaths = make([]string, 1)

	for _, rootEnt := range rootEnts {
		path := filepath.Join(prefix, testPath, rootEnt.Name())
		params.InputPaths[0] = path

		baseTesting(t, rootEnt.Name())

		t.Log("Comparing MD-5 hashsum in/out files")
		checkMD5(t, filepath.Join(prefix, testPath, rootEnt.Name()))
	}
}

func archivateFile(t *testing.T, rootPaths []string) {
	params.InputPaths = make([]string, 1)

	for _, rootPath := range rootPaths {
		files := fetchDir(rootPath, t)

		for _, fpath := range files {
			params.InputPaths[0] = fpath

			baseTesting(t, fpath)

			t.Log("Comparing MD-5 hashsum in/out files")
			checkMD5(t, fpath)
		}
	}
}

func TestArchiveAll(t *testing.T) {
	t.Cleanup(clearArcOut)

	initRootEnts(t)
	for ct := compressor.Type(0); ct < 4; ct++ {
		t.Log("Testing archivate with", ct, "algorithm")

		params.CompType = ct
		archivateAll(t)
	}
}

func TestArchiveRootEnt(t *testing.T) {
	t.Cleanup(clearArcOut)

	initRootEnts(t)
	for ct := compressor.Type(1); ct < 4; ct++ {
		t.Log("Testing archivate per directory with", ct, "algorithm")

		params.CompType = ct
		archivateRootEnt(t)
	}
}

func TestArchiveFile(t *testing.T) {
	t.Cleanup(clearArcOut)

	initRootEnts(t)
	var rPaths []string
	for _, rE := range rootEnts {
		rPaths = append(rPaths, filepath.Join(prefix, testPath, rE.Name()))
	}

	for ct := compressor.Type(1); ct < 4; ct++ {
		t.Log("Testing archivate per file with", ct, "algorithm")

		params.CompType = ct
		archivateFile(t, rPaths)
	}
}

func initRootEnts(t *testing.T) {
	rootEnts, err = fetchRootDir()
	if err != nil {
		t.Fatal("can't fetch root entries:", err)
	}

	if len(rootEnts) == 0 {
		t.Skip("No entries in testdata for test")
	}
}

func init() {
	clearArcOut()
	log.SetOutput(io.Discard)
}

func clearArcOut() {
	if err := os.RemoveAll(outPath); err != nil {
		fmt.Println("error deleting outPath:", err)
	} else {
		fmt.Println("outPath deleted")
	}

	if err := os.Remove(archivePath); err != nil {
		fmt.Println("error deleting archivePath:", err)
	} else {
		fmt.Println("archivePath deleted")
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
