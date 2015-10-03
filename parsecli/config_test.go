package parsecli

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/facebookgo/ensure"
)

func TestGetConfigFile(t *testing.T) {
	t.Parallel()
	h := NewHarness(t)
	defer h.Stop()

	h.Env.Type = LegacyParseFormat
	ensure.DeepEqual(t, GetConfigFile(h.Env), filepath.Join(h.Env.Root, LegacyConfigFile))

	h.Env.Type = ParseFormat
	ensure.DeepEqual(t, GetConfigFile(h.Env), filepath.Join(h.Env.Root, ParseLocal))
}

func TestConfigFromDirFormatType(t *testing.T) {
	t.Parallel()
	h := NewHarness(t)
	h.MakeEmptyRoot()
	defer h.Stop()

	ensure.Nil(t, CloneSampleCloudCode(h.Env, true))

	c, err := ConfigFromDir(h.Env.Root)
	ensure.Nil(t, err)

	ensure.DeepEqual(t, c.GetProjectConfig().Type, ParseFormat)

	ensure.Nil(t, os.MkdirAll(filepath.Join(h.Env.Root, ConfigDir), 0755))
	ensure.Nil(t, ioutil.WriteFile(filepath.Join(h.Env.Root, LegacyConfigFile),
		[]byte("{}"),
		0600),
	)
	c, err = ConfigFromDir(h.Env.Root)
	ensure.Nil(t, err)

	ensure.DeepEqual(t, c.GetProjectConfig().Type, LegacyParseFormat)
}

func TestGetProjectRoot(t *testing.T) {
	t.Parallel()
	h := NewHarness(t)
	h.MakeEmptyRoot()
	defer h.Stop()

	ensure.Nil(t, os.Mkdir(filepath.Join(h.Env.Root, "parse"), 0755))
	ensure.Nil(t, os.Mkdir(filepath.Join(h.Env.Root, "parse", "config"), 0755))
	f, err := os.Create(filepath.Join(h.Env.Root, "parse", LegacyConfigFile))
	ensure.Nil(t, err)
	defer f.Close()
	ensure.Nil(t, os.Mkdir(filepath.Join(h.Env.Root, "parse", "cloud"), 0755))
	ensure.Nil(t, os.Mkdir(filepath.Join(h.Env.Root, "parse", "public"), 0755))
	ensure.Nil(t, os.MkdirAll(filepath.Join(h.Env.Root, "parse", "cloud", "other", "config"), 0755))

	ensure.DeepEqual(t, GetLegacyProjectRoot(h.Env, h.Env.Root), h.Env.Root)

	ensure.DeepEqual(t, GetLegacyProjectRoot(h.Env, filepath.Join(h.Env.Root, "parse", "config")), filepath.Join(h.Env.Root, "parse"))

	ensure.DeepEqual(t, GetLegacyProjectRoot(h.Env, filepath.Join(h.Env.Root, "parse", "cloud")), filepath.Join(h.Env.Root, "parse"))

	ensure.DeepEqual(t, GetLegacyProjectRoot(h.Env, filepath.Join(h.Env.Root, "parse", "public")), filepath.Join(h.Env.Root, "parse"))

	ensure.DeepEqual(t, GetLegacyProjectRoot(h.Env, filepath.Join(h.Env.Root, "parse", "cloud", "other")), filepath.Join(h.Env.Root, "parse"))
}
