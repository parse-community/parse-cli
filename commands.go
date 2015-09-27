package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

func parseRootCmd(e *env) ([]string, *cobra.Command) {
	c := &cobra.Command{
		Use: "parse",
		Long: fmt.Sprintf(
			`Parse Command Line Interface
Version %v
Copyright %d Parse, Inc.
http://parse.com`,
			version,
			time.Now().Year(),
		),
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	c.AddCommand(newAddCmd(e))
	c.AddCommand(newConfigureCmd(e))
	c.AddCommand(newDefaultCmd(e))
	c.AddCommand(newDeployCmd(e))
	c.AddCommand(newDevelopCmd(e))
	c.AddCommand(newDownloadCmd(e))
	c.AddCommand(newFunctionHooksCmd(e))
	c.AddCommand(newGenerateCmd(e))
	c.AddCommand(newJsSdkCmd(e))
	c.AddCommand(newListCmd(e))
	c.AddCommand(newLogsCmd(e))
	c.AddCommand(newMigrateCmd(e))
	c.AddCommand(newNewCmd(e))
	c.AddCommand(newReleasesCmd(e))
	c.AddCommand(newRollbackCmd(e))
	c.AddCommand(newSymbolsCmd(e))
	c.AddCommand(newTriggerHooksCmd(e))
	c.AddCommand(newUpdateCmd(e))
	c.AddCommand(newVersionCmd(e))

	if len(os.Args) <= 1 {
		return nil, c
	}

	commands := []string{"help"}
	for _, command := range c.Commands() {
		commands = append(commands, command.Name())
	}

	args := make([]string, len(os.Args)-1)
	copy(args, os.Args[1:])

	if message := makeCorrections(commands, args); message != "" {
		fmt.Fprintln(e.Out, message)
	}
	c.SetArgs(args)

	return args, c
}
