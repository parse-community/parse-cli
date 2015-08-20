package main

import (
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/facebookgo/ensure"
	"github.com/facebookgo/parse"
)

func newJsSdkHarness(t testing.TB) *Harness {
	h := newHarness(t)
	ht := transportFunc(func(r *http.Request) (*http.Response, error) {
		ensure.DeepEqual(t, r.URL.Path, "/1/jsVersions")
		rows := jsSDKVersion{JS: []string{"1.2.8", "1.2.9", "1.2.10", "1.2.11", "0.2.0"}}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(strings.NewReader(jsonStr(t, rows))),
		}, nil
	})
	h.env.ParseAPIClient = &ParseAPIClient{apiClient: &parse.Client{Transport: ht}}
	return h
}

func newJsSdkHarnessError(t testing.TB) *Harness {
	h := newHarness(t)
	ht := transportFunc(func(r *http.Request) (*http.Response, error) {
		ensure.DeepEqual(t, r.URL.Path, "/1/jsVersions")
		return &http.Response{
			StatusCode: http.StatusExpectationFailed,
			Body:       ioutil.NopCloser(strings.NewReader(`{"error":"something is wrong"}`)),
		}, nil
	})
	h.env.ParseAPIClient = &ParseAPIClient{apiClient: &parse.Client{Transport: ht}}
	return h
}

func newJsSdkHarnessWithConfig(t testing.TB) (*Harness, *context) {
	h := newJsSdkHarness(t)
	h.makeEmptyRoot()

	ensure.Nil(t, (&newCmd{}).cloneSampleCloudCode(h.env, &app{Name: "test"}, false))
	h.Out.Reset()

	c, err := configFromDir(h.env.Root)
	ensure.Nil(t, err)

	config, ok := (c).(*parseConfig)
	ensure.True(t, ok)

	return h, &context{Config: config}
}

func TestGetAllJSVersions(t *testing.T) {
	t.Parallel()
	h := newJsSdkHarness(t)
	defer h.Stop()
	j := jsSDKCmd{}
	versions, err := j.getAllJSSdks(h.env)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, versions, []string{"1.2.11", "1.2.10", "1.2.9", "1.2.8", "0.2.0"})
}

func TestGetAllJSVersionsError(t *testing.T) {
	t.Parallel()
	h := newJsSdkHarnessError(t)
	defer h.Stop()
	j := jsSDKCmd{}
	_, err := j.getAllJSSdks(h.env)
	ensure.Err(t, err, regexp.MustCompile(`something is wrong`))
}

func TestPrintVersions(t *testing.T) {
	t.Parallel()
	h, c := newJsSdkHarnessWithConfig(t)
	defer h.Stop()
	j := jsSDKCmd{}
	ensure.Nil(t, j.printVersions(h.env, c))
	ensure.DeepEqual(t, h.Out.String(),
		`   1.2.11
   1.2.10
   1.2.9
   1.2.8
   0.2.0
`)
}

func TestSetVersionInvalid(t *testing.T) {
	t.Parallel()
	h, c := newJsSdkHarnessWithConfig(t)
	defer h.Stop()
	j := jsSDKCmd{newVersion: "1.2.12"}
	ensure.Err(t, j.setVersion(h.env, c), regexp.MustCompile("Invalid SDK version selected"))
}

func TestSetVersionNoneSelected(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	defer h.Stop()

	c := &context{Config: defaultParseConfig}
	var j jsSDKCmd

	c.Config.getProjectConfig().Parse.JSSDK = "1.2.1"
	ensure.Nil(t, j.getVersion(h.env, c))
	ensure.DeepEqual(t, "Current JavaScript SDK version is 1.2.1\n",
		h.Out.String())

	c.Config.getProjectConfig().Parse.JSSDK = ""
	ensure.Err(t, j.getVersion(h.env, c),
		regexp.MustCompile("JavaScript SDK version not set for this project."))
}

func TestSetValidVersion(t *testing.T) {
	t.Parallel()

	h, c := newJsSdkHarnessWithConfig(t)
	defer h.Stop()
	j := jsSDKCmd{newVersion: "1.2.11"}
	ensure.Nil(t, j.setVersion(h.env, c))
	ensure.DeepEqual(t, h.Out.String(), "Current JavaScript SDK version is 1.2.11\n")

	content, err := ioutil.ReadFile(filepath.Join(h.env.Root, parseProject))
	ensure.Nil(t, err)
	ensure.DeepEqual(t, string(content), `{
  "project_type": 1,
  "parse": {
    "jssdk": "1.2.11"
  }
}`)
}

// NOTE: testing for legacy config format
func newLegacyJsSdkHarnessWithConfig(t testing.TB) (*Harness, *context) {
	h := newJsSdkHarness(t)
	h.makeEmptyRoot()

	ensure.Nil(t, os.Mkdir(filepath.Join(h.env.Root, configDir), 0755))
	path := filepath.Join(h.env.Root, legacyConfigFile)
	ensure.Nil(t, ioutil.WriteFile(path,
		[]byte(`{
		"global": {
			"parseVersion" : "1.2.9"
		}
	}`),
		0600))

	c, err := configFromDir(h.env.Root)
	ensure.Nil(t, err)

	config, ok := (c).(*parseConfig)
	ensure.True(t, ok)

	return h, &context{Config: config}
}

func TestLegacySetValidVersion(t *testing.T) {
	t.Parallel()

	h, c := newLegacyJsSdkHarnessWithConfig(t)
	defer h.Stop()
	j := jsSDKCmd{newVersion: "1.2.11"}
	ensure.Nil(t, j.setVersion(h.env, c))
	ensure.DeepEqual(t, h.Out.String(), "Current JavaScript SDK version is 1.2.11\n")
}
