package parsecmd

import (
	"fmt"

	"github.com/ParsePlatform/parse-cli/parsecli"
	"github.com/facebookgo/stackerr"
)

func GetParseAppConfig(app *parsecli.App) *parsecli.ParseAppConfig {
	pc := &parsecli.ParseAppConfig{
		ApplicationID: app.ApplicationID,
	}
	return pc.WithInternalMasterKey(app.MasterKey)
}

func AddSelectedParseApp(
	appName string,
	appConfig *parsecli.ParseAppConfig,
	args []string,
	makeDefault, verbose bool,
	e *parsecli.Env,
) error {
	config, err := parsecli.ConfigFromDir(e.Root)
	if err != nil {
		return err
	}
	parseConfig, ok := config.(*parsecli.ParseConfig)
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
			parseConfig.Applications[alias] = &parsecli.ParseAppConfig{Link: appName}
		}
		if ok && aliasConfig.GetLink() != "" {
			fmt.Fprintf(e.Out, "Overwriting alias: %q to point to %q\n", alias, appName)
			parseConfig.Applications[alias] = &parsecli.ParseAppConfig{Link: appName}
		}
	}

	if makeDefault {
		if _, ok := parseConfig.Applications[parsecli.DefaultKey]; ok {
			return stackerr.New(`Default key already set. To override default, use command "parse default"`)
		}
		parseConfig.Applications[parsecli.DefaultKey] = &parsecli.ParseAppConfig{Link: appName}
	}

	if err := parsecli.StoreConfig(e, parseConfig); err != nil {
		return err
	}
	if verbose {
		fmt.Fprintf(e.Out, "Written config for %q\n", appName)
		if makeDefault {
			fmt.Fprintf(e.Out, "Set %q as default\n", appName)
		}
	}

	return nil
}
