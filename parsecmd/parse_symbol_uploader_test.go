package parsecmd

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/ParsePlatform/parse-cli/parsecli"
	"github.com/facebookgo/ensure"
	"github.com/facebookgo/parse"
)

func createRandomFiles(t *testing.T, h *parsecli.Harness) {
	for _, filename := range []string{"a", "b"} {
		ensure.Nil(t, ioutil.WriteFile(filepath.Join(h.Env.Root, filename),
			[]byte(fmt.Sprintf(`Content of file : %s`, filename)), 0755))
	}
}

func TestBase64MD5OfFile(t *testing.T) {
	t.Parallel()
	h := parsecli.NewHarness(t)
	defer h.Stop()

	h.MakeEmptyRoot()
	testFile := filepath.Join(h.Env.Root, "test")
	ensure.Nil(t, ioutil.WriteFile(testFile,
		[]byte(`A test string`), 0755))

	val, err := base64MD5OfFile(testFile)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, val, "mc5NvcbbGrh2RDETriS4Fg==")
}

func TestBase64MD5OfFileNoFile(t *testing.T) {
	t.Parallel()
	h := parsecli.NewHarness(t)
	defer h.Stop()

	h.MakeEmptyRoot()
	testFile := filepath.Join(h.Env.Root, "test")
	_, err := base64MD5OfFile(testFile)
	ensure.NotNil(t, err)
}

func TestReadDirNames(t *testing.T) {
	t.Parallel()
	h := parsecli.NewHarness(t)
	defer h.Stop()

	h.MakeEmptyRoot()
	createRandomFiles(t, h)
	files, err := readDirNames(h.Env.Root)
	sort.Strings(files)
	ensure.DeepEqual(t, len(files), 2)
	ensure.DeepEqual(t, files[:], []string{
		filepath.Join(h.Env.Root, "a"),
		filepath.Join(h.Env.Root, "b"),
	})
	ensure.Nil(t, err)
}

func TestUploadFiles(t *testing.T) {
	t.Parallel()
	h := parsecli.NewHarness(t)
	defer h.Stop()

	h.MakeEmptyRoot()
	createRandomFiles(t, h)

	names := []string{"a", "b"}
	ht := parsecli.TransportFunc(func(r *http.Request) (*http.Response, error) {
		switch filepath.Base(r.URL.Path) {
		case names[0]:
			ensure.NotNil(t, r.Header)
			ensure.DeepEqual(t, r.Header.Get("Key"), "Value")
			ensure.DeepEqual(t, r.Header.Get("Content-Type"), "application/octet-stream")
			ensure.DeepEqual(t, r.Header.Get("Content-MD5"), "4JnleFGzGppuArF6N50EWg==")
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(strings.NewReader(`{"status":"success"}`)),
			}, nil
		case names[1]:
			ensure.NotNil(t, r.Header)
			ensure.DeepEqual(t, r.Header.Get("Key"), "Value")
			ensure.DeepEqual(t, r.Header.Get("Content-Type"), "application/octet-stream")
			ensure.DeepEqual(t, r.Header.Get("Content-MD5"), "Fv43qsp6mnGCJlC00VkOcA==")
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(strings.NewReader(`{"status":"success"}`)),
			}, nil
		default:
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       ioutil.NopCloser(strings.NewReader(`{"error":"something is wrong"}`)),
			}, nil
		}
	})
	h.Env.ParseAPIClient = &parsecli.ParseAPIClient{APIClient: &parse.Client{Transport: ht}}

	var filenames []string
	for _, name := range names {
		filenames = append(filenames, filepath.Join(h.Env.Root, name))
	}
	ensure.Nil(t, uploadSymbolFiles(filenames[:],
		map[string]string{"Key": "Value"}, true, h.Env))
	for _, filename := range filenames {
		_, err := os.Lstat(filename)
		ensure.NotNil(t, err)
		ensure.True(t, os.IsNotExist(err))
	}
}
