package main

import (
	"fmt"
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
				"main.js": "9a0364b9e99bb480dd25e1f0284c8555",
			},
			"public": map[string]string{
				"index.html": "ea46dea1ca5f0b7a728aa3c2a87ae8a1",
			},
		},
	}
}

func TestDownloadDownload(t *testing.T) {
	t.Parallel()
	h := setupForDownload(t)
	defer h.Stop()

	d := downloadCmd{}
	release := getRelease()
	tempDir, err := ioutil.TempDir("", "download")
	ensure.Nil(t, err)
	fmt.Println("root", h.env.Root)
	fmt.Println("temp", tempDir)

	for file := range release.UserFiles[cloudDir] {
		_, err := os.Open(fmt.Sprintf("%s/%s/%s", tempDir, cloudDir, file))
		ensure.NotNil(t, err)
	}
	for file := range release.UserFiles[hostingDir] {
		_, err := os.Open(fmt.Sprintf("%s/%s/%s", tempDir, hostingDir, file))
		ensure.NotNil(t, err)
	}
	err = d.download(h.env, tempDir, release)
	ensure.Nil(t, err)
	for file := range release.UserFiles[cloudDir] {
		_, err := os.Open(fmt.Sprintf("%s/%s/%s", tempDir, cloudDir, file))
		ensure.Nil(t, err)
	}
	for file := range release.UserFiles[hostingDir] {
		_, err := os.Open(fmt.Sprintf("%s/%s/%s", tempDir, hostingDir, file))
		ensure.Nil(t, err)
	}
}

func TestDownloadMoveFiles(t *testing.T) {
	t.Parallel()
	h := setupForDownload(t)
	defer h.Stop()

	content := []byte("content")
	release := getRelease()
	release.Checksums[cloudDir]["main.js"] = "9a0364b9e99bb480dd25e1f0284c8555"
	release.Checksums[hostingDir]["index.html"] = "9a0364b9e99bb480dd25e1f0284c8555"

	tempDir, err := ioutil.TempDir("", "move-files")
	err = os.MkdirAll(fmt.Sprintf("%s/%s", tempDir, hostingDir),
		os.ModeDir|os.ModePerm)
	ensure.Nil(t, err)
	err = os.MkdirAll(fmt.Sprintf("%s/%s", tempDir, cloudDir),
		os.ModeDir|os.ModePerm)
	ensure.Nil(t, err)
	fmt.Println("temp", tempDir)
	ensure.Nil(t, err)

	err = ioutil.WriteFile(fmt.Sprintf("%s/%s/index.html", tempDir, hostingDir),
		content,
		os.ModePerm)
	ensure.Nil(t, err)
	err = ioutil.WriteFile(fmt.Sprintf("%s/%s/main.js", tempDir, cloudDir),
		content,
		os.ModePerm)
	ensure.Nil(t, err)

	d := downloadCmd{}
	err = d.moveFiles(h.env, tempDir, release)
	ensure.Nil(t, err)

	readData, err := ioutil.ReadFile(fmt.Sprintf("%s/%s/%s", h.env.Root, hostingDir, "index.html"))
	ensure.DeepEqual(t, readData, content)
	readData, err = ioutil.ReadFile(fmt.Sprintf("%s/%s/%s", h.env.Root, cloudDir, "main.js"))
	ensure.DeepEqual(t, readData, content)
}

func TestDownload(t *testing.T) {
	t.Parallel()
	h := setupForDownload(t)
	defer h.Stop()

	release := getRelease()
	tempDir := os.TempDir()

	d := downloadCmd{}
	err := d.download(h.env, tempDir, release)
	ensure.Nil(t, err)
	err = d.moveFiles(h.env, tempDir, release)
	ensure.Nil(t, err)
}
