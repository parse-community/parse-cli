package main

import (
	"github.com/ParsePlatform/parse-cli/parsecli"
	"github.com/ParsePlatform/parse-cli/parsecmd"
	"github.com/facebookgo/stackerr"
	"github.com/spf13/cobra"
)

type addCmd struct {
	MakeDefault bool
	apps        *parsecli.Apps
	verbose     bool
}

func (a *addCmd) addSelectedApp(
	name string,
	appConfig parsecli.AppConfig,
	args []string,
	e *parsecli.Env,
) error {
	switch e.Type {
	case parsecli.LegacyParseFormat, parsecli.ParseFormat:
		parseAppConfig, ok := appConfig.(*parsecli.ParseAppConfig)
		if !ok {
			return stackerr.New("invalid parse app config passed.")
		}
		return parsecmd.AddSelectedParseApp(name, parseAppConfig, args, a.MakeDefault, a.verbose, e)
	}

	return stackerr.Newf("Unknown project type: %d.", e.Type)
}

func (a *addCmd) selectApp(e *parsecli.Env, appName string) (*parsecli.App, error) {
	apps, err := a.apps.RestFetchApps(e)
	if err != nil {
		return nil, err
	}
	if appName != "" {
		for _, app := range apps {
			if app.Name == appName {
				return app, nil
			}
		}
		return nil, stackerr.Newf("You are not a collaborator on app: %s", appName)
	}

	app, err := a.apps.SelectApp(apps, "Select an App to add to config: ", e)
	if err != nil {
		return nil, err
	}
	return app, nil
}

func (a *addCmd) run(e *parsecli.Env, args []string) error {

	if err := a.apps.Login.AuthUser(e); err != nil {
		return err
	}
	var appName string
	if len(args) > 1 {
		return stackerr.New("Only an optional Parse app name is expected.")
	}
	if len(args) == 1 {
		appName = args[0]
	}

	app, err := a.selectApp(e, appName)
	if err != nil {
		return err
	}
	appConfig := parsecmd.GetParseAppConfig(app)
	return a.addSelectedApp(app.Name, appConfig, args, e)
}

func NewAddCmd(e *parsecli.Env) *cobra.Command {
	a := &addCmd{
		MakeDefault: false,
		apps:        &parsecli.Apps{},
		verbose:     true,
	}
	cmd := &cobra.Command{
		Use:   "add [app]",
		Short: "Adds a new Parse App to config in current Cloud Code directory",
		Long: `Adds a new Parse App to config in current Cloud Code directory.
If an argument is given, the added application can also be referenced by that name.`,
		Run: parsecli.RunWithArgs(e, a.run),
	}
	cmd.Flags().BoolVarP(&a.MakeDefault, "default", "d", a.MakeDefault,
		"Make the selected app default")
	return cmd
}
