package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/ParsePlatform/parse-cli/parsecli"
	"github.com/facebookgo/ensure"
)

func TestUpgradeLegacyNoOp(t *testing.T) {
	t.Parallel()

	h := parsecli.NewHarness(t)
	defer h.Stop()

	var m migrateCmd
	c := &parsecli.ParseConfig{ProjectConfig: &parsecli.ProjectConfig{Type: parsecli.ParseFormat}}
	_, err := m.upgradeLegacy(h.Env, c)
	ensure.Err(t, err, regexp.MustCompile("Already using the preferred config format."))
}

func TestUpgradeLegacyRetainMaster(t *testing.T) {
	t.Parallel()

	h := parsecli.NewHarness(t)
	defer h.Stop()

	m := migrateCmd{retainMaster: true}
	c := &parsecli.ParseConfig{
		Applications: map[string]*parsecli.ParseAppConfig{
			"app": {ApplicationID: "a", MasterKey: "m"},
		},
		ProjectConfig: &parsecli.ProjectConfig{Type: parsecli.LegacyParseFormat},
	}
	config, err := m.upgradeLegacy(h.Env, c)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, config.Applications["app"].MasterKey, "m")
}

func TestUpgradeLegacy(t *testing.T) {
	t.Parallel()

	h := parsecli.NewHarness(t)
	defer h.Stop()

	m := migrateCmd{retainMaster: false}
	c := &parsecli.ParseConfig{
		Applications: map[string]*parsecli.ParseAppConfig{
			"app": {ApplicationID: "a", MasterKey: "m"},
		},
		ProjectConfig: &parsecli.ProjectConfig{Type: parsecli.LegacyParseFormat},
	}
	config, err := m.upgradeLegacy(h.Env, c)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, config.Applications["app"].MasterKey, "")
}

func TestUpgradeLegacyWithEmail(t *testing.T) {
	t.Parallel()

	h := parsecli.NewHarness(t)
	defer h.Stop()

	m := migrateCmd{retainMaster: false}
	c := &parsecli.ParseConfig{
		Applications: map[string]*parsecli.ParseAppConfig{
			"app": {ApplicationID: "a", MasterKey: "m"},
		},
		ProjectConfig: &parsecli.ProjectConfig{
			Type:        parsecli.LegacyParseFormat,
			ParserEmail: "test@email.com",
		},
	}
	config, err := m.upgradeLegacy(h.Env, c)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, config.Applications["app"].MasterKey, "")
	ensure.DeepEqual(t, config.GetProjectConfig().ParserEmail, "test@email.com")
}

func TestRunMigrateCmd(t *testing.T) {
	t.Parallel()

	h := parsecli.NewHarness(t)
	h.MakeEmptyRoot()
	defer h.Stop()

	n := &newCmd{}
	h.Env.Type = parsecli.ParseFormat
	h.Env.In = ioutil.NopCloser(strings.NewReader("\n"))
	_, err := n.setupSample(h.Env,
		"yolo",
		&parsecli.ParseAppConfig{
			ApplicationID: "yolo-id",
			MasterKey:     "yoda",
		},
		false,
		false,
	)
	ensure.Nil(t, err)

	m := &migrateCmd{}
	err = m.run(h.Env)
	ensure.Err(t, err, regexp.MustCompile("Already using the preferred config format"))

	ensure.Nil(t, os.Remove(filepath.Join(h.Env.Root, parsecli.ParseLocal)))
	ensure.Nil(t, os.Remove(filepath.Join(h.Env.Root, parsecli.ParseProject)))

	ensure.Nil(t, os.MkdirAll(filepath.Join(h.Env.Root, parsecli.ConfigDir), 0755))
	ensure.Nil(t,
		ioutil.WriteFile(
			filepath.Join(h.Env.Root, parsecli.LegacyConfigFile),
			[]byte(`{
			"applications": {
				"yolo": {
					"applicationId": "yolo-id",
					"masterKey": "yoda"
				}
			}
		}`),
			0600,
		),
	)

	ensure.Nil(t, m.run(h.Env))
	ensure.StringContains(t, h.Out.String(), "Successfully migrated")
}
