package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/facebookgo/ensure"
)

func TestGetConfigFile(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	defer h.Stop()

	h.env.Type = legacyParseFormat
	ensure.DeepEqual(t, getConfigFile(h.env), filepath.Join(h.env.Root, legacyConfigFile))

	h.env.Type = parseFormat
	ensure.DeepEqual(t, getConfigFile(h.env), filepath.Join(h.env.Root, parseLocal))
}

func TestConfigFromDirFormatType(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.makeEmptyRoot()
	defer h.Stop()

	ensure.Nil(t, (&newCmd{}).cloneSampleCloudCode(h.env, &app{Name: "test"}, false, true))

	c, err := configFromDir(h.env.Root)
	ensure.Nil(t, err)

	ensure.DeepEqual(t, c.getProjectConfig().Type, parseFormat)

	ensure.Nil(t, os.MkdirAll(filepath.Join(h.env.Root, configDir), 0755))
	ensure.Nil(t, ioutil.WriteFile(filepath.Join(h.env.Root, legacyConfigFile),
		[]byte("{}"),
		0600),
	)
	c, err = configFromDir(h.env.Root)
	ensure.Nil(t, err)

	ensure.DeepEqual(t, c.getProjectConfig().Type, legacyParseFormat)
}
