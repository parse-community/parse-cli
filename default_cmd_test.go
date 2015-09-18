package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/facebookgo/ensure"
)

func newDefaultParseConfig(t testing.TB, h *Harness) *parseConfig {
	c, err := configFromDir(h.env.Root)
	ensure.Nil(t, err)

	config, ok := c.(*parseConfig)
	ensure.True(t, ok)
	config.Applications = map[string]*parseAppConfig{"first": {}, "second": {}, defaultKey: {Link: "first"}}
	config.projectConfig = &projectConfig{Type: legacyParseFormat, Parse: &parseProjectConfig{}}
	ensure.Nil(t, storeConfig(h.env, config))

	return config
}

func newDefaultCmdHarness(t testing.TB) (*Harness, *defaultCmd, *parseConfig) {
	h := newHarness(t)
	h.makeEmptyRoot()

	ensure.Nil(t, (&newCmd{}).cloneSampleCloudCode(h.env, true))

	config := newDefaultParseConfig(t, h)
	h.Out.Reset()

	d := &defaultCmd{}
	return h, d, config
}

func TestPrintDefaultNotSet(t *testing.T) {
	t.Parallel()
	h, _, _ := newDefaultCmdHarness(t)
	defer h.Stop()
	ensure.Err(t, printDefault(h.env, ""), regexp.MustCompile("No app is set as default app"))
}

func TestPrintDefaultSet(t *testing.T) {
	t.Parallel()
	h, _, _ := newDefaultCmdHarness(t)
	defer h.Stop()
	printDefault(h.env, "first")
	ensure.DeepEqual(t, h.Out.String(),
		`Current default app is first
`)
}

func TestSetDefaultInvalid(t *testing.T) {
	t.Parallel()
	h, _, config := newDefaultCmdHarness(t)
	defer h.Stop()
	ensure.Err(
		t,
		setDefault(h.env, "invalid", "", config),
		regexp.MustCompile(`Invalid application name "invalid". Please select from the valid applications printed above.`),
	)
	ensure.DeepEqual(t, h.Out.String(),
		`The following apps are associated with cloud code in the current directory:
* first
  second
`)
}

func TestSetDefault(t *testing.T) {
	t.Parallel()
	h, _, config := newDefaultCmdHarness(t)
	defer h.Stop()
	ensure.Nil(t, setDefault(h.env, "first", "", config))
	ensure.DeepEqual(t, h.Out.String(), "Default app set to first.\n")
}

func TestSetDefaultWithApp(t *testing.T) {
	t.Parallel()
	h, _, config := newDefaultCmdHarness(t)
	defer h.Stop()
	ensure.Nil(t, setDefault(h.env, "first", "first", config))
	ensure.DeepEqual(t, h.Out.String(), "Default app set to first.\n")
}

func TestDefaultRunMoreArgs(t *testing.T) {
	t.Parallel()
	h, d, _ := newDefaultCmdHarness(t)
	defer h.Stop()
	ensure.Err(t, d.run(h.env, []string{"one", "two"}), regexp.MustCompile(`unexpected arguments, only an optional app name is expected:`))
}

// NOTE: testing for legacy format
func newLegacyDefaultCmdHarness(t testing.TB) (*Harness, *defaultCmd, *parseConfig) {
	h := newHarness(t)
	h.makeEmptyRoot()

	ensure.Nil(t, (&newCmd{}).cloneSampleCloudCode(h.env, true))
	ensure.Nil(t, os.MkdirAll(filepath.Join(h.env.Root, configDir), 0755))
	ensure.Nil(t, ioutil.WriteFile(filepath.Join(h.env.Root, legacyConfigFile),
		[]byte("{}"),
		0600),
	)

	config := newDefaultParseConfig(t, h)
	h.Out.Reset()

	d := &defaultCmd{}
	return h, d, config
}

func TestLegacySetDefaultInvalid(t *testing.T) {
	t.Parallel()
	h, _, config := newLegacyDefaultCmdHarness(t)
	defer h.Stop()
	ensure.Err(
		t,
		setDefault(h.env, "invalid", "", config),
		regexp.MustCompile(`Invalid application name "invalid". Please select from the valid applications printed above.`),
	)
	ensure.DeepEqual(t, h.Out.String(),
		`The following apps are associated with cloud code in the current directory:
* first
  second
`)
}

func TestLegacySetDefault(t *testing.T) {
	t.Parallel()
	h, _, config := newLegacyDefaultCmdHarness(t)
	defer h.Stop()
	ensure.Nil(t, setDefault(h.env, "first", "", config))
	ensure.DeepEqual(t, h.Out.String(), "Default app set to first.\n")
}

func TestLegacySetDefaultWithApp(t *testing.T) {
	t.Parallel()
	h, _, config := newLegacyDefaultCmdHarness(t)
	defer h.Stop()
	ensure.Nil(t, setDefault(h.env, "first", "first", config))
	ensure.DeepEqual(t, h.Out.String(), "Default app set to first.\n")
}
