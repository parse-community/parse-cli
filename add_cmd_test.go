package main

import (
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/facebookgo/ensure"
	"github.com/facebookgo/parse"
)

func newAddCmdHarness(t testing.TB) (*Harness, []*app) {
	h := newHarness(t)
	defer h.Stop()

	apps := []*app{
		{
			Name:          "A",
			ApplicationID: "appId.A",
			MasterKey:     "masterKey.A",
		},
		{
			Name:          "B",
			ApplicationID: "appId.B",
			MasterKey:     "masterKey.B",
		},
	}
	res := map[string][]*app{"results": apps}
	ht := transportFunc(func(r *http.Request) (*http.Response, error) {
		ensure.DeepEqual(t, r.URL.Path, "/1/apps")

		email := r.Header.Get("X-Parse-Email")
		password := r.Header.Get("X-Parse-Password")
		token := r.Header.Get("X-Parse-Account-Key")
		if !((email == "email" && password == "password") || (token == "token")) {
			return &http.Response{
				StatusCode: http.StatusUnauthorized,
				Body:       ioutil.NopCloser(strings.NewReader(`{"error": "incorrect credentials"}`)),
			}, nil
		}

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(strings.NewReader(jsonStr(t, &res))),
		}, nil

	})
	h.env.ParseAPIClient = &ParseAPIClient{apiClient: &parse.Client{Transport: ht}}
	return h, apps
}

func newAddCmdHarnessWithConfig(t testing.TB) (*Harness, *addCmd, error) {
	a := addCmd{apps: &defaultApps}

	h, _ := newAddCmdHarness(t)
	h.makeEmptyRoot()

	ensure.Nil(t, (&newCmd{}).cloneSampleCloudCode(h.env, &app{Name: "B"}, false, true))

	return h, &a, nil
}

func TestAddCmdGlobalMakeDefault(t *testing.T) {
	t.Parallel()
	h, a, err := newAddCmdHarnessWithConfig(t)
	defer h.Stop()
	a.MakeDefault = true
	ensure.Nil(t, err)
	h.env.Type = parseFormat
	h.env.In = ioutil.NopCloser(strings.NewReader("1\n"))
	ensure.Nil(t, a.run(h.env, []string{}))

	content, err := ioutil.ReadFile(filepath.Join(h.env.Root, parseLocal))
	ensure.Nil(t, err)
	ensure.DeepEqual(t, string(content), `{
  "applications": {
    "A": {
      "applicationId": "appId.A"
    },
    "_default": {
      "link": "A"
    }
  }
}`)
}

func TestAddCmdGlobalNotDefault(t *testing.T) {
	t.Parallel()
	h, a, err := newAddCmdHarnessWithConfig(t)
	defer h.Stop()
	ensure.Nil(t, err)
	h.env.Type = parseFormat
	h.env.In = ioutil.NopCloser(strings.NewReader("1\n"))
	ensure.Nil(t, a.run(h.env, []string{}))

	content, err := ioutil.ReadFile(filepath.Join(h.env.Root, parseLocal))
	ensure.Nil(t, err)
	ensure.DeepEqual(t, string(content), `{
  "applications": {
    "A": {
      "applicationId": "appId.A"
    }
  }
}`)
}

// NOTE: test functionality for legacy config format

func newLegacyAddCmdHarnessWithConfig(t testing.TB) (*Harness, *addCmd, error) {
	a := addCmd{apps: &defaultApps}

	h, _ := newAddCmdHarness(t)
	h.makeEmptyRoot()

	ensure.Nil(t, (&newCmd{}).cloneSampleCloudCode(h.env, &app{Name: "B"}, false, true))
	ensure.Nil(t, os.MkdirAll(filepath.Join(h.env.Root, configDir), 0755))
	ensure.Nil(t,
		ioutil.WriteFile(
			filepath.Join(h.env.Root, legacyConfigFile),
			[]byte("{}"),
			0600,
		),
	)
	return h, &a, nil
}

func TestLegacyAddCmdGlobalMakeDefault(t *testing.T) {
	t.Parallel()
	h, a, err := newLegacyAddCmdHarnessWithConfig(t)
	defer h.Stop()
	a.MakeDefault = true
	ensure.Nil(t, err)
	h.env.Type = legacyParseFormat
	h.env.In = ioutil.NopCloser(strings.NewReader("1\n"))
	ensure.Nil(t, a.run(h.env, []string{}))

	content, err := ioutil.ReadFile(filepath.Join(h.env.Root, legacyConfigFile))
	ensure.Nil(t, err)
	ensure.DeepEqual(t, string(content), `{
  "global": {},
  "applications": {
    "A": {
      "applicationId": "appId.A"
    },
    "_default": {
      "link": "A"
    }
  }
}`)
}

func TestLegacyAddCmdGlobalNotDefault(t *testing.T) {
	t.Parallel()
	h, a, err := newLegacyAddCmdHarnessWithConfig(t)
	defer h.Stop()
	ensure.Nil(t, err)
	h.env.Type = legacyParseFormat
	h.env.In = ioutil.NopCloser(strings.NewReader("1\n"))
	ensure.Nil(t, a.run(h.env, []string{}))

	content, err := ioutil.ReadFile(filepath.Join(h.env.Root, legacyConfigFile))
	ensure.Nil(t, err)
	ensure.DeepEqual(t, string(content), `{
  "global": {},
  "applications": {
    "A": {
      "applicationId": "appId.A"
    }
  }
}`)
}
