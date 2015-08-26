package main

import (
	"regexp"
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
