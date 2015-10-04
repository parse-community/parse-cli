package parsecli

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/facebookgo/stackerr"
)

const (
	LegacyParseFormat = iota
	ParseFormat
	HerokuFormat

	ParseLocal   = ".parse.local"
	ParseProject = ".parse.project"
	DefaultKey   = "_default"
)

type AppConfig interface {
	GetApplicationID() string
	GetMasterKey(e *Env) (string, error)
	GetApplicationAuth(e *Env) (string, error)
	GetLink() string
}

type Config interface {
	App(string) (AppConfig, error)
	AddAlias(string, string) error
	SetDefaultApp(string) error

	GetProjectConfig() *ProjectConfig
	GetDefaultApp() string
	GetNumApps() int
	PrettyPrintApps(*Env)
}

type ProjectConfig struct {
	// Type indicates the type of config.
	// Some examples are legacy, parse, etc.
	Type int `json:"project_type,omitempty"`
	// Parse stores the project config for Parse type.
	// Currently jssdk version is the only
	// project level config for Parse type.
	Parse *ParseProjectConfig `json:"parse,omitempty"`
	// ParserEmail is an email id of the Parse developer.
	// It is associated with this project.
	// It is used to fetch appropriate credentials from netrc.
	ParserEmail string `json:"email,omitempty"`
}

func GetConfigFile(e *Env) string {
	switch e.Type {
	case LegacyParseFormat:
		return filepath.Join(e.Root, LegacyConfigFile)
	case ParseFormat:
		return filepath.Join(e.Root, ParseLocal)
	}
	return ""
}

func ConfigFromDir(dir string) (Config, error) {
	l, err := readLegacyConfigFile(filepath.Join(dir, LegacyConfigFile))
	if err != nil && !stackerr.HasUnderlying(err, stackerr.MatcherFunc(os.IsNotExist)) {
		return nil, err
	}
	if l != nil { // legacy config format
		projectConfig := &ProjectConfig{
			Type: LegacyParseFormat,
			Parse: &ParseProjectConfig{
				JSSDK: l.Global.ParseVersion,
			},
			ParserEmail: l.Global.ParserEmail,
		}
		applications := l.Applications
		if applications == nil {
			applications = make(map[string]*ParseAppConfig)
		}
		return &ParseConfig{
			Applications:  applications,
			ProjectConfig: projectConfig,
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
	p, err := readProjectConfigFile(filepath.Join(dir, ParseProject))
	if err != nil {
		return nil, canonicalize(err)
	}
	configFile := filepath.Join(dir, ParseLocal)
	switch p.Type {
	case ParseFormat:
		c, err := readParseConfigFile(configFile)
		if err != nil {
			return nil, canonicalize(err)
		}
		if c.Applications == nil {
			c.Applications = make(map[string]*ParseAppConfig)
		}
		c.ProjectConfig = p
		return c, nil

	case HerokuFormat:
		c, err := readHerokuConfigFile(configFile)
		if err != nil {
			return nil, canonicalize(err)
		}
		if c.Applications == nil {
			c.Applications = make(map[string]*HerokuAppConfig)
		}
		c.ProjectConfig = p
		return c, nil
	}

	return nil, stackerr.Newf("Unknown project type: %d.", p.Type)
}

func writeConfigFile(c Config, path string) error {
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return stackerr.Wrap(err)
	}
	return stackerr.Wrap(ioutil.WriteFile(path, b, 0600))
}

func StoreConfig(e *Env, c Config) error {
	projectType := c.GetProjectConfig().Type
	switch projectType {
	case LegacyParseFormat:
		pc, ok := (c).(*ParseConfig)
		if !ok {
			return stackerr.Newf("Incorrect project type: 'legacy'.")
		}
		lconf := &legacyConfig{Applications: pc.Applications}
		lconf.Global.ParseVersion = pc.ProjectConfig.Parse.JSSDK
		return writeLegacyConfigFile(
			lconf,
			filepath.Join(e.Root, LegacyConfigFile),
		)
	case ParseFormat, HerokuFormat:
		return writeConfigFile(c, filepath.Join(e.Root, ParseLocal))
	}
	return stackerr.Newf("Unknown project type: %d.", projectType)
}

func readProjectConfigFile(path string) (*ProjectConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, stackerr.Wrap(err)
	}
	defer f.Close()
	var p ProjectConfig
	if err := json.NewDecoder(f).Decode(&p); err != nil {
		return nil, stackerr.Newf("Config file %q is not valid JSON.", path)
	}
	return &p, err
}

func writeProjectConfig(p *ProjectConfig, path string) error {
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return stackerr.Wrap(err)
	}
	return stackerr.Wrap(ioutil.WriteFile(path, b, 0600))
}

func StoreProjectConfig(e *Env, c Config) error {
	p := c.GetProjectConfig()
	switch p.Type {
	case LegacyParseFormat:
		p, ok := (c).(*ParseConfig)
		if !ok {
			return stackerr.New("Invalid Cloud Code config.")
		}
		lconf := &legacyConfig{Applications: p.Applications}
		lconf.Global.ParseVersion = p.ProjectConfig.Parse.JSSDK
		lconf.Global.ParserEmail = p.ProjectConfig.ParserEmail
		return writeLegacyConfigFile(
			lconf,
			filepath.Join(e.Root, LegacyConfigFile),
		)

	case ParseFormat, HerokuFormat:
		return writeProjectConfig(p, filepath.Join(e.Root, ParseProject))
	}

	return stackerr.Newf("Unknown project type: %d.", p.Type)
}

func IsProjectDir(cur string) bool {
	_, err := os.Stat(filepath.Join(cur, ParseProject))
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(cur, ParseLocal))
	if err != nil {
		return false
	}
	return true
}

func GetProjectRoot(e *Env, cur string) string {
	if IsProjectDir(cur) {
		return cur
	}

	root := cur
	base := filepath.Base(root)

	for base != "." && base != string(filepath.Separator) {
		base = filepath.Base(root)
		root = filepath.Dir(root)
		if IsProjectDir(root) {
			return root
		}
	}

	return cur
}

func PrintDefault(e *Env, defaultApp string) error {
	if defaultApp == "" {
		return stackerr.New("No app is set as default app")
	}
	fmt.Fprintf(e.Out, "Current default app is %s\n", defaultApp)
	return nil
}

func SetDefault(e *Env, newDefault, defaultApp string, c Config) error {
	projectType := c.GetProjectConfig().Type
	switch projectType {
	case LegacyParseFormat, ParseFormat:
		p, ok := c.(*ParseConfig)
		if !ok {
			return stackerr.New("Invalid Cloud Code config.")
		}
		return setParseDefault(e, newDefault, defaultApp, p)

	case HerokuFormat:
		h, ok := c.(*HerokuConfig)
		if !ok {
			return stackerr.New("Invalid Cloud Code config.")
		}
		return SetHerokuDefault(e, newDefault, defaultApp, h)
	}

	return stackerr.Newf("Unknown project type: %d.", projectType)
}

func SetParserEmail(e *Env, email string) error {
	config, err := ConfigFromDir(e.Root)
	if err != nil {
		return err
	}
	config.GetProjectConfig().ParserEmail = email
	err = StoreProjectConfig(e, config)
	if err != nil {
		fmt.Fprintln(e.Err, "Could not set parser email for project.")
		return err
	}
	fmt.Fprintf(e.Out, "Successfully configured email for current project to: %q\n", email)
	return nil
}
