package main

import (
	"fmt"

	"github.com/facebookgo/stackerr"
)

func (a *addCmd) writeParseConfig(
	parseConfig *parseConfig,
	app *app,
	args []string,
	e *env,
	verbose bool,
) error {
	// add app to config
	if _, ok := parseConfig.Applications[app.Name]; ok {
		return stackerr.Newf("App %s has already been added", app.Name)
	}
	parseConfig.Applications[app.Name] = &parseAppConfig{
		ApplicationID: app.ApplicationID,
		MasterKey:     app.MasterKey,
	}

	if len(args) > 0 && args[0] != "" {
		alias := args[0]
		aliasConfig, ok := parseConfig.Applications[alias]
		if !ok {
			parseConfig.Applications[alias] = &parseAppConfig{Link: app.Name}
		}
		if ok && aliasConfig.getLink() != "" {
			fmt.Fprintf(e.Out, "Overwriting alias: %q to point to %q\n", alias, app.Name)
			parseConfig.Applications[alias] = &parseAppConfig{Link: app.Name}
		}
	}

	if a.MakeDefault {
		if _, ok := parseConfig.Applications[defaultKey]; ok {
			return stackerr.New(`Default key already set. To override default, use command "parse default"`)
		}
		parseConfig.Applications[defaultKey] = &parseAppConfig{Link: app.Name}
	}

	if err := storeParseConfig(e, parseConfig); err != nil {
		return err
	}
	if verbose {
		fmt.Fprintf(e.Out, "Written config for %q\n", app.Name)
		if a.MakeDefault {
			fmt.Fprintf(e.Out, "Set %q as default\n", app.Name)
		}
	}

	return nil
}
