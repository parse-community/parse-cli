package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ParsePlatform/parse-cli/parsecli"
	"github.com/facebookgo/ensure"
	"github.com/facebookgo/parse"
)

var (
	defaultCredentials      = parsecli.Credentials{Email: "email", Password: "password"}
	defaultTokenCredentials = parsecli.Credentials{Email: "email", Token: "token"}
	defaultApps             = parsecli.Apps{Login: parsecli.Login{Credentials: defaultCredentials}}
	defaultAppsWithToken    = parsecli.Apps{Login: parsecli.Login{Credentials: defaultTokenCredentials}}
)

func jsonStr(t testing.TB, v interface{}) string {
	b, err := json.Marshal(v)
	ensure.Nil(t, err)
	return string(b)
}

func newAddCmdHarness(t testing.TB) (*parsecli.Harness, []*parsecli.App) {
	h := parsecli.NewHarness(t)
	defer h.Stop()

	apps := []*parsecli.App{
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
	res := map[string][]*parsecli.App{"results": apps}
	ht := parsecli.TransportFunc(func(r *http.Request) (*http.Response, error) {
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
	h.Env.ParseAPIClient = &parsecli.ParseAPIClient{APIClient: &parse.Client{Transport: ht}}
	return h, apps
}

func newAddCmdHarnessWithConfig(t testing.TB) (*parsecli.Harness, *addCmd, error) {
	a := addCmd{apps: &defaultApps}

	h, _ := newAddCmdHarness(t)
	h.MakeEmptyRoot()

	ensure.Nil(t, parsecli.CloneSampleCloudCode(h.Env, true))

	return h, &a, nil
}

func TestAddCmdGlobalMakeDefault(t *testing.T) {
	t.Parallel()
	h, a, err := newAddCmdHarnessWithConfig(t)
	defer h.Stop()
	a.MakeDefault = true
	ensure.Nil(t, err)
	h.Env.Type = parsecli.ParseFormat
	h.Env.In = ioutil.NopCloser(strings.NewReader("1\n"))
	ensure.Nil(t, a.run(h.Env, []string{}))

	content, err := ioutil.ReadFile(filepath.Join(h.Env.Root, parsecli.ParseLocal))
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
	h.Env.Type = parsecli.ParseFormat
	h.Env.In = ioutil.NopCloser(strings.NewReader("1\n"))
	ensure.Nil(t, a.run(h.Env, []string{}))

	content, err := ioutil.ReadFile(filepath.Join(h.Env.Root, parsecli.ParseLocal))
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

func newLegacyAddCmdHarnessWithConfig(t testing.TB) (*parsecli.Harness, *addCmd, error) {
	a := addCmd{apps: &defaultApps}

	h, _ := newAddCmdHarness(t)
	h.MakeEmptyRoot()

	ensure.Nil(t, parsecli.CloneSampleCloudCode(h.Env, true))
	ensure.Nil(t, os.MkdirAll(filepath.Join(h.Env.Root, parsecli.ConfigDir), 0755))
	ensure.Nil(t,
		ioutil.WriteFile(
			filepath.Join(h.Env.Root, parsecli.LegacyConfigFile),
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
	h.Env.Type = parsecli.LegacyParseFormat
	h.Env.In = ioutil.NopCloser(strings.NewReader("1\n"))
	ensure.Nil(t, a.run(h.Env, []string{}))

	content, err := ioutil.ReadFile(filepath.Join(h.Env.Root, parsecli.LegacyConfigFile))
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
	h.Env.Type = parsecli.LegacyParseFormat
	h.Env.In = ioutil.NopCloser(strings.NewReader("1\n"))
	ensure.Nil(t, a.run(h.Env, []string{}))

	content, err := ioutil.ReadFile(filepath.Join(h.Env.Root, parsecli.LegacyConfigFile))
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
