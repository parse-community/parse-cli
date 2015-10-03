package main

import (
	"fmt"
	"os"

	"github.com/ParsePlatform/parse-cli/parsecli"
	"github.com/facebookgo/stackerr"
	"github.com/spf13/cobra"
)

const (
	applications = "applications"
)

type listCmd struct{}

func (l *listCmd) printListOfApps(e *parsecli.Env) error {
	config, err := parsecli.ConfigFromDir(e.Root)
	if err != nil {
		if os.IsNotExist(err) {
			// print nothing if we are not in a parse app directory
			return nil
		}
		return stackerr.Wrap(err)
	}
	config.PrettyPrintApps(e)
	fmt.Fprintln(e.Out, "")
	return nil
}

func (l *listCmd) run(e *parsecli.Env, args []string) error {
	l.printListOfApps(e)

	var apps parsecli.Apps
	if err := apps.Login.AuthUser(e); err != nil {
		return err
	}
	var appName string
	if len(args) > 0 {
		appName = args[0]
	}
	return apps.ShowApps(e, appName)
}

func NewListCmd(e *parsecli.Env) *cobra.Command {
	l := listCmd{}
	return &cobra.Command{
		Use:   "list [app]",
		Short: "Lists properties of given Parse app and Parse apps associated with given project",
		Long: `Lists Parse apps and aliases added to the current Cloud Code directory
when executed inside the directory.
Additionally, it prints the list of Parse apps associated with current Parse account.
If an optional app name is provided, it prints all the keys for that app.`,
		Run: parsecli.RunWithArgs(e, l.run),
	}
}
