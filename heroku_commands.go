package main

import (
	"fmt"
	"os"
	"time"

	"github.com/ParsePlatform/parse-cli/herokucmd"
	"github.com/ParsePlatform/parse-cli/parsecli"
	"github.com/ParsePlatform/parse-cli/webhooks"
	"github.com/spf13/cobra"
)

func herokuRootCmd(e *parsecli.Env) ([]string, *cobra.Command) {
	c := &cobra.Command{
		Use: "parse",
		Long: fmt.Sprintf(
			`Parse Command Line Tool

Tools to help you set up server code on Heroku
Version %v
Copyright %d Parse, Inc.
https://www.parse.com`,
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
	c.AddCommand(herokucmd.NewDownloadCmd(e))
	c.AddCommand(herokucmd.NewDeployCmd(e))
	c.AddCommand(webhooks.NewFunctionHooksCmd(e))
	c.AddCommand(NewListCmd(e))
	c.AddCommand(herokucmd.NewLogsCmd(e))
	c.AddCommand(NewNewCmd(e))
	c.AddCommand(herokucmd.NewReleasesCmd(e))
	c.AddCommand(herokucmd.NewRollbackCmd(e))
	c.AddCommand(webhooks.NewTriggerHooksCmd(e))
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
