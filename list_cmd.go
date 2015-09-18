package main

import (
	"fmt"
	"os"

	"github.com/facebookgo/stackerr"
	"github.com/spf13/cobra"
)

const (
	applications = "applications"
)

type listCmd struct{}

func (l *listCmd) printListOfApps(e *env) error {
	config, err := configFromDir(e.Root)
	if err != nil {
		if os.IsNotExist(err) {
			// print nothing if we are not in a parse app directory
			return nil
		}
		return stackerr.Wrap(err)
	}
	config.pprintApps(e)
	fmt.Fprintln(e.Out, "")
	return nil
}

func (l *listCmd) run(e *env, args []string) error {
	l.printListOfApps(e)

	var apps apps
	if err := apps.login.authUser(e); err != nil {
		return err
	}
	var appName string
	if len(args) > 0 {
		appName = args[0]
	}
	return apps.showApps(e, appName)
}

func newListCmd(e *env) *cobra.Command {
	l := listCmd{}
	return &cobra.Command{
		Use:   "list",
		Short: "Lists Parse apps associated with current Parse account",
		Long: `Lists Parse apps and aliases added to the current Cloud Code directory
when executed inside the directory.
Additionally, it prints the list of Parse apps associated with current Parse account.`,
		Run: runWithArgs(e, l.run),
	}
}
