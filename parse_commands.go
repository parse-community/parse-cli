package main

import (
	"fmt"
	"os"
	"time"

	"github.com/ParsePlatform/parse-cli/parsecli"
	"github.com/ParsePlatform/parse-cli/parsecmd"
	"github.com/spf13/cobra"
)

func parseRootCmd(e *parsecli.Env) ([]string, *cobra.Command) {
	c := &cobra.Command{
		Use: "parse",
		Long: fmt.Sprintf(
			`Parse Command Line Interface
Version %v
Copyright %d Parse, Inc.
http://parse.com`,
			parsecli.Version,
			time.Now().Year(),
		),
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	c.AddCommand(NewAddCmd(e))
	c.AddCommand(NewConfigureCmd(e))
	c.AddCommand(NewDefaultCmd(e))
	c.AddCommand(parsecmd.NewDeployCmd(e))
	c.AddCommand(parsecmd.NewDevelopCmd(e))
	c.AddCommand(parsecmd.NewDownloadCmd(e))
	c.AddCommand(NewFunctionHooksCmd(e))
	c.AddCommand(parsecmd.NewGenerateCmd(e))
	c.AddCommand(parsecmd.NewJsSdkCmd(e))
	c.AddCommand(NewListCmd(e))
	c.AddCommand(parsecmd.NewLogsCmd(e))
	c.AddCommand(NewMigrateCmd(e))
	c.AddCommand(NewNewCmd(e))
	c.AddCommand(parsecmd.NewReleasesCmd(e))
	c.AddCommand(parsecmd.NewRollbackCmd(e))
	c.AddCommand(parsecmd.NewSymbolsCmd(e))
	c.AddCommand(NewTriggerHooksCmd(e))
	c.AddCommand(NewUpdateCmd(e))
	c.AddCommand(NewVersionCmd(e))

	if len(os.Args) <= 1 {
		return nil, c
	}

	commands := []string{"help"}
	for _, command := range c.Commands() {
		commands = append(commands, command.Name())
	}

	args := make([]string, len(os.Args)-1)
	copy(args, os.Args[1:])

	if message := parsecli.MakeCorrections(commands, args); message != "" {
		fmt.Fprintln(e.Out, message)
	}
	c.SetArgs(args)

	return args, c
}
