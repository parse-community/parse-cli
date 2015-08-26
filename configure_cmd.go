package main

import (
	"fmt"

	"github.com/facebookgo/stackerr"
	"github.com/spf13/cobra"
)

type configureCmd struct {
	login     login
	isDefault bool
}

func (c *configureCmd) accountKey(e *env) error {
	token, err := c.login.helpCreateToken(e)
	if err != nil {
		return err
	}

	email, err := c.login.authToken(e, token)
	if err != nil {
		fmt.Fprintln(e.Err, "Could not store credentials. Please try again.\n")
		return err
	}

	if c.isDefault {
		email = ""
	}

	creds, _ := (&login{}).getTokenCredentials(e, email)
	if creds != nil {
		if c.isDefault {
			fmt.Fprintln(
				e.Err,
				"Note: this operation will overwrite the default account key",
			)
		} else {
			fmt.Fprintf(
				e.Err,
				`Note: this operation will overwrite the account key:
 %q
for email: %q
`,
				last4(token),
				email,
			)
		}
	}

	err = c.login.storeCredentials(e, email, &credentials{token: token})
	if err == nil {
		fmt.Fprintln(e.Out, "Successfully stored credentials.")
	}
	return stackerr.Wrap(err)
}

func (c *configureCmd) parserEmail(e *env, args []string) error {
	config, err := configFromDir(e.Root)
	if err != nil {
		return err
	}
	if len(args) != 1 {
		return fmt.Errorf("Invalid args: %v, only an email argument is expected.", args)
	}
	config.getProjectConfig().ParserEmail = args[0]
	err = storeProjectConfig(e, config)
	if err != nil {
		fmt.Fprintln(e.Err, "Could not set parser email for project.")
		return err
	}
	fmt.Fprintf(e.Out, "Successfully configured email for current project to: %q\n", args[0])
	return nil
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

	keyCmd := &cobra.Command{
		Use:     "accountkey",
		Short:   "Store Parse account key on machine",
		Long:    "Stores Parse account key in ~/.parse/netrc.",
		Run:     runNoArgs(e, c.accountKey),
		Aliases: []string{"key"},
	}
	keyCmd.Flags().BoolVarP(&c.isDefault, "default", "d", c.isDefault,
		"Make this token a system default")
	cmd.AddCommand(keyCmd)

	emailCmd := &cobra.Command{
		Use:   "email",
		Short: "Configures the parser email for this project",
		Long:  "Configures the parser email for current project.",
		Run:   runWithArgs(e, c.parserEmail),
	}
	cmd.AddCommand(emailCmd)

	return cmd
}
