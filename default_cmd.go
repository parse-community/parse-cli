package main

import (
	"github.com/facebookgo/stackerr"
	"github.com/spf13/cobra"
)

type defaultCmd struct{}

func (d *defaultCmd) run(e *env, args []string) error {
	var newDefault string
	if len(args) > 1 {
		return stackerr.Newf("unexpected arguments, only an optional app name is expected: %v", args)
	}
	if len(args) == 1 {
		newDefault = args[0]
	}

	config, err := configFromDir(e.Root)
	if err != nil {
		return err
	}

	if config.getNumApps() == 0 {
		return stackerr.New("No apps are associated with this project. You can add some with parse add")
	}

	defaultApp := config.getDefaultApp()

	switch newDefault {
	case "":
		return printDefault(e, defaultApp)
	default:
		return setDefault(e, newDefault, defaultApp, config)
	}
}

func newDefaultCmd(e *env) *cobra.Command {
	d := defaultCmd{}
	return &cobra.Command{
		Use:   "default [app]",
		Short: "Sets or gets the default Parse App",
		Long: `Gets the default Parse App. If an argument is given, sets the ` +
			`default Parse App.`,
		Run: runWithArgs(e, d.run),
	}
}
