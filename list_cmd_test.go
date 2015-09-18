package main

import (
	"testing"

	"github.com/facebookgo/ensure"
)

func newListCmdHarness(t testing.TB) (*Harness, *listCmd) {
	h := newHarness(t)
	l := &listCmd{}
	return h, l
}

func TestPrintListOneAppNoDefaultKey(t *testing.T) {
	t.Parallel()
	conf := &parseConfig{Applications: map[string]*parseAppConfig{"first": {}}}
	h := newHarness(t)
	defer h.Stop()
	conf.pprintApps(h.env)
	ensure.DeepEqual(t, h.Out.String(), "The following apps are associated with cloud code in the current directory:\n  first\n")
}

func TestPrintListOneAppWithDefaultKey(t *testing.T) {
	t.Parallel()
	conf := &parseConfig{Applications: map[string]*parseAppConfig{"first": {}}}
	conf.Applications[defaultKey] = &parseAppConfig{Link: "first"}

	h := newHarness(t)
	defer h.Stop()
	conf.pprintApps(h.env)
	ensure.DeepEqual(t, h.Out.String(), "The following apps are associated with cloud code in the current directory:\n* first\n")
}

func TestPrintListTwoAppsWithDefaultKey(t *testing.T) {
	t.Parallel()
	conf := &parseConfig{Applications: map[string]*parseAppConfig{"first": {}, "second": {}}}
	conf.Applications[defaultKey] = &parseAppConfig{Link: "first"}

	h := newHarness(t)
	defer h.Stop()
	conf.pprintApps(h.env)
	ensure.DeepEqual(t, h.Out.String(), "The following apps are associated with cloud code in the current directory:\n* first\n  second\n")
}

func TestPrintListTwoAppsWithLinks(t *testing.T) {
	t.Parallel()
	conf := &parseConfig{Applications: map[string]*parseAppConfig{"first": {}, "second": {Link: "first"}}}
	conf.Applications[defaultKey] = &parseAppConfig{Link: "first"}
	h := newHarness(t)
	defer h.Stop()
	conf.pprintApps(h.env)
	ensure.DeepEqual(t, h.Out.String(), "The following apps are associated with cloud code in the current directory:\n* first\n  second -> first\n")
}
func TestPrintListNoConfig(t *testing.T) {
	t.Parallel()
	h, l := newListCmdHarness(t)
	h.makeEmptyRoot()
	defer h.Stop()
	ensure.NotNil(t, l.printListOfApps(h.env))
	ensure.DeepEqual(t, h.Out.String(), "")
}

func TestPrintListNoApps(t *testing.T) {
	t.Parallel()
	h, l := newListCmdHarness(t)
	h.makeEmptyRoot()
	defer h.Stop()
	n := &newCmd{}
	n.cloneSampleCloudCode(h.env, &app{Name: "test"}, false, true)
	ensure.Nil(t, l.printListOfApps(h.env))
}
