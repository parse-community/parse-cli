package main

import (
	"io/ioutil"
	"net/http"
	"regexp"
	"testing"

	"github.com/ParsePlatform/parse-cli/parsecli"
	"github.com/facebookgo/ensure"
	"github.com/facebookgo/jsonpipe"
	"github.com/facebookgo/parse"
)

func TestNewClientInvalidServerURL(t *testing.T) {
	t.Parallel()
	c, err := parsecli.NewParseAPIClient(&parsecli.Env{Server: ":"})
	ensure.True(t, c == nil)
	ensure.Err(t, err, regexp.MustCompile("invalid server URL"))
}

func TestIsSupportedWarning(t *testing.T) {
	t.Parallel()

	h := parsecli.NewHarness(t)
	defer h.Stop()

	ht := parsecli.TransportFunc(func(r *http.Request) (*http.Response, error) {
		ensure.DeepEqual(t, r.URL.Path, "/1/supported")
		return &http.Response{
			StatusCode: http.StatusOK,
			Body: ioutil.NopCloser(
				jsonpipe.Encode(
					map[string]string{"warning": "please update"},
				),
			),
		}, nil
	})
	h.Env.ParseAPIClient = &parsecli.ParseAPIClient{APIClient: &parse.Client{Transport: ht}}
	message, err := checkIfSupported(h.Env, "2.0.2")
	ensure.Nil(t, err)
	ensure.DeepEqual(t, message, "please update")
}

func TestIsSupportedError(t *testing.T) {
	t.Parallel()

	h := parsecli.NewHarness(t)
	defer h.Stop()

	ht := parsecli.TransportFunc(func(r *http.Request) (*http.Response, error) {
		ensure.DeepEqual(t, r.URL.Path, "/1/supported")
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Body: ioutil.NopCloser(
				jsonpipe.Encode(
					map[string]string{"error": "not supported"},
				),
			),
		}, nil
	})
	h.Env.ParseAPIClient = &parsecli.ParseAPIClient{APIClient: &parse.Client{Transport: ht}}
	_, err := checkIfSupported(h.Env, "2.0.2")
	ensure.Err(t, err, regexp.MustCompile("not supported"))
}

// this just exists to make code coverage less noisy.
func TestRootCommand(t *testing.T) {
	t.Parallel()
	h := parsecli.NewHarness(t)
	defer h.Stop()
	parseRootCmd(h.Env)
}
