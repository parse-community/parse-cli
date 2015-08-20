package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/facebookgo/ensure"
	"github.com/facebookgo/parse"
)

func newRollbackCmdHarness(t testing.TB) *Harness {
	h := newHarness(t)
	ht := transportFunc(func(r *http.Request) (*http.Response, error) {
		ensure.DeepEqual(t, r.URL.Path, "/1/deploy")
		var req, res rollbackInfo
		ensure.Nil(t, json.NewDecoder(r.Body).Decode(&req))
		if req.ReleaseName == "" {
			res.ReleaseName = "v0"
		} else {
			res.ReleaseName = req.ReleaseName
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(strings.NewReader(jsonStr(t, &res))),
		}, nil
	})
	h.env.ParseAPIClient = &ParseAPIClient{apiClient: &parse.Client{Transport: ht}}
	return h
}

func TestRollbackToPrevious(t *testing.T) {
	t.Parallel()
	var r rollbackCmd
	h := newRollbackCmdHarness(t)
	defer h.Stop()
	ensure.Nil(t, r.run(h.env, nil))
	ensure.DeepEqual(t, h.Out.String(),
		`Rolling back to previous release
Rolled back to version v0
`)
}

func TestRollbackToVersionOne(t *testing.T) {
	t.Parallel()
	r := rollbackCmd{ReleaseName: "v1"}
	h := newRollbackCmdHarness(t)
	defer h.Stop()
	ensure.Nil(t, r.run(h.env, nil))
	ensure.DeepEqual(t, h.Out.String(),
		`Rolling back to v1
Rolled back to version v1
`)
}

func TestRollbackError(t *testing.T) {
	t.Parallel()
	var r rollbackCmd
	h := newHarness(t)
	defer h.Stop()
	var res struct{ Error string }
	res.Error = "something is wrong"
	ht := transportFunc(func(r *http.Request) (*http.Response, error) {
		ensure.DeepEqual(t, r.URL.Path, "/1/deploy")
		return &http.Response{
			StatusCode: http.StatusExpectationFailed,
			Body:       ioutil.NopCloser(strings.NewReader(jsonStr(t, &res))),
		}, nil
	})
	h.env.ParseAPIClient = &ParseAPIClient{apiClient: &parse.Client{Transport: ht}}
	ensure.Err(t, r.run(h.env, nil), regexp.MustCompile("something is wrong"))
}
