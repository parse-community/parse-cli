package main

import (
	"io/ioutil"
	"os"
	"regexp"
	"testing"

	"github.com/facebookgo/ensure"
)

var defaultParseConfig = &parseConfig{
	projectConfig: &projectConfig{
		Type:  legacyParseFormat,
		Parse: &parseProjectConfig{},
	},
}

func TestConfigAddAlias(t *testing.T) {
	t.Parallel()
	const name = "foo"
	const link = "bar"
	c := parseConfig{Applications: map[string]*parseAppConfig{link: {}}}
	ensure.Nil(t, c.addAlias(name, link))
	ensure.DeepEqual(t, c.Applications, map[string]*parseAppConfig{
		link: {},
		name: {Link: link},
	})
}

func TestConfigAddAliasBadLink(t *testing.T) {
	t.Parallel()
	const name = "foo"
	const link = "bar"
	c := parseConfig{}
	ensure.Err(t, c.addAlias(name, link), regexp.MustCompile("wasn't found"))
}

func TestConfigAddAliasDupe(t *testing.T) {
	t.Parallel()
	const name = "foo"
	const link = "bar"
	c := parseConfig{Applications: map[string]*parseAppConfig{link: {}}}
	ensure.Nil(t, c.addAlias(name, link))
	ensure.Err(t, c.addAlias(name, link), regexp.MustCompile("has already been added"))
}

func TestConfigSetDefault(t *testing.T) {
	t.Parallel()
	const name = "foo"
	c := parseConfig{Applications: map[string]*parseAppConfig{name: {}}}
	ensure.Nil(t, c.setDefaultApp(name))
	ensure.DeepEqual(t, c.Applications, map[string]*parseAppConfig{
		name:       {},
		defaultKey: {Link: name},
	})
}

func TestConfigApp(t *testing.T) {
	t.Parallel()
	const name = "foo"
	c := parseConfig{Applications: map[string]*parseAppConfig{name: {}}}
	ac, err := c.app(name)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, ac, &parseAppConfig{})
}

func TestConfigAppNotFound(t *testing.T) {
	t.Parallel()
	const name = "foo"
	c := parseConfig{Applications: map[string]*parseAppConfig{}}
	ac, err := c.app(name)
	ensure.True(t, ac == nil)
	ensure.Err(t, err, regexp.MustCompile(`App "foo" wasn't found`))
}

func TestConfigAppDefaultNotFound(t *testing.T) {
	t.Parallel()
	c := parseConfig{Applications: map[string]*parseAppConfig{}}
	ac, err := c.app(defaultKey)
	ensure.True(t, ac == nil)
	ensure.Err(t, err, regexp.MustCompile("No default app configured"))
}

func TestConfigAppLink(t *testing.T) {
	t.Parallel()
	const name = "foo"
	const link = "bar"
	expected := &parseAppConfig{ApplicationID: "xyz"}
	c := parseConfig{Applications: map[string]*parseAppConfig{
		link: expected,
		name: {Link: link},
	}}
	ac, err := c.app(name)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, ac, expected)
}

func TestConfigMissing(t *testing.T) {
	t.Parallel()
	dir, err := ioutil.TempDir("", "parse-cli-config-")
	ensure.Nil(t, err)
	defer os.RemoveAll(dir)
	c, err := configFromDir(dir)
	ensure.True(t, c == nil)
	ensure.Err(t, err, regexp.MustCompile("Command must be run inside a Parse project."))
}

func TestConfigGlobalMalformed(t *testing.T) {
	t.Parallel()
	dir := makeDirWithConfig(t, "foo")
	defer os.RemoveAll(dir)
	c, err := configFromDir(dir)
	ensure.True(t, c == nil)
	ensure.Err(t, err, regexp.MustCompile("is not valid JSON"))
}

func TestConfigGlobalEmpty(t *testing.T) {
	t.Parallel()
	dir := makeDirWithConfig(t, "")
	defer os.RemoveAll(dir)
	c, err := configFromDir(dir)
	ensure.True(t, c == nil)
	ensure.Err(t, err, regexp.MustCompile("is not valid JSON"))
}

func TestConfigEmptyGlobalOnly(t *testing.T) {
	t.Parallel()
	dir := makeDirWithConfig(t, "{}")
	defer os.RemoveAll(dir)
	c, err := configFromDir(dir)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, c.getNumApps(), 0)
}
