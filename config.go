package main

import (
	"os"
	"path/filepath"

	"github.com/facebookgo/stackerr"
)

const (
	defaultKey = "_default"
	legacy     = iota + 1
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
	if e.Type == legacy {
		return filepath.Join(e.Root, legacyConfigFile)
	}
	return ""
}

func configFromDir(dir string) (config, error) {
	l, err := readLegacyConfigFile(filepath.Join(dir, legacyConfigFile))
	if err != nil {
		if stackerr.HasUnderlying(err, stackerr.MatcherFunc(os.IsNotExist)) {
			return nil, stackerr.New("Command must be run in a directory containing a Parse project.")
		}
		return nil, err
	}
	projectConfig := &projectConfig{
		Type:  legacy,
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

func storeProjectConfig(e *env, c config) error {
	projectType := c.getProjectConfig().Type
	if projectType == legacy {
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
	}

	return stackerr.Newf("Unknown project type: %s.", projectType)
}
