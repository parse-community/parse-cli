package main

import (
	"fmt"

	"github.com/facebookgo/stackerr"
)

func (a *addCmd) getParseAppConfig(app *app) *parseAppConfig {
	return &parseAppConfig{
		ApplicationID: app.ApplicationID,
		masterKey:     app.MasterKey,
	}
}

func (a *addCmd) addSelectedParseApp(
	appName string,
	appConfig *parseAppConfig,
	args []string,
	e *env,
) error {
	config, err := configFromDir(e.Root)
	if err != nil {
		return err
	}
	parseConfig, ok := config.(*parseConfig)
	if !ok {
		return stackerr.New("Invalid Cloud Code config.")
	}

	// add app to config
	if _, ok := parseConfig.Applications[appName]; ok {
		return stackerr.Newf("App %s has already been added", appName)
	}

	parseConfig.Applications[appName] = appConfig

	if len(args) > 0 && args[0] != "" {
		alias := args[0]
		aliasConfig, ok := parseConfig.Applications[alias]
		if !ok {
			parseConfig.Applications[alias] = &parseAppConfig{Link: appName}
		}
		if ok && aliasConfig.getLink() != "" {
			fmt.Fprintf(e.Out, "Overwriting alias: %q to point to %q\n", alias, appName)
			parseConfig.Applications[alias] = &parseAppConfig{Link: appName}
		}
	}

	if a.MakeDefault {
		if _, ok := parseConfig.Applications[defaultKey]; ok {
			return stackerr.New(`Default key already set. To override default, use command "parse default"`)
		}
		parseConfig.Applications[defaultKey] = &parseAppConfig{Link: appName}
	}

	if err := storeConfig(e, parseConfig); err != nil {
		return err
	}
	if a.verbose {
		fmt.Fprintf(e.Out, "Written config for %q\n", appName)
		if a.MakeDefault {
			fmt.Fprintf(e.Out, "Set %q as default\n", appName)
		}
	}

	return nil
}
