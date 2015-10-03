package parsecmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/ParsePlatform/parse-cli/parsecli"
	"github.com/facebookgo/ensure"
)

func TestMain(m *testing.M) {
	if filepath.Base(os.Args[0]) == "aapt" {
		fmt.Fprintln(os.Stdout, `Android Asset Packaging Tool
	versionCode='2778'
	`)
		os.Exit(0)
	}

	aaptBin := filepath.Join("Resources", "build_tools", "aapt")

	bin, err := os.Create(aaptBin)
	if err != nil {
		panic(err)
	}
	defer bin.Close()

	cur, err := os.Open(os.Args[0])
	if err != nil {
		panic(err)
	}
	defer cur.Close()

	if _, err := io.Copy(bin, cur); err != nil {
		panic(err)
	}
	if err := os.Chmod(aaptBin, 0755); err != nil {
		panic(err)
	}

	if err := bin.Close(); err != nil {
		panic(err)
	}

	retCode := m.Run()

	if err := os.Remove(aaptBin); err != nil {
		panic(err)
	}
	os.Exit(retCode)
}

func TestParseVersionFromBytes(t *testing.T) {
	t.Parallel()

	_, err := parseVersionFromBytes([]byte(`versionCode='a'`))
	ensure.NotNil(t, err)

	tests := []struct {
		input    string
		expected int
	}{
		{`versionCode='1'`, 1},
		{`versionCode='12'`, 12},
		{`versionCode='1234'`, 1234},
		{`versionCode='12345678'`, 12345678},
	}

	for _, test := range tests {
		actual, err := parseVersionFromBytes([]byte(test.input))
		ensure.Nil(t, err)
		ensure.DeepEqual(t, actual, test.expected)
	}
}

func TestAndroidSymbolUploaderValidate(t *testing.T) {
	t.Parallel()
	var a androidSymbolUploader

	ensure.Err(t, a.validate(), regexp.MustCompile(
		"You need to supply one of either the apk or manifest file."))

	a.Apk = "a"
	a.Manifest = "m"
	ensure.Err(t, a.validate(), regexp.MustCompile(
		"You need to supply one of either the apk or manifest file."))

	a.Apk = ""
	a.Manifest = "m"
	ensure.Nil(t, a.validate())

	a.Manifest = ""
	a.Apk = "a"
	ensure.Nil(t, a.validate())
}

func TestAndroidAcceptsPath(t *testing.T) {
	t.Parallel()
	a := androidSymbolUploader{
		Path: filepath.Join("Resources", "mapping.txt"),
	}
	ensure.True(t, a.acceptsPath())
}

func TestFindAAPT(t *testing.T) {
	t.Parallel()
	h := parsecli.NewHarness(t)
	defer h.Stop()

	aapts := findAAPT("Resources")
	ensure.DeepEqual(t, aapts[:], []string{filepath.Join("Resources", "build_tools", "aapt")})
}

func TestBuildVersionFromManifest(t *testing.T) {
	t.Parallel()
	a := androidSymbolUploader{
		Path:     filepath.Join("Resources", "mapping.txt"),
		Manifest: filepath.Join("Resources", "AndroidManifest.xml"),
	}

	v, err := a.getBuildVersionFromManifest()
	ensure.Nil(t, err)
	ensure.DeepEqual(t, v, 2778)
}

func TestBuildVersionFromApk(t *testing.T) {
	t.Parallel()
	a := androidSymbolUploader{
		Path: filepath.Join("Resources", "mapping.txt"),
		Apk:  "test",
		AAPT: filepath.Join("Resources", "build_tools", "aapt"),
	}

	v, err := a.getBuildVersionFromAPK()
	ensure.Nil(t, err)
	ensure.DeepEqual(t, v, 2778)
}

func TestGetAAPT(t *testing.T) {
	t.Parallel()
	a := androidSymbolUploader{
		Path: filepath.Join("Resources", "mapping.txt"),
	}

	path, err := a.getAAPT("Resources")
	ensure.Nil(t, err)
	ensure.DeepEqual(t, path, filepath.Join("Resources", "build_tools", "aapt"))
}

func TestGetBuildVersion(t *testing.T) {
	t.Parallel()
	a := androidSymbolUploader{
		Path:     filepath.Join("Resources", "mapping.txt"),
		Manifest: filepath.Join("Resources", "AndroidManifest.xml"),
		Apk:      "test",
		AAPT:     filepath.Join("Resources", "build_tools", "aapt"),
	}

	h := parsecli.NewHarness(t)
	defer h.Stop()

	manifestVersion, err := a.getBuildVersion(h.Env)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, manifestVersion, 2778)

	a.Manifest = ""
	apkVersion, err := a.getBuildVersion(h.Env)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, apkVersion, 2778)
}
