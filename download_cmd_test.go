package main

import (
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/facebookgo/ensure"
	"github.com/facebookgo/parse"
)

var scriptPath = regexp.MustCompile("/1/scripts/.*")
var hostedPath = regexp.MustCompile("/1/hosted_files/.*")

func setupForDownload(t testing.TB) *Harness {
	h := createParseProject(t)
	ht := transportFunc(func(r *http.Request) (*http.Response, error) {
		ensure.DeepEqual(t, r.FormValue("version"), "version")
		ensure.DeepEqual(t, r.FormValue("checksum"), "checksum")
		switch {
		case scriptPath.MatchString(r.URL.Path):
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(strings.NewReader(`"content"`)),
			}, nil
		case hostedPath.MatchString(r.URL.Path):
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(strings.NewReader(`[]byte("content")`)),
			}, nil
		default:
			return &http.Response{
				StatusCode: http.StatusExpectationFailed,
				Body:       ioutil.NopCloser(strings.NewReader(`{"error": "something is wrong"}`)),
			}, nil
		}
	})

	h.env.Client = &Client{client: &parse.Client{Transport: ht}}
	return h
}

func getRelease() releaseFiles {
	return releaseFiles{
		UserFiles: map[string]map[string]string{
			"cloud": map[string]string{
				"main.js": "version",
			},
			"public": map[string]string{
				"index.html": "version",
			},
		},
		Checksums: map[string]map[string]string{
			"cloud": map[string]string{
				"main.js": "checksum",
			},
			"public": map[string]string{
				"index.html": "checksum",
			},
		},
	}
}

func TestDownloadHosted(t *testing.T) {
	t.Parallel()
	h := setupForDownload(t)
	defer h.Stop()

	d := downloadCmd{}
	release := getRelease()
	files := fileMap{}

	_, ok := files[h.env.Root+"/public/index.html"]
	ensure.False(t, ok)
	err := d.downloadHosted(h.env, release, files)
	ensure.Nil(t, err)
	_, ok = files[h.env.Root+"/public/index.html"]
	ensure.True(t, ok)
}

func TestDownloadScripts(t *testing.T) {
	t.Parallel()
	h := setupForDownload(t)
	defer h.Stop()

	d := downloadCmd{}
	release := getRelease()
	files := fileMap{}

	_, ok := files[h.env.Root+"/cloud/main.js"]
	ensure.False(t, ok)
	err := d.downloadScripts(h.env, release, files)
	ensure.Nil(t, err)
	_, ok = files[h.env.Root+"/cloud/main.js"]
	ensure.True(t, ok)
}

func TestDownloadWriteFiles(t *testing.T) {
	t.Parallel()
	h := setupForDownload(t)
	defer h.Stop()

	d := downloadCmd{}
	files := fileMap{
		"/scripts/main.js":  []byte("javascript"),
		"/cloud/index.html": []byte("html"),
	}
	err := d.writeFiles(files)
	ensure.Nil(t, err)
	for path, data := range files {
		file, err := os.Open(path)
		ensure.Nil(t, err)
		content := make([]byte, len(data))
		n, err := file.Read(content)
		ensure.Nil(t, err)
		ensure.DeepEqual(t, n, len(data))
		ensure.DeepEqual(t, content, data)
	}
}

func TestDownload(t *testing.T) {
	t.Parallel()
	h := setupForDownload(t)
	defer h.Stop()

	d := downloadCmd{}
	release := getRelease()
	files := fileMap{}

	err := d.downloadHosted(h.env, release, files)
	ensure.Nil(t, err)
	err = d.downloadScripts(h.env, release, files)
	ensure.Nil(t, err)
	err = d.writeFiles(files)
	ensure.Nil(t, err)
}
