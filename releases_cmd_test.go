package main

import (
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/facebookgo/ensure"
	"github.com/facebookgo/parse"
	"github.com/facebookgo/stackerr"
)

func newReleasesCmdHarness(t testing.TB) (*Harness, *releasesCmd) {
	h := newHarness(t)
	h.makeEmptyRoot()
	r := &releasesCmd{}
	return h, r
}

func TestReleasesCmd(t *testing.T) {
	h, r := newReleasesCmdHarness(t)
	defer h.Stop()
	rows := []deployInfo{
		{ParseVersion: "v1", Description: "version 1", Timestamp: "time 1"},
		{ParseVersion: "v2", Description: "version 2", Timestamp: "time 2"},
	}
	ht := transportFunc(func(r *http.Request) (*http.Response, error) {
		ensure.DeepEqual(t, r.URL.Path, "/1/releases")
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(strings.NewReader(jsonStr(t, rows))),
		}, nil
	})
	h.env.ParseAPIClient = &ParseAPIClient{apiClient: &parse.Client{Transport: ht}}

	ensure.Nil(t, r.run(h.env, &context{}))

	expected := `Name                            Description                     Date
v1                              version 1                       time 1
v2                              version 2                       time 2
`
	ensure.DeepEqual(t, h.Out.String(), expected)
}

func TestReleasesCmdError(t *testing.T) {
	h, c := newReleasesCmdHarness(t)
	defer h.Stop()
	ht := transportFunc(func(r *http.Request) (*http.Response, error) {
		return nil, stackerr.New("Throws error")
	})
	h.env.ParseAPIClient = &ParseAPIClient{apiClient: &parse.Client{Transport: ht}}

	ensure.NotNil(t, c.run(h.env, &context{}))
}

func TestReleasesCmdPrintVersion(t *testing.T) {
	h, r := newReleasesCmdHarness(t)
	releases := []deployInfo{
		{
			ParseVersion: "v1",
			Versions: deployFileData{
				Cloud: map[string]string{
					"main.js":        "1",
					"app.js":         "1",
					"views/index.js": "1",
				},
			},
		},
		{
			ParseVersion: "v2",
			Versions: deployFileData{
				Cloud: map[string]string{
					"main.js":       "2",
					"app.js":        "2",
					"views/docs.js": "1",
				},
				Public: map[string]string{
					"index.html": "2",
					"docs.html":  "2",
				},
			},
		},
		{
			ParseVersion: "v2",
			Versions: deployFileData{
				Cloud: map[string]string{
					"v2_main.js":       "2",
					"v2_app.js":        "2",
					"views/v2_docs.js": "2",
				},
				Public: map[string]string{
					"v2_index.html": "2",
					"v2_docs.html":  "2",
				},
			},
		},
	}
	err := r.printFiles("", releases, h.env)
	ensure.Err(t, err, regexp.MustCompile("Unable to fetch files for release version"))

	h.Out.Reset()
	err = r.printFiles("v1", releases, h.env)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, h.Out.String(), `Deployed cloud code files:
app.js
main.js
views/index.js
`)

	h.Out.Reset()
	err = r.printFiles("v2", releases, h.env)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, h.Out.String(), `Deployed cloud code files:
app.js
main.js
views/docs.js

Deployed public hosting files:
docs.html
index.html
`)
}
