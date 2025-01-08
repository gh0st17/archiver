package arc_test

import (
	"archiver/arc"
	"archiver/compressor"
	p "archiver/params"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"testing"
)

const (
	prefix   = "../"
	testPath = "testdata"
	arcName  = "archive.arc"
)

var (
	archivePath = filepath.Join(os.TempDir(), arcName)
	outPath     = filepath.Join(os.TempDir(), "/out")
	params      = p.Params{
		ArchivePath: archivePath,
		OutputDir:   outPath,
		Level:       -1,
	}
	rootEnts []os.DirEntry
	stdout   = os.Stdout
	stderr   = os.Stderr
)

func TestNopAll(t *testing.T) {
	runTestAll(t, compressor.Nop)
}

func TestGzipAll(t *testing.T) {
	runTestAll(t, compressor.GZip)
}

func TestLzwAll(t *testing.T) {
	runTestAll(t, compressor.LempelZivWelch)
}

func TestZlibAll(t *testing.T) {
	runTestAll(t, compressor.ZLib)
}

func TestNopByEntry(t *testing.T) {
	runTestByEntry(t, compressor.Nop)
}

func TestGzipByEntry(t *testing.T) {
	runTestByEntry(t, compressor.GZip)
}

func TestLzwByEntry(t *testing.T) {
	runTestByEntry(t, compressor.LempelZivWelch)
}

func TestZlibByEntry(t *testing.T) {
	runTestByEntry(t, compressor.ZLib)
}

func TestNopByFile(t *testing.T) {
	runTestByFile(t, compressor.Nop)
}

func TestGzipByFile(t *testing.T) {
	runTestByFile(t, compressor.GZip)
}

func TestLzwByFile(t *testing.T) {
	runTestByFile(t, compressor.LempelZivWelch)
}

func TestZlibByFile(t *testing.T) {
	runTestByFile(t, compressor.ZLib)
}

func runTestAll(t *testing.T, ct compressor.Type) {
	t.Cleanup(clearArcOut)
	initRootEnts(t)
	t.Log("Testing archivate all files in",
		testPath, "with", ct, "algorithm")
	params.CompType = ct
	runAll(t)
	clearArcOut()
}

func runTestByEntry(t *testing.T, ct compressor.Type) {
	t.Cleanup(clearArcOut)
	initRootEnts(t)
	t.Log("Testing archivate by directory with",
		ct, "algorithm")
	params.CompType = ct
	runByEntry(t)
	clearArcOut()
}

func runTestByFile(t *testing.T, ct compressor.Type) {
	t.Cleanup(clearArcOut)

	initRootEnts(t)
	var rootPaths []string
	for _, e := range rootEnts {
		rootPaths = append(rootPaths,
			filepath.Join(prefix, testPath, e.Name()))
	}

	t.Log("Testing archivate by file with", ct, "algorithm")
	params.CompType = ct
	runByFile(t, rootPaths)
	clearArcOut()
}

func baseTesting(t *testing.T, path string) {
	archive, err := arc.NewArc(&params)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Testing %s compress '%s'", params.CompType, path)
	disableStdout()
	if err = archive.Compress(params.InputPaths); err != nil {
		enableStdout()
		t.Fatal(err)
	}
	enableStdout()

	t.Logf("Testing %s decompress '%s'", params.CompType, path)
	disableStdout()
	if err = archive.Decompress(params.OutputDir, false); err != nil {
		enableStdout()
		t.Fatal(err)
	}
	enableStdout()
}

func runAll(t *testing.T) {
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

func runByEntry(t *testing.T) {
	params.InputPaths = make([]string, 1)

	for _, rootEnt := range rootEnts {
		path := filepath.Join(prefix, testPath, rootEnt.Name())
		params.InputPaths[0] = path

		baseTesting(t, rootEnt.Name())

		t.Log("Comparing MD-5 hashsum in/out files")
		checkMD5(t, filepath.Join(prefix, testPath, rootEnt.Name()))
	}
}

func runByFile(t *testing.T, rootPaths []string) {
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

func initRootEnts(t *testing.T) {
	var err error
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
	os.RemoveAll(outPath)
	os.Remove(archivePath)
}

func disableStdout() {
	os.Stdout = nil
	os.Stderr = nil
}

func enableStdout() {
	os.Stdout = stdout
	os.Stderr = stderr
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
