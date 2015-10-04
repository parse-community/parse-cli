package parsecli

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/facebookgo/stackerr"
)

const ConfigDir = "config"

var LegacyConfigFile = filepath.Join(ConfigDir, "global.json")

type legacyConfig struct {
	Global struct {
		ParseVersion string `json:"parseVersion,omitempty"`
		ParserEmail  string `json:"email,omitempty"`
	} `json:"global,omitempty"`
	Applications map[string]*ParseAppConfig `json:"applications,omitempty"`
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

func GetLegacyProjectRoot(e *Env, cur string) string {
	if _, err := os.Stat(filepath.Join(cur, LegacyConfigFile)); err == nil {
		return cur
	}

	root := cur
	base := filepath.Base(root)

	for base != "." && base != string(filepath.Separator) {
		base = filepath.Base(root)
		root = filepath.Dir(root)
		if base == "cloud" || base == "public" || base == "config" {
			if _, err := os.Stat(filepath.Join(root, LegacyConfigFile)); err == nil {
				return root
			}
		}
	}

	return cur
}
