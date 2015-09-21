package main

import (
	"github.com/facebookgo/stackerr"
	"github.com/spf13/cobra"
)

type addCmd struct {
	MakeDefault bool
	apps        *apps
	verbose     bool
}

func (a *addCmd) addSelectedApp(
	name string,
	appConfig appConfig,
	args []string,
	e *env,
) error {
	switch e.Type {
	case legacyParseFormat, parseFormat:
		parseAppConfig, ok := appConfig.(*parseAppConfig)
		if !ok {
			return stackerr.New("invalid parse app config passed.")
		}
		return a.addSelectedParseApp(name, parseAppConfig, args, e)
	}

	return stackerr.Newf("Unknown project type: %d.", e.Type)
}

func (a *addCmd) selectApp(e *env, appName string) (*app, error) {
	apps, err := a.apps.restFetchApps(e)
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

	app, err := a.apps.selectApp(apps, "Select an App to add to config: ", e)
	if err != nil {
		return nil, err
	}
	return app, nil
}

func (a *addCmd) run(e *env, args []string) error {

	if err := a.apps.login.authUser(e); err != nil {
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
	appConfig := a.getParseAppConfig(app)
	return a.addSelectedApp(app.Name, appConfig, args, e)
}

func newAddCmd(e *env) *cobra.Command {
	a := &addCmd{
		MakeDefault: false,
		apps:        &apps{},
		verbose:     true,
	}
	cmd := &cobra.Command{
		Use:   "add [app]",
		Short: "Adds a new Parse App to config in current Cloud Code directory",
		Long: `Adds a new Parse App to config in current Cloud Code directory.
If an argument is given, the added application can also be referenced by that name.`,
		Run: runWithArgs(e, a.run),
	}
	cmd.Flags().BoolVarP(&a.MakeDefault, "default", "d", a.MakeDefault,
		"Make the selected app default")
	return cmd
}
