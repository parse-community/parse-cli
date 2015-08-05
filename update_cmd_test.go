package main

import (
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/facebookgo/ensure"
	"github.com/facebookgo/parse"
)

func TestLatestVersion(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	defer h.Stop()

	ht := transportFunc(func(r *http.Request) (*http.Response, error) {
		var result string
		switch r.URL.String() {
		case windowsCliDownloadURL:
			result = "http://parse-cli.aws.com/hash/parse-windows-2.0.2.exe"
		case unixCliDownloadURL:
			result = "http://parse-cli.aws.com/hash/parse-linux-2.0.2"
		case macCliDownloadURL:
			result = "http://parse-cli.aws.com/hash/parse-osx-2.0.2"
		}
		resp := http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(strings.NewReader("Success!")),
		}
		if resp.Header == nil {
			resp.Header = make(http.Header)
		}
		resp.Header.Set("Location", result)
		return &resp, nil
	})
	h.env.Client = &Client{client: &parse.Client{Transport: ht}}
	u := new(updateCmd)

	version, err := u.latestVersion(h.env, unixCliDownloadURL)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, version, "2.0.2")

	version, err = u.latestVersion(h.env, macCliDownloadURL)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, version, "2.0.2")

	version, err = u.latestVersion(h.env, windowsCliDownloadURL)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, version, "2.0.2")
}
