package main

import (
	"regexp"
	"testing"

	"github.com/facebookgo/ensure"
)

func createNewProject(t testing.TB, h *Harness) (*parseConfig, error) {
	var n newCmd
	ensure.Nil(t, n.cloneSampleCloudCode(h.env, &app{Name: "test"}, false))
	c, err := configFromDir(h.env.Root)
	if err != nil {
		return nil, err
	}
	config, ok := c.(*parseConfig)
	ensure.True(t, ok)
	config.Applications = map[string]*parseAppConfig{"first": {}, "second": {}, defaultKey: {Link: "first"}}
	config.projectConfig = &projectConfig{Type: legacy, Parse: &parseProjectConfig{}}
	return config, storeParseConfig(h.env, config)
}

func newDefaultCmdHarness(t testing.TB) (*Harness, *defaultCmd, *parseConfig) {
	h := newHarness(t)
	h.makeEmptyRoot()

	config, err := createNewProject(t, h)
	ensure.Nil(t, err)
	h.Out.Reset()

	d := &defaultCmd{}
	return h, d, config
}

func TestPrintDefaultNotSet(t *testing.T) {
	t.Parallel()
	h, d, _ := newDefaultCmdHarness(t)
	defer h.Stop()
	ensure.Err(t, d.printDefault(h.env, ""), regexp.MustCompile("No app is set as default app"))
}

func TestPrintDefaultSet(t *testing.T) {
	h, d, _ := newDefaultCmdHarness(t)
	defer h.Stop()
	d.printDefault(h.env, "first")
	ensure.DeepEqual(t, h.Out.String(),
		`Current default app is first
`)
}

func TestSetDefaultInvalid(t *testing.T) {
	t.Parallel()
	h, d, config := newDefaultCmdHarness(t)
	defer h.Stop()
	ensure.Err(t, d.setDefault(h.env, "invalid", "", config), regexp.MustCompile(`Invalid application name "invalid". Please select from the valid applications printed above.`))
	ensure.DeepEqual(t, h.Out.String(),
		`The following apps are associated with cloud code in the current directory:
* first
  second
`)
}

func TestSetDefault(t *testing.T) {
	t.Parallel()
	h, d, config := newDefaultCmdHarness(t)
	defer h.Stop()
	ensure.Nil(t, d.setDefault(h.env, "first", "", config))
	ensure.DeepEqual(t, h.Out.String(), "Default app set to first.\n")
}

func TestSetDefaultWithApp(t *testing.T) {
	t.Parallel()
	h, d, config := newDefaultCmdHarness(t)
	defer h.Stop()
	ensure.Nil(t, d.setDefault(h.env, "first", "first", config))
	ensure.DeepEqual(t, h.Out.String(), "Default app set to first.\n")
}

func TestDefaultRunMoreArgs(t *testing.T) {
	t.Parallel()
	h, d, _ := newDefaultCmdHarness(t)
	defer h.Stop()
	ensure.Err(t, d.run(h.env, []string{"one", "two"}), regexp.MustCompile(`unexpected arguments, only an optional app name is expected:`))
}
