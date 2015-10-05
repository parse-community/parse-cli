package parsecmd

import (
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/ParsePlatform/parse-cli/parsecli"
	"github.com/facebookgo/ensure"
	"github.com/facebookgo/parse"
	"github.com/facebookgo/stackerr"
)

func newReleasesCmdHarness(t testing.TB) (*parsecli.Harness, *releasesCmd) {
	h := parsecli.NewHarness(t)
	h.MakeEmptyRoot()
	r := &releasesCmd{}
	return h, r
}

func TestReleasesCmd(t *testing.T) {
	h, r := newReleasesCmdHarness(t)
	defer h.Stop()
	rows := []releasesResponse{
		{Version: "v1", Description: "version 1", Timestamp: "time 1"},
		{Version: "v2", Description: "version 2", Timestamp: "time 2"},
	}
	ht := parsecli.TransportFunc(func(r *http.Request) (*http.Response, error) {
		ensure.DeepEqual(t, r.URL.Path, "/1/releases")
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(strings.NewReader(jsonStr(t, rows))),
		}, nil
	})
	h.Env.ParseAPIClient = &parsecli.ParseAPIClient{APIClient: &parse.Client{Transport: ht}}

	ensure.Nil(t, r.run(h.Env, &parsecli.Context{}))

	expected := `Name                            Description                     Date
v1                              version 1                       time 1
v2                              version 2                       time 2
`
	ensure.DeepEqual(t, h.Out.String(), expected)
}

func TestReleasesCmdError(t *testing.T) {
	h, c := newReleasesCmdHarness(t)
	defer h.Stop()
	ht := parsecli.TransportFunc(func(r *http.Request) (*http.Response, error) {
		return nil, stackerr.New("Throws error")
	})
	h.Env.ParseAPIClient = &parsecli.ParseAPIClient{APIClient: &parse.Client{Transport: ht}}

	ensure.NotNil(t, c.run(h.Env, &parsecli.Context{}))
}

func TestReleasesCmdPrintVersion(t *testing.T) {
	h, r := newReleasesCmdHarness(t)
	releases := []releasesResponse{
		{Version: "v1",
			UserFiles: `{
			"cloud": {"main.js": "1", "app.js": "1", "views/index.js": "1"}
			}`,
		},
		{Version: "v2",
			UserFiles: `{
			"cloud": {"main.js": "2", "app.js": "2", "views/docs.js": "1"},
			"public": {"index.html": "2", "docs.html": "2"}
			}`,
		},
		{Version: "v2",
			UserFiles: `{
			"cloud": {"v2_main.js": "2", "v2_app.js": "2", "views/v2_docs.js": "2"},
			"public": {"v2_index.html": "2", "v2_docs.html": "2"}
			}`,
		},
	}
	err := r.printFiles("", releases, h.Env)
	ensure.Err(t, err, regexp.MustCompile("Unable to fetch files for release version"))

	h.Out.Reset()
	err = r.printFiles("v1", releases, h.Env)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, h.Out.String(), `Deployed Cloud Code files:
app.js
main.js
views/index.js
`)

	h.Out.Reset()
	err = r.printFiles("v2", releases, h.Env)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, h.Out.String(), `Deployed Cloud Code files:
app.js
main.js
views/docs.js

Deployed public hosting files:
docs.html
index.html
`)
}
