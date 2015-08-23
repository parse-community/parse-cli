package main

import (
	"fmt"

	"github.com/facebookgo/stackerr"
	"github.com/spf13/cobra"
)

type configureCmd struct {
	login login
}

func (c *configureCmd) accountKey(e *env) error {
	token, err := c.login.helpCreateToken(e)
	if err != nil {
		return err
	}

	credentials := credentials{token: token}
	_, err = (&apps{login: login{credentials: credentials}}).restFetchApps(e)
	if err != nil {
		if err == errAuth {
			fmt.Fprintf(e.Err,
				`Sorry, the account key you provided is not valid.
Please follow instructions at %s to generate a new account key.
`,
				keysURL,
			)
		} else {
			fmt.Fprintf(e.Err, "Unable to validate token with error:\n%s\n", err)
		}
		return stackerr.New("Could not store credentials. Please try again.")
	}

	err = c.login.storeCredentials(e, &credentials)
	if err == nil {
		fmt.Fprintln(e.Out, "Successfully stored credentials.")
	}
	return stackerr.Wrap(err)
}

func newConfigureCmd(e *env) *cobra.Command {
	var c configureCmd

	cmd := &cobra.Command{
		Use:   "configure",
		Short: "Configure various Parse settings",
		Long:  "Configure various Parse settings like account keys, project type, and more.",
		Run: func(c *cobra.Command, args []string) {
			c.Help()
		},
	}
	cmd.AddCommand(&cobra.Command{
		Use:     "accountkey",
		Short:   "Store Parse account key on machine",
		Long:    "Stores Parse account key in ~/.parse/netrc.",
		Run:     runNoArgs(e, c.accountKey),
		Aliases: []string{"key"},
	})
	return cmd
}
