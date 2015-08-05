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
		email, password := r.Header.Get("X-Parse-Email"), r.Header.Get("X-Parse-Password")
		if email == "email" && password == "password" {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(strings.NewReader(jsonStr(t, &res))),
			}, nil
		}
		return &http.Response{
			StatusCode: http.StatusUnauthorized,
			Body:       ioutil.NopCloser(strings.NewReader(`{"error": "incorrect username / password"}`)),
		}, nil
	})
	h.env.Client = &Client{client: &parse.Client{Transport: ht}}
	return h, apps
}

func newAddCmdHarnessWithConfig(t testing.TB) (*Harness, *addCmd, error) {
	a := addCmd{apps: &defaultApps}

	h, _ := newAddCmdHarness(t)
	h.makeEmptyRoot()

	if err := os.MkdirAll(filepath.Join(h.env.Root, configDir), 0755); err != nil {
		return nil, nil, err
	}

	if err := ioutil.WriteFile(
		filepath.Join(h.env.Root, legacyConfigFile),
		[]byte("{}"),
		0600); err != nil {
		return nil, nil, err
	}

	return h, &a, nil
}

func TestAddCmdGlobalMakeDefault(t *testing.T) {
	t.Parallel()
	h, a, err := newAddCmdHarnessWithConfig(t)
	defer h.Stop()
	a.MakeDefault = true
	ensure.Nil(t, err)
	h.env.In = ioutil.NopCloser(strings.NewReader("1\n"))
	ensure.Nil(t, a.run(h.env, []string{}))
}

func TestAddCmdGlobalNotDefault(t *testing.T) {
	t.Parallel()
	h, a, err := newAddCmdHarnessWithConfig(t)
	defer h.Stop()
	ensure.Nil(t, err)
	h.env.In = ioutil.NopCloser(strings.NewReader("1\n"))
	ensure.Nil(t, a.run(h.env, []string{}))
}
