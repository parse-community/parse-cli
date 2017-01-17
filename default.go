package main

import (
	"github.com/ParsePlatform/parse-cli/parsecli"
	"github.com/facebookgo/stackerr"
	"github.com/spf13/cobra"
)

type defaultCmd struct{}

func (d *defaultCmd) run(e *parsecli.Env, args []string) error {
	var newDefault string
	if len(args) > 1 {
		return stackerr.Newf("unexpected arguments, only an optional app name is expected: %v", args)
	}
	if len(args) == 1 {
		newDefault = args[0]
	}

	config, err := parsecli.ConfigFromDir(e.Root)
	if err != nil {
		return err
	}

	if config.GetNumApps() == 0 {
		return stackerr.New("No apps are associated with this project. You can add some with back4app add")
	}

	defaultApp := config.GetDefaultApp()

	switch newDefault {
	case "":
		return parsecli.PrintDefault(e, defaultApp)
	default:
		return parsecli.SetDefault(e, newDefault, defaultApp, config)
	}
}

func NewDefaultCmd(e *parsecli.Env) *cobra.Command {
	d := defaultCmd{}
	return &cobra.Command{
		Use:   "default [app]",
		Short: "Sets or gets the default Parse App",
		Long: `Gets the default Parse App. If an argument is given, sets the ` +
			`default Parse App.`,
		Run: parsecli.RunWithArgs(e, d.run),
	}
}
