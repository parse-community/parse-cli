package main

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"

	"github.com/facebookgo/ensure"
)

var testDwarfPath = filepath.Join("Resources", "Test.xcarchive",
	"dSYMs", "Test.app.dSYM", "Contents", "Resources", "DWARF", "Test")

func TestIOSSymbolUploaderValidate(t *testing.T) {
	t.Parallel()
	var i iosSymbolUploader
	err := i.validate()
	if runtime.GOOS != "darwin" {
		ensure.Err(t, err,
			regexp.MustCompile(`Upload of iOS symbol files is only available on OS X.`))
	} else {
		ensure.Nil(t, err)
	}
}

func TestDwarfPath(t *testing.T) {
	t.Parallel()
	path, err := getDwarfPath("Resources/Test.xcarchive")
	ensure.Nil(t, err)
	ensure.DeepEqual(t, path, testDwarfPath)
}

func TestAcceptsPath(t *testing.T) {
	t.Parallel()
	i := iosSymbolUploader{Path: testDwarfPath}
	ensure.True(t, i.acceptsPath())
	i.Path = ""
	ensure.False(t, i.acceptsPath())
}

func TestBuildVersionFromXArchive(t *testing.T) {
	t.Parallel()
	var i iosSymbolUploader
	def, err := i.getBuildVersionFromXcarchive()
	ensure.DeepEqual(t, def, "1")
	ensure.Nil(t, err)

	i.Path = "./Resources/Test.xcarchive"
	v, err := i.getBuildVersionFromXcarchive()
	ensure.Nil(t, err)
	ensure.DeepEqual(t, v, "1.0")
}

func TestPrepareSymbolsFolder(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	defer h.Stop()

	h.makeEmptyRoot()
	createRandomFiles(t, h)

	var i iosSymbolUploader
	ensure.Nil(t, i.prepareSymbolsFolder(h.env.Root, h.env))
	files, err := readDirNames(h.env.Root)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, len(files), 0)
}

func TestSymbolConversionToolPath(t *testing.T) {
	t.Parallel()
	var i iosSymbolUploader
	h := newHarness(t)
	defer h.Stop()

	h.makeEmptyRoot()
	ensure.Nil(t, os.MkdirAll(filepath.Join(h.env.Root, ".parse"), 0755))
	path, err := i.symbolConversionTool(h.env.Root,
		func(path string, url string) error {
			return nil
		},
		h.env)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, path, filepath.Join(h.env.Root, ".parse", "parse_symbol_converter"))
	ensure.DeepEqual(t, h.Out.String(),
		`Additional resources installed.
`)
}
