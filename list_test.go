package main

import (
	"io/ioutil"
	"strings"
	"testing"

	"github.com/ParsePlatform/parse-cli/parsecli"
	"github.com/facebookgo/ensure"
)

func newListCmdHarness(t testing.TB) (*parsecli.Harness, *listCmd) {
	h := parsecli.NewHarness(t)
	l := &listCmd{}
	return h, l
}

func TestPrintListOneAppNoDefaultKey(t *testing.T) {
	t.Parallel()
	conf := &parsecli.ParseConfig{Applications: map[string]*parsecli.ParseAppConfig{"first": {}}}
	h := parsecli.NewHarness(t)
	defer h.Stop()
	conf.PrettyPrintApps(h.Env)
	ensure.DeepEqual(t, h.Out.String(), "The following apps are associated with cloud code in the current directory:\n  first\n")
}

func TestPrintListOneAppWithDefaultKey(t *testing.T) {
	t.Parallel()
	conf := &parsecli.ParseConfig{Applications: map[string]*parsecli.ParseAppConfig{"first": {}}}
	conf.Applications[parsecli.DefaultKey] = &parsecli.ParseAppConfig{Link: "first"}

	h := parsecli.NewHarness(t)
	defer h.Stop()
	conf.PrettyPrintApps(h.Env)
	ensure.DeepEqual(t, h.Out.String(), "The following apps are associated with cloud code in the current directory:\n* first\n")
}

func TestPrintListTwoAppsWithDefaultKey(t *testing.T) {
	t.Parallel()
	conf := &parsecli.ParseConfig{Applications: map[string]*parsecli.ParseAppConfig{"first": {}, "second": {}}}
	conf.Applications[parsecli.DefaultKey] = &parsecli.ParseAppConfig{Link: "first"}

	h := parsecli.NewHarness(t)
	defer h.Stop()
	conf.PrettyPrintApps(h.Env)
	ensure.DeepEqual(t, h.Out.String(), "The following apps are associated with cloud code in the current directory:\n* first\n  second\n")
}

func TestPrintListTwoAppsWithLinks(t *testing.T) {
	t.Parallel()
	conf := &parsecli.ParseConfig{Applications: map[string]*parsecli.ParseAppConfig{"first": {}, "second": {Link: "first"}}}
	conf.Applications[parsecli.DefaultKey] = &parsecli.ParseAppConfig{Link: "first"}
	h := parsecli.NewHarness(t)
	defer h.Stop()
	conf.PrettyPrintApps(h.Env)
	ensure.DeepEqual(t, h.Out.String(), "The following apps are associated with cloud code in the current directory:\n* first\n  second -> first\n")
}
func TestPrintListNoConfig(t *testing.T) {
	t.Parallel()
	h, l := newListCmdHarness(t)
	h.MakeEmptyRoot()
	defer h.Stop()
	ensure.NotNil(t, l.printListOfApps(h.Env))
	ensure.DeepEqual(t, h.Out.String(), "")
}

func TestPrintListNoApps(t *testing.T) {
	t.Parallel()
	h, l := newListCmdHarness(t)
	h.MakeEmptyRoot()
	defer h.Stop()
	parsecli.CloneSampleCloudCode(h.Env, true)
	ensure.Nil(t, l.printListOfApps(h.Env))
}

func TestRunListCmd(t *testing.T) {
	t.Parallel()

	h, _ := parsecli.NewAppHarness(t)
	defer h.Stop()

	l := &listCmd{}
	h.Env.In = ioutil.NopCloser(strings.NewReader("email\npassword\n"))
	ensure.Nil(t, l.run(h.Env, []string{"A"}))
	ensure.StringContains(t, h.Out.String(), `Properties of the app "A"`)
}
