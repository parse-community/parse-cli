package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/facebookgo/stackerr"
)

const configDir = "config"

var legacyConfigFile = filepath.Join(configDir, "global.json")

type legacyConfig struct {
	Global struct {
		ParseVersion string `json:"parseVersion,omitempty"`
	} `json:"global,omitempty"`
	Applications map[string]*parseAppConfig `json:"applications,omitempty"`
}

func readLegacyConfigFile(path string) (*legacyConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, stackerr.Wrap(err)
	}
	defer f.Close()
	var c legacyConfig
	if err := json.NewDecoder(f).Decode(&c); err != nil {
		return nil, stackerr.Newf("Config file %q is not valid JSON.", path)
	}
	return &c, nil
}

func writeLegacyConfigFile(c *legacyConfig, path string) error {
	// if config directory does not exist yet, first create it
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return stackerr.Wrap(err)
	}

	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return stackerr.Wrap(err)
	}
	return stackerr.Wrap(ioutil.WriteFile(path, b, 0600))
}

func getLegacyProjectRoot(e *env, cur string) string {
	if _, err := os.Stat(filepath.Join(cur, legacyConfigFile)); err == nil {
		return cur
	}

	root := cur
	base := filepath.Base(root)

	for base != "." && base != string(filepath.Separator) {
		base = filepath.Base(root)
		root = filepath.Dir(root)
		if base == "cloud" || base == "public" || base == "config" {
			if _, err := os.Stat(filepath.Join(root, legacyConfigFile)); err == nil {
				return root
			}
		}
	}

	return cur
}
