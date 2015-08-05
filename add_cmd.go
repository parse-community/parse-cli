package main

import (
	"os"
	"path/filepath"

	"github.com/facebookgo/stackerr"
	"github.com/spf13/cobra"
)

type addCmd struct {
	MakeDefault bool
	apps        *apps
}

func (a *addCmd) writeConfig(
	app *app,
	args []string,
	e *env,
	verbose bool,
) error {
	config, err := configFromDir(e.Root)
	if err != nil {
		return err
	}

	p := config.getProjectConfig()
	if p.Type == legacy {
		parseConfig, ok := config.(*parseConfig)
		if !ok {
			return stackerr.New("Invalid Cloud Code config.")
		}
		return a.writeParseConfig(parseConfig, app, args, e, verbose)
	}

	return stackerr.Newf("Unknown project type: %s.", p.Type)
}

func (a *addCmd) selectApp(e *env) (*app, error) {
	apps, err := a.apps.restFetchApps(e)
	if err != nil {
		return nil, err
	}
	app, err := a.apps.selectApp(apps, "Select an App to add to config: ", e)
	if err != nil {
		return nil, err
	}
	return app, nil
}

func (a *addCmd) run(e *env, args []string) error {
	_, err := os.Lstat(filepath.Join(e.Root, legacyConfigFile))
	if os.IsNotExist(err) {
		return stackerr.New("Please run add command inside a parse project.")
	}
	if err := a.apps.login.authUser(e); err != nil {
		return err
	}
	app, err := a.selectApp(e)
	if err != nil {
		return err
	}
	return a.writeConfig(app, args, e, true)
}

func newAddCmd(e *env) *cobra.Command {
	a := &addCmd{
		MakeDefault: false,
		apps:        &apps{},
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
