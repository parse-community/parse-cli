package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/facebookgo/ensure"
)

func TestUpgradeLegacyNoOp(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	defer h.Stop()

	var m migrateCmd
	c := &parseConfig{projectConfig: &projectConfig{Type: parseFormat}}
	_, err := m.upgradeLegacy(h.env, c)
	ensure.Err(t, err, regexp.MustCompile("Already using the preferred config format."))
}

func TestUpgradeLegacyRetainMaster(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	defer h.Stop()

	m := migrateCmd{retainMaster: true}
	c := &parseConfig{
		Applications: map[string]*parseAppConfig{
			"app": {ApplicationID: "a", MasterKey: "m"},
		},
		projectConfig: &projectConfig{Type: legacyParseFormat},
	}
	config, err := m.upgradeLegacy(h.env, c)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, config.Applications["app"].MasterKey, "m")
}

func TestUpgradeLegacy(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	defer h.Stop()

	m := migrateCmd{retainMaster: false}
	c := &parseConfig{
		Applications: map[string]*parseAppConfig{
			"app": {ApplicationID: "a", MasterKey: "m"},
		},
		projectConfig: &projectConfig{Type: legacyParseFormat},
	}
	config, err := m.upgradeLegacy(h.env, c)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, config.Applications["app"].MasterKey, "")
}

func TestUpgradeLegacyWithEmail(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	defer h.Stop()

	m := migrateCmd{retainMaster: false}
	c := &parseConfig{
		Applications: map[string]*parseAppConfig{
			"app": {ApplicationID: "a", MasterKey: "m"},
		},
		projectConfig: &projectConfig{
			Type:        legacyParseFormat,
			ParserEmail: "test@email.com",
		},
	}
	config, err := m.upgradeLegacy(h.env, c)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, config.Applications["app"].MasterKey, "")
	ensure.DeepEqual(t, config.getProjectConfig().ParserEmail, "test@email.com")
}

func TestRunMigrateCmd(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.makeEmptyRoot()
	defer h.Stop()

	n := &newCmd{}
	h.env.Type = parseFormat
	h.env.In = ioutil.NopCloser(strings.NewReader("\n"))
	_, err := n.setupSample(h.env,
		"yolo",
		&parseAppConfig{
			ApplicationID: "yolo-id",
			MasterKey:     "yoda",
		},
		false,
		false,
	)
	ensure.Nil(t, err)

	m := &migrateCmd{}
	err = m.run(h.env)
	ensure.Err(t, err, regexp.MustCompile("Already using the preferred config format"))

	ensure.Nil(t, os.Remove(filepath.Join(h.env.Root, parseLocal)))
	ensure.Nil(t, os.Remove(filepath.Join(h.env.Root, parseProject)))

	ensure.Nil(t, os.MkdirAll(filepath.Join(h.env.Root, configDir), 0755))
	ensure.Nil(t,
		ioutil.WriteFile(
			filepath.Join(h.env.Root, legacyConfigFile),
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

	ensure.Nil(t, m.run(h.env))
	ensure.StringContains(t, h.Out.String(), "Successfully migrated")
}
