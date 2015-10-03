package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/ParsePlatform/parse-cli/parsecli"
	"github.com/facebookgo/ensure"
)

func newDefaultParseConfig(t testing.TB, h *parsecli.Harness) *parsecli.ParseConfig {
	c, err := parsecli.ConfigFromDir(h.Env.Root)
	ensure.Nil(t, err)

	config, ok := c.(*parsecli.ParseConfig)
	ensure.True(t, ok)
	config.Applications = map[string]*parsecli.ParseAppConfig{
		"first":             {},
		"second":            {},
		parsecli.DefaultKey: {Link: "first"},
	}
	config.ProjectConfig = &parsecli.ProjectConfig{
		Type:  parsecli.LegacyParseFormat,
		Parse: &parsecli.ParseProjectConfig{},
	}
	ensure.Nil(t, parsecli.StoreConfig(h.Env, config))

	return config
}

func newDefaultCmdHarness(t testing.TB) (*parsecli.Harness, *defaultCmd, *parsecli.ParseConfig) {
	h := parsecli.NewHarness(t)
	h.MakeEmptyRoot()

	ensure.Nil(t, parsecli.CloneSampleCloudCode(h.Env, true))

	config := newDefaultParseConfig(t, h)
	h.Out.Reset()

	d := &defaultCmd{}
	return h, d, config
}

func TestPrintDefaultNotSet(t *testing.T) {
	t.Parallel()
	h, _, _ := newDefaultCmdHarness(t)
	defer h.Stop()
	ensure.Err(t, parsecli.PrintDefault(h.Env, ""), regexp.MustCompile("No app is set as default app"))
}

func TestPrintDefaultSet(t *testing.T) {
	t.Parallel()
	h, _, _ := newDefaultCmdHarness(t)
	defer h.Stop()
	parsecli.PrintDefault(h.Env, "first")
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
		parsecli.SetDefault(h.Env, "invalid", "", config),
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
	ensure.Nil(t, parsecli.SetDefault(h.Env, "first", "", config))
	ensure.DeepEqual(t, h.Out.String(), "Default app set to first.\n")
}

func TestSetDefaultWithApp(t *testing.T) {
	t.Parallel()
	h, _, config := newDefaultCmdHarness(t)
	defer h.Stop()
	ensure.Nil(t, parsecli.SetDefault(h.Env, "first", "first", config))
	ensure.DeepEqual(t, h.Out.String(), "Default app set to first.\n")
}

func TestDefaultRunMoreArgs(t *testing.T) {
	t.Parallel()
	h, d, _ := newDefaultCmdHarness(t)
	defer h.Stop()
	ensure.Err(t, d.run(h.Env, []string{"one", "two"}), regexp.MustCompile(`unexpected arguments, only an optional app name is expected:`))
}

// NOTE: testing for legacy format
func newLegacyDefaultCmdHarness(t testing.TB) (*parsecli.Harness, *defaultCmd, *parsecli.ParseConfig) {
	h := parsecli.NewHarness(t)
	h.MakeEmptyRoot()

	ensure.Nil(t, parsecli.CloneSampleCloudCode(h.Env, true))
	ensure.Nil(t, os.MkdirAll(filepath.Join(h.Env.Root, parsecli.ConfigDir), 0755))
	ensure.Nil(t, ioutil.WriteFile(filepath.Join(h.Env.Root, parsecli.LegacyConfigFile),
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
		parsecli.SetDefault(h.Env, "invalid", "", config),
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
	ensure.Nil(t, parsecli.SetDefault(h.Env, "first", "", config))
	ensure.DeepEqual(t, h.Out.String(), "Default app set to first.\n")
}

func TestLegacySetDefaultWithApp(t *testing.T) {
	t.Parallel()
	h, _, config := newLegacyDefaultCmdHarness(t)
	defer h.Stop()
	ensure.Nil(t, parsecli.SetDefault(h.Env, "first", "first", config))
	ensure.DeepEqual(t, h.Out.String(), "Default app set to first.\n")
}
