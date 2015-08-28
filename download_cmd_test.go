package main

import (
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/facebookgo/ensure"
	"github.com/facebookgo/parse"
)

var (
	scriptPath = regexp.MustCompile("/1/scripts/.*")
	hostedPath = regexp.MustCompile("/1/hosted_files/.*")
)

func newDownloadHarness(t testing.TB) (*Harness, *downloadCmd) {
	h := createParseProject(t)
	ht := transportFunc(func(r *http.Request) (*http.Response, error) {
		ensure.DeepEqual(t, r.FormValue("version"), "version")
		switch {
		case scriptPath.MatchString(r.URL.Path):
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(strings.NewReader(`"content"`)),
			}, nil
		case hostedPath.MatchString(r.URL.Path):
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(strings.NewReader(`[1, 1, 2, 3, 5, 8, 13]`)),
			}, nil
		default:
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       ioutil.NopCloser(strings.NewReader(`{"error": "something is wrong"}`)),
			}, nil
		}
	})

	h.makeEmptyRoot()
	h.env.ParseAPIClient = &ParseAPIClient{apiClient: &parse.Client{Transport: ht}}

	d := &downloadCmd{
		release: &deployInfo{
			Versions: deployFileData{
				Cloud:  map[string]string{"main.js": "version"},
				Public: map[string]string{"index.html": "version"},
			},
			Checksums: deployFileData{
				Cloud:  map[string]string{"main.js": "9a0364b9e99bb480dd25e1f0284c8555"},
				Public: map[string]string{"index.html": "ea46dea1ca5f0b7a728aa3c2a87ae8a1"},
			},
		},
	}

	return h, d
}

func TestDownload(t *testing.T) {
	t.Parallel()

	h, d := newDownloadHarness(t)
	defer h.Stop()

	tempDir, err := ioutil.TempDir("", "download_")
	ensure.Nil(t, err)
	defer func() {
		os.RemoveAll(tempDir)
	}()

	err = d.download(h.env, tempDir, d.release)
	ensure.Nil(t, err)
	for file := range d.release.Versions.Cloud {
		_, err := os.Open(filepath.Join(tempDir, cloudDir, file))
		ensure.Nil(t, err)
	}
	for file := range d.release.Versions.Public {
		_, err := os.Open(filepath.Join(tempDir, hostingDir, file))
		ensure.Nil(t, err)
	}
}

func TestDownloadMoveFiles(t *testing.T) {
	t.Parallel()
	h, d := newDownloadHarness(t)
	defer h.Stop()

	content := []byte("content")
	d.release.Checksums.Cloud["main.js"] = "9a0364b9e99bb480dd25e1f0284c8555"
	d.release.Checksums.Public["index.html"] = "9a0364b9e99bb480dd25e1f0284c8555"

	tempDir, err := ioutil.TempDir("", "move_files_")
	defer func() {
		os.RemoveAll(tempDir)
	}()

	err = os.MkdirAll(
		filepath.Join(tempDir, hostingDir),
		0755,
	)
	ensure.Nil(t, err)
	err = os.MkdirAll(
		filepath.Join(tempDir, cloudDir),
		0755,
	)
	ensure.Nil(t, err)

	err = ioutil.WriteFile(
		filepath.Join(tempDir, hostingDir, "index.html"),
		content,
		0644,
	)
	ensure.Nil(t, err)
	err = ioutil.WriteFile(
		filepath.Join(tempDir, cloudDir, "main.js"),
		content,
		0644,
	)
	ensure.Nil(t, err)

	err = d.moveFiles(h.env, tempDir, d.release)
	ensure.Nil(t, err)

	readData, err := ioutil.ReadFile(
		filepath.Join(h.env.Root, hostingDir, "index.html"),
	)
	ensure.DeepEqual(t, readData, content)
	readData, err = ioutil.ReadFile(
		filepath.Join(h.env.Root, cloudDir, "main.js"),
	)
	ensure.DeepEqual(t, readData, content)
}
