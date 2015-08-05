package main

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/facebookgo/stackerr"
)

type parseProjectConfig struct {
	JSSDK string `json:"jssdk,omitempty"`
}

type parseAppConfig struct {
	ApplicationID string `json:"applicationId,omitempty"`
	MasterKey     string `json:"masterKey,omitempty"`
	Link          string `json:"link,omitempty"`

	masterKey string
}

func (c *parseAppConfig) getApplicationID() string {
	return c.ApplicationID
}

func (c *parseAppConfig) getMasterKey(e *env) (string, error) {
	if c.MasterKey != "" {
		return c.MasterKey, nil
	}
	if c.masterKey != "" {
		return c.masterKey, nil
	}
	app, err := fetchAppKeys(e, c.getApplicationID())
	if err != nil {
		return "", err
	}
	c.masterKey = app.MasterKey
	return app.MasterKey, nil
}

func (c *parseAppConfig) getApplicationAuth(e *env) (string, error) {
	return c.getMasterKey(e)
}

func (c *parseAppConfig) getLink() string {
	return c.Link
}

type parseConfig struct {
	Applications  map[string]*parseAppConfig `json:"applications,omitempty"`
	projectConfig *projectConfig
}

func (c *parseConfig) addAlias(name, link string) error {
	if _, found := c.Applications[name]; found {
		return stackerr.Newf("App %q has already been added.", name)
	}
	if _, found := c.Applications[link]; !found {
		return stackerr.Newf("App %q wasn't found.", link)
	}
	c.Applications[name] = &parseAppConfig{Link: link}
	return nil
}

func (c *parseConfig) setDefaultApp(name string) error {
	delete(c.Applications, defaultKey)
	return c.addAlias(defaultKey, name)
}

func (c *parseConfig) app(name string) (appConfig, error) {
	ac, found := c.Applications[name]
	if !found {
		if name == defaultKey {
			return nil, stackerr.Newf("No default app configured.")
		}
		return nil, stackerr.Newf("App %q wasn't found.", name)
	}
	if ac.Link != "" {
		return c.app(ac.Link)
	}
	return ac, nil
}

func (c *parseConfig) getProjectConfig() *projectConfig {
	return c.projectConfig
}

func (c *parseConfig) getDefaultApp() string {
	var defaultApp string
	if defaultKeyLink, ok := c.Applications[defaultKey]; ok {
		defaultApp = defaultKeyLink.Link
	}
	return defaultApp
}

func (c *parseConfig) getNumApps() int {
	return len(c.Applications)
}

func (c *parseConfig) pprintApps(e *env) {
	apps := c.Applications

	defaultApp := c.getDefaultApp()

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
		"The following apps are associated with cloud code in the current directory:",
	)

	for _, appName := range appNames {
		if appName == defaultKey {
			continue
		}
		if defaultApp == appName {
			fmt.Fprint(e.Out, "* ")
		} else {
			fmt.Fprint(e.Out, "  ")
		}
		fmt.Fprintf(e.Out, "%s", appName)

		if config, _ := apps[appName]; config.getLink() != "" {
			fmt.Fprintf(e.Out, " -> %s", config.getLink())
		}
		fmt.Fprintln(e.Out, "")
	}
}

func storeParseConfig(e *env, c *parseConfig) error {
	projectType := c.projectConfig.Type
	if projectType == legacy {
		lconf := &legacyConfig{Applications: c.Applications}
		lconf.Global.ParseVersion = c.projectConfig.Parse.JSSDK
		return writeLegacyConfigFile(
			lconf,
			filepath.Join(e.Root, legacyConfigFile),
		)
	}
	return stackerr.Newf("Unknown project type: %s.", projectType)
}
