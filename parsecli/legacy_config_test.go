package parsecli

import (
	"io/ioutil"
	"os"
	"regexp"
	"testing"

	"github.com/facebookgo/ensure"
)

var defaultParseConfig = &ParseConfig{
	ProjectConfig: &ProjectConfig{
		Type:  LegacyParseFormat,
		Parse: &ParseProjectConfig{},
	},
}

func TestConfigAddAlias(t *testing.T) {
	t.Parallel()
	const name = "foo"
	const link = "bar"
	c := ParseConfig{Applications: map[string]*ParseAppConfig{link: {}}}
	ensure.Nil(t, c.AddAlias(name, link))
	ensure.DeepEqual(t, c.Applications, map[string]*ParseAppConfig{
		link: {},
		name: {Link: link},
	})
}

func TestConfigAddAliasBadLink(t *testing.T) {
	t.Parallel()
	const name = "foo"
	const link = "bar"
	c := ParseConfig{}
	ensure.Err(t, c.AddAlias(name, link), regexp.MustCompile("wasn't found"))
}

func TestConfigAddAliasDupe(t *testing.T) {
	t.Parallel()
	const name = "foo"
	const link = "bar"
	c := ParseConfig{Applications: map[string]*ParseAppConfig{link: {}}}
	ensure.Nil(t, c.AddAlias(name, link))
	ensure.Err(t, c.AddAlias(name, link), regexp.MustCompile("has already been added"))
}

func TestConfigSetDefault(t *testing.T) {
	t.Parallel()
	const name = "foo"
	c := ParseConfig{Applications: map[string]*ParseAppConfig{name: {}}}
	ensure.Nil(t, c.SetDefaultApp(name))
	ensure.DeepEqual(t, c.Applications, map[string]*ParseAppConfig{
		name:       {},
		DefaultKey: {Link: name},
	})
}

func TestConfigApp(t *testing.T) {
	t.Parallel()
	const name = "foo"
	c := ParseConfig{Applications: map[string]*ParseAppConfig{name: {}}}
	ac, err := c.App(name)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, ac, &ParseAppConfig{})
}

func TestConfigAppNotFound(t *testing.T) {
	t.Parallel()
	const name = "foo"
	c := ParseConfig{Applications: map[string]*ParseAppConfig{}}
	ac, err := c.App(name)
	ensure.True(t, ac == nil)
	ensure.Err(t, err, regexp.MustCompile(`App "foo" wasn't found`))
}

func TestConfigAppDefaultNotFound(t *testing.T) {
	t.Parallel()
	c := ParseConfig{Applications: map[string]*ParseAppConfig{}}
	ac, err := c.App(DefaultKey)
	ensure.True(t, ac == nil)
	ensure.Err(t, err, regexp.MustCompile("No default app configured"))
}

func TestConfigAppLink(t *testing.T) {
	t.Parallel()
	const name = "foo"
	const link = "bar"
	expected := &ParseAppConfig{ApplicationID: "xyz"}
	c := ParseConfig{Applications: map[string]*ParseAppConfig{
		link: expected,
		name: {Link: link},
	}}
	ac, err := c.App(name)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, ac, expected)
}

func TestConfigMissing(t *testing.T) {
	t.Parallel()
	dir, err := ioutil.TempDir("", "parse-cli-config-")
	ensure.Nil(t, err)
	defer os.RemoveAll(dir)
	c, err := ConfigFromDir(dir)
	ensure.True(t, c == nil)
	ensure.Err(t, err, regexp.MustCompile("Command must be run inside a Parse project."))
}

func TestConfigGlobalMalformed(t *testing.T) {
	t.Parallel()
	dir := makeDirWithConfig(t, "foo")
	defer os.RemoveAll(dir)
	c, err := ConfigFromDir(dir)
	ensure.True(t, c == nil)
	ensure.Err(t, err, regexp.MustCompile("is not valid JSON"))
}

func TestConfigGlobalEmpty(t *testing.T) {
	t.Parallel()
	dir := makeDirWithConfig(t, "")
	defer os.RemoveAll(dir)
	c, err := ConfigFromDir(dir)
	ensure.True(t, c == nil)
	ensure.Err(t, err, regexp.MustCompile("is not valid JSON"))
}

func TestConfigEmptyGlobalOnly(t *testing.T) {
	t.Parallel()
	dir := makeDirWithConfig(t, "{}")
	defer os.RemoveAll(dir)
	c, err := ConfigFromDir(dir)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, c.GetNumApps(), 0)
}
