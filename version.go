package main

import (
	"fmt"

	"github.com/ParsePlatform/parse-cli/parsecli"
	"github.com/spf13/cobra"
)

type versionCmd struct{}

func (c *versionCmd) run(e *parsecli.Env) error {
	fmt.Fprintln(e.Out, parsecli.Version)
	return nil
}

func NewVersionCmd(e *parsecli.Env) *cobra.Command {
	var c versionCmd
	cmd := &cobra.Command{
		Use:     "version",
		Short:   "Gets the Command Line Tools version",
		Long:    `Gets the Command Line Tools version.`,
		Run:     parsecli.RunNoArgs(e, c.run),
		Aliases: []string{"cliversion"},
	}
	return cmd
}
