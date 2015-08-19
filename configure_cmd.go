package main

import (
	"fmt"

	"github.com/facebookgo/stackerr"
	"github.com/spf13/cobra"
)

type configureCmd struct {
	login login
}

func (c *configureCmd) accessToken(e *env) error {
	fmt.Fprintf(e.Out,
		`Please enter an access token if you already generated it.

If you do not have an access token or would like to generate a new one,
please type: "y" to open the browser or "n" to continue: `,
	)

	c.login.helpCreateToken(e)

	var credentials credentials
	fmt.Fprintf(e.Out, "Access Token: ")
	fmt.Fscanf(e.In, "%s\n", &credentials.token)

	_, err := (&apps{login: login{credentials: credentials}}).restFetchApps(e)
	if err != nil {
		if err == errAuth {
			fmt.Fprintf(e.Err,
				`Sorry, the access token you provided is not valid.
Please follow instructions at %s to generate a new access token.
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
		Short: "Configure various Parse settings.",
		Long:  "Configure various Parse settings like access tokens, project type, and more.",
		Run: func(c *cobra.Command, args []string) {
			c.Help()
		},
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "token",
		Short: "Store Parse access token on machine",
		Long:  "Stores Parse access token in ~/.parse/netrc.",
		Run:   runNoArgs(e, c.accessToken),
	})
	return cmd
}
