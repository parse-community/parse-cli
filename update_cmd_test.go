package main

import (
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/facebookgo/ensure"
	"github.com/facebookgo/jsonpipe"
	"github.com/facebookgo/parse"
)

func TestLatestVersion(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	defer h.Stop()

	ht := transportFunc(func(r *http.Request) (*http.Response, error) {
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
	h.env.ParseAPIClient = &ParseAPIClient{apiClient: &parse.Client{Transport: ht}}
	u := new(updateCmd)

	latestVersion, err := u.latestVersion(h.env)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, latestVersion, "2.0.2")
}
