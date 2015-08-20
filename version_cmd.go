package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

type versionCmd struct{}

func (c *versionCmd) run(e *env) error {
	fmt.Fprintln(e.Out, version)
	return nil
}

func newVersionCmd(e *env) *cobra.Command {
	var c versionCmd
	cmd := &cobra.Command{
		Use:     "version",
		Short:   "Gets the Command Line Tools version",
		Long:    `Gets the Command Line Tools version.`,
		Run:     runNoArgs(e, c.run),
		Aliases: []string{"cliversion"},
	}
	return cmd
}
