package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/facebookgo/stackerr"
)

const (
	legacyParseFormat = iota
	parseFormat

	defaultKey   = "_default"
	parseLocal   = ".parse.local"
	parseProject = ".parse.project"
)

type appConfig interface {
	getApplicationID() string
	getMasterKey(e *env) (string, error)
	getApplicationAuth(e *env) (string, error)
	getLink() string
}

type config interface {
	app(string) (appConfig, error)
	addAlias(string, string) error
	setDefaultApp(string) error

	getProjectConfig() *projectConfig
	getDefaultApp() string
	getNumApps() int
	pprintApps(*env)
}

type projectConfig struct {
	// Type indicates the type of config.
	// Some examples are legacy, parse, etc.
	Type int `json:"project_type,omitempty"`
	// Parse stores the project config for Parse type.
	// Currently jssdk version is the only
	// project level config for Parse type.
	Parse *parseProjectConfig `json:"parse,omitempty"`
}

func getConfigFile(e *env) string {
	switch e.Type {
	case legacyParseFormat:
		return filepath.Join(e.Root, legacyConfigFile)
	case parseFormat:
		return filepath.Join(e.Root, parseLocal)
	}
	return ""
}

func configFromDir(dir string) (config, error) {
	l, err := readLegacyConfigFile(filepath.Join(dir, legacyConfigFile))
	if err != nil && !stackerr.HasUnderlying(err, stackerr.MatcherFunc(os.IsNotExist)) {
		return nil, err
	}
	if l != nil { // legacy config format
		projectConfig := &projectConfig{
			Type:  legacyParseFormat,
			Parse: &parseProjectConfig{JSSDK: l.Global.ParseVersion},
		}
		applications := l.Applications
		if applications == nil {
			applications = make(map[string]*parseAppConfig)
		}
		return &parseConfig{
			Applications:  applications,
			projectConfig: projectConfig,
		}, nil
	}

	canonicalize := func(err error) error {
		if err == nil {
			return nil
		}
		if stackerr.HasUnderlying(err, stackerr.MatcherFunc(os.IsNotExist)) {
			return stackerr.New("Command must be run inside a Parse project.")
		}
		return err
	}

	// current config format
	p, err := readProjectConfigFile(filepath.Join(dir, parseProject))
	if err != nil {
		return nil, canonicalize(err)
	}
	configFile := filepath.Join(dir, parseLocal)
	switch p.Type {
	case parseFormat:
		c, err := readParseConfigFile(configFile)
		if err != nil {
			return nil, canonicalize(err)
		}
		if c.Applications == nil {
			c.Applications = make(map[string]*parseAppConfig)
		}
		c.projectConfig = p
		return c, nil
	}

	return nil, stackerr.Newf("Unknown project type: %d.", p.Type)
}

func writeConfigFile(c config, path string) error {
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return stackerr.Wrap(err)
	}
	return stackerr.Wrap(ioutil.WriteFile(path, b, 0600))
}

func storeConfig(e *env, c config) error {
	projectType := c.getProjectConfig().Type
	switch projectType {
	case legacyParseFormat:
		pc, ok := (c).(*parseConfig)
		if !ok {
			return stackerr.Newf("Incorrect project type: 'legacy'.")
		}
		lconf := &legacyConfig{Applications: pc.Applications}
		lconf.Global.ParseVersion = pc.projectConfig.Parse.JSSDK
		return writeLegacyConfigFile(
			lconf,
			filepath.Join(e.Root, legacyConfigFile),
		)
	case parseFormat:
		return writeConfigFile(c, filepath.Join(e.Root, parseLocal))
	}
	return stackerr.Newf("Unknown project type: %d.", projectType)
}

func readProjectConfigFile(path string) (*projectConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, stackerr.Wrap(err)
	}
	defer f.Close()
	var p projectConfig
	if err := json.NewDecoder(f).Decode(&p); err != nil {
		return nil, stackerr.Newf("Config file %q is not valid JSON.", path)
	}
	return &p, err
}

func writeProjectConfig(p *projectConfig, path string) error {
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return stackerr.Wrap(err)
	}
	return stackerr.Wrap(ioutil.WriteFile(path, b, 0600))
}

func storeProjectConfig(e *env, c config) error {
	p := c.getProjectConfig()
	switch p.Type {
	case legacyParseFormat:
		p, ok := (c).(*parseConfig)
		if !ok {
			return stackerr.New("Invalid Cloud Code config.")
		}
		lconf := &legacyConfig{Applications: p.Applications}
		lconf.Global.ParseVersion = p.projectConfig.Parse.JSSDK
		return writeLegacyConfigFile(
			lconf,
			filepath.Join(e.Root, legacyConfigFile),
		)

	case parseFormat:
		return writeProjectConfig(p, filepath.Join(e.Root, parseProject))
	}

	return stackerr.Newf("Unknown project type: %d.", p.Type)
}

func isProjectDir(cur string) bool {
	_, err := os.Stat(filepath.Join(cur, parseProject))
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(cur, parseLocal))
	if err != nil {
		return false
	}
	return true
}

func getProjectRoot(e *env, cur string) string {
	if isProjectDir(cur) {
		return cur
	}

	root := cur
	base := filepath.Base(root)

	for base != "." && base != string(filepath.Separator) {
		base = filepath.Base(root)
		root = filepath.Dir(root)
		if isProjectDir(root) {
			return root
		}
	}

	return cur
}

func printDefault(e *env, defaultApp string) error {
	if defaultApp == "" {
		return stackerr.New("No app is set as default app")
	}
	fmt.Fprintf(e.Out, "Current default app is %s\n", defaultApp)
	return nil
}

func setDefault(e *env, newDefault, defaultApp string, c config) error {
	projectType := c.getProjectConfig().Type
	switch projectType {
	case legacyParseFormat, parseFormat:
		p, ok := c.(*parseConfig)
		if !ok {
			return stackerr.New("Invalid Cloud Code config.")
		}
		return setParseDefault(e, newDefault, defaultApp, p)
	}

	return stackerr.Newf("Unknown project type: %d.", projectType)
}
