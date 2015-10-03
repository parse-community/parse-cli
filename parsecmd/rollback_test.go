package parsecmd

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/ParsePlatform/parse-cli/parsecli"
	"github.com/facebookgo/ensure"
	"github.com/facebookgo/parse"
)

func newRollbackCmdHarness(t testing.TB) *parsecli.Harness {
	h := parsecli.NewHarness(t)
	ht := parsecli.TransportFunc(func(r *http.Request) (*http.Response, error) {
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
	h.Env.ParseAPIClient = &parsecli.ParseAPIClient{APIClient: &parse.Client{Transport: ht}}
	return h
}

func TestRollbackToPrevious(t *testing.T) {
	t.Parallel()
	var r rollbackCmd
	h := newRollbackCmdHarness(t)
	defer h.Stop()
	ensure.Nil(t, r.run(h.Env, nil))
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
	ensure.Nil(t, r.run(h.Env, nil))
	ensure.DeepEqual(t, h.Out.String(),
		`Rolling back to v1
Rolled back to version v1
`)
}

func TestRollbackError(t *testing.T) {
	t.Parallel()
	var r rollbackCmd
	h := parsecli.NewHarness(t)
	defer h.Stop()
	var res struct{ Error string }
	res.Error = "something is wrong"
	ht := parsecli.TransportFunc(func(r *http.Request) (*http.Response, error) {
		ensure.DeepEqual(t, r.URL.Path, "/1/deploy")
		return &http.Response{
			StatusCode: http.StatusExpectationFailed,
			Body:       ioutil.NopCloser(strings.NewReader(jsonStr(t, &res))),
		}, nil
	})
	h.Env.ParseAPIClient = &parsecli.ParseAPIClient{APIClient: &parse.Client{Transport: ht}}
	ensure.Err(t, r.run(h.Env, nil), regexp.MustCompile("something is wrong"))
}
