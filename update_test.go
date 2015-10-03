package main

import (
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/ParsePlatform/parse-cli/parsecli"
	"github.com/facebookgo/ensure"
	"github.com/facebookgo/jsonpipe"
	"github.com/facebookgo/parse"
)

func TestLatestVersion(t *testing.T) {
	t.Parallel()

	h := parsecli.NewHarness(t)
	defer h.Stop()

	ht := parsecli.TransportFunc(func(r *http.Request) (*http.Response, error) {
		ensure.DeepEqual(t, r.URL.Path, "/1/supported")
		return &http.Response{
			StatusCode: http.StatusOK,
			Body: ioutil.NopCloser(
				jsonpipe.Encode(
					map[string]string{"version": "2.0.2"},
				),
			),
		}, nil
	})
	h.Env.ParseAPIClient = &parsecli.ParseAPIClient{APIClient: &parse.Client{Transport: ht}}
	u := new(updateCmd)

	latestVersion, err := u.latestVersion(h.Env)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, latestVersion, "2.0.2")

	downloadURL, err := u.getDownloadURL(h.Env)
	ensure.StringContains(t,
		downloadURL, "https://github.com/ParsePlatform/parse-cli/releases/download/release_2.0.2")
}
