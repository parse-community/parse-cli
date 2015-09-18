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
		Use:   "list [app]",
		Short: "Lists properties of given Parse app and Parse apps associated with given project",
		Long: `Lists Parse apps and aliases added to the current Cloud Code directory
when executed inside the directory.
Additionally, it prints the list of Parse apps associated with current Parse account.
If an optional app name is provided, it prints all the keys for that app.`,
		Run: runWithArgs(e, l.run),
	}
}
