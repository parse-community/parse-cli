package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"

	"github.com/facebookgo/stackerr"
	"github.com/spf13/cobra"
)

type configureCmd struct {
	login       login
	isDefault   bool
	tokenReader io.Reader // for testing
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

	var l login
	if c.tokenReader != nil {
		l.tokenReader = c.tokenReader
	}
	foundEmail, creds, err := l.getTokenCredentials(e, email)
	firstEverConfigure := false
	if stackerr.HasUnderlying(err, stackerr.MatcherFunc(os.IsNotExist)) && !c.isDefault {
		firstEverConfigure = true
	}

	if creds != nil {
		if c.isDefault {
			fmt.Fprintln(
				e.Err,
				"Note: this operation will overwrite the default account key",
			)
		} else if foundEmail {
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
		if c.isDefault {
			fmt.Fprintln(e.Out, "Successfully stored default account key.")
		} else {
			fmt.Fprintf(e.Out, "Successfully stored account key for: %q.\n", email)
		}
	}
	if err != nil {
		fmt.Fprintln(e.Err, "Could not save account key.")
		return stackerr.Wrap(err)
	}

	if firstEverConfigure {
		fmt.Fprintln(
			e.Out,
			`
Looks like this is the first time you have configured an account key.
Note that "parse new" and "parse list" can automatically pick up a default key if present.
Otherwise, you'll have to explicitly set the PARSER_EMAIL environment variable
for them to pick the correct account key.
Further, if the command line tool cannot find an account key for a configured email it will try to
use the default account key.
Hence, we are automatically configuring the default account key to be the same as current account key.
`,
		)
		err = c.login.storeCredentials(e, "", &credentials{token: token})
		if err != nil {
			fmt.Fprintln(e.Err, "Could not save account key.")
			return stackerr.Wrap(err)
		}
		fmt.Fprintln(e.Out, `Successfully configured the default account key.
To change the default account key in future use:

       "parse configure accountkey -d"
`)
	}

	return nil
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

func (c *configureCmd) projectType(e *env, args []string) error {
	config, err := configFromDir(e.Root)
	if err != nil {
		return err
	}
	if len(args) > 1 {
		return stackerr.Newf("Invalid args: %v, only an optional project type argument is expected.", args)
	}
	validTypes := map[string]int{"parse": parseFormat}
	invertedTypes := map[int]string{parseFormat: "parse"}
	numKeys := len(validTypes)
	var validKeys []string
	for key := range validTypes {
		validKeys = append(validKeys, key)
	}
	sort.Strings(validKeys)
	var selectionString string
	for n, key := range validKeys {
		selectionString += fmt.Sprintf("%d: %s\n", 1+n, key)
	}

	selectedProjectType := -1
	if len(args) != 0 {
		projectType, ok := validTypes[args[0]]
		if !ok {
			return stackerr.Newf("Invalid projectType: %v, valid types are: \n%s", selectionString)
		}
		selectedProjectType = projectType
	}

	for i := 0; i < 3; i++ {
		fmt.Fprintf(e.Out, `Select from the listed project types:
%s
Enter a number between 1 and %d: `,
			selectionString,
			numKeys,
		)
		var selection string
		fmt.Fscanf(e.In, "%s\n", &selection)
		num, err := strconv.Atoi(selection)
		if err != nil || num < 1 || num > numKeys {
			fmt.Fprintf(e.Err, "Invalid selection. Please enter a number between 1 and %d\n", numKeys)
			continue
		}
		projectType, ok := validTypes[validKeys[num-1]]
		if !ok {
			return stackerr.Newf("Invalid projectType: %v, valid types are: \n%s", selectionString)
		}
		selectedProjectType = projectType
		break
	}
	if selectedProjectType == -1 {
		return stackerr.Newf("Could not make a selection. Please try again.")
	}

	config.getProjectConfig().Type = selectedProjectType
	if err := storeProjectConfig(e, config); err != nil {
		fmt.Fprintln(e.Err, "Could not save selected project type to project config")
		return err
	}
	fmt.Fprintf(e.Out, "Successfully set project type to: %v\n", invertedTypes[selectedProjectType])
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
		Use:     "email [id]",
		Short:   "Configures the parser email for this project",
		Long:    "Configures the parser email for current project.",
		Run:     runWithArgs(e, c.parserEmail),
		Aliases: []string{"user"},
	}
	cmd.AddCommand(emailCmd)

	projectCmd := &cobra.Command{
		Use:   "project [type]",
		Short: "Set the project type to one among listed options",
		Long:  "Set the project type to one among listed options. For instance, 'parse'",
		Run:   runWithArgs(e, c.projectType),
	}
	cmd.AddCommand(projectCmd)

	return cmd
}
