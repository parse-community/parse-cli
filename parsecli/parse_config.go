package parsecli

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/facebookgo/stackerr"
)

type ParseProjectConfig struct {
	JSSDK string `json:"jssdk,omitempty"`
}

type ParseAppConfig struct {
	ApplicationID string `json:"applicationId,omitempty"`
	MasterKey     string `json:"masterKey,omitempty"`
	Link          string `json:"link,omitempty"`

	masterKey string
}

func (c *ParseAppConfig) WithInternalMasterKey(masterKey string) *ParseAppConfig {
	c.masterKey = masterKey
	return c
}

func (c *ParseAppConfig) GetApplicationID() string {
	return c.ApplicationID
}

func (c *ParseAppConfig) GetMasterKey(e *Env) (string, error) {
	if c.MasterKey != "" {
		return c.MasterKey, nil
	}
	if c.masterKey != "" {
		return c.masterKey, nil
	}
	app, err := FetchAppKeys(e, c.GetApplicationID())
	if err != nil {
		return "", err
	}
	c.masterKey = app.MasterKey
	return app.MasterKey, nil
}

func (c *ParseAppConfig) GetApplicationAuth(e *Env) (string, error) {
	return c.GetMasterKey(e)
}

func (c *ParseAppConfig) GetLink() string {
	return c.Link
}

type ParseConfig struct {
	Applications  map[string]*ParseAppConfig `json:"applications,omitempty"`
	ProjectConfig *ProjectConfig             `json:"-"`
}

func (c *ParseConfig) AddAlias(name, link string) error {
	if _, found := c.Applications[name]; found {
		return stackerr.Newf("App %q has already been added.", name)
	}
	if _, found := c.Applications[link]; !found {
		return stackerr.Newf("App %q wasn't found.", link)
	}
	c.Applications[name] = &ParseAppConfig{Link: link}
	return nil
}

func (c *ParseConfig) SetDefaultApp(name string) error {
	delete(c.Applications, DefaultKey)
	return c.AddAlias(DefaultKey, name)
}

func (c *ParseConfig) App(name string) (AppConfig, error) {
	ac, found := c.Applications[name]
	if !found {
		if name == DefaultKey {
			return nil, stackerr.Newf("No default app configured.")
		}
		return nil, stackerr.Newf("App %q wasn't found.", name)
	}
	if ac.Link != "" {
		return c.App(ac.Link)
	}
	return ac, nil
}

func (c *ParseConfig) GetProjectConfig() *ProjectConfig {
	return c.ProjectConfig
}

func (c *ParseConfig) GetDefaultApp() string {
	var defaultApp string
	if DefaultKeyLink, ok := c.Applications[DefaultKey]; ok {
		defaultApp = DefaultKeyLink.Link
	}
	return defaultApp
}

func (c *ParseConfig) GetNumApps() int {
	return len(c.Applications)
}

func (c *ParseConfig) PrettyPrintApps(e *Env) {
	apps := c.Applications

	defaultApp := c.GetDefaultApp()

	var appNames []string
	for appName := range apps {
		appNames = append(appNames, appName)
	}
	sort.Strings(appNames)

	if len(appNames) == 0 {
		return
	}

	fmt.Fprintln(
		e.Out,
		"The following apps are associated with Cloud Code in the current directory:",
	)

	for _, appName := range appNames {
		if appName == DefaultKey {
			continue
		}
		if defaultApp == appName {
			fmt.Fprint(e.Out, "* ")
		} else {
			fmt.Fprint(e.Out, "  ")
		}
		fmt.Fprintf(e.Out, "%s", appName)

		if config, _ := apps[appName]; config.GetLink() != "" {
			fmt.Fprintf(e.Out, " -> %s", config.GetLink())
		}
		fmt.Fprintln(e.Out, "")
	}
}

func readParseConfigFile(path string) (*ParseConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, stackerr.Wrap(err)
	}
	defer f.Close()
	var c ParseConfig
	if err := json.NewDecoder(f).Decode(&c); err != nil {
		return nil, stackerr.Newf("Config file %q is not valid JSON.", path)
	}
	return &c, nil
}

func setParseDefault(e *Env, newDefault, defaultApp string, parseConfig *ParseConfig) error {
	apps := parseConfig.Applications
	if _, ok := apps[newDefault]; !ok {
		parseConfig.PrettyPrintApps(e)
		return stackerr.Newf("Invalid application name \"%s\". Please select from the valid applications printed above.", newDefault)
	}

	if defaultApp == "" {
		apps[DefaultKey] = &ParseAppConfig{Link: newDefault}
	} else {
		apps[DefaultKey].Link = newDefault
	}
	if err := StoreConfig(e, parseConfig); err != nil {
		return err
	}
	fmt.Fprintf(e.Out, "Default app set to %s.\n", newDefault)
	return nil
}
