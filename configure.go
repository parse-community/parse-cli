package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"

	"github.com/ParsePlatform/parse-cli/parsecli"
	"github.com/ParsePlatform/parse-cli/webhooks"
	"github.com/facebookgo/stackerr"
	"github.com/spf13/cobra"
)

type configureCmd struct {
	login       parsecli.Login
	isDefault   bool
	tokenReader io.Reader // for testing

	hooks webhooks.Hooks
}

func (c *configureCmd) accountKey(e *parsecli.Env) error {
	token, err := c.login.HelpCreateToken(e)
	if err != nil {
		return err
	}

	email, err := c.login.AuthToken(e, token)
	if err != nil {
		fmt.Fprintln(e.Err, "Could not store credentials. Please try again.\n")
		return err
	}

	if c.isDefault {
		email = ""
	}

	var l parsecli.Login
	if c.tokenReader != nil {
		l.TokenReader = c.tokenReader
	}
	foundEmail, creds, err := l.GetTokenCredentials(e, email)
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
				parsecli.Last4(token),
				email,
			)
		}
	}

	err = c.login.StoreCredentials(e, email, &parsecli.Credentials{Token: token})
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
Otherwise, you'll have to explicitly set the PARSER_EMAIL common.Environment variable
for them to pick the correct account key.
Further, if the command line tool cannot find an account key for a configured email it will try to
use the default account key.
Hence, we are automatically configuring the default account key to be the same as current account key.
`,
		)
		err = c.login.StoreCredentials(e, "", &parsecli.Credentials{Token: token})
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

func (c *configureCmd) parserEmail(e *parsecli.Env, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("Invalid args: %v, only an email argument is expected.", args)
	}
	return parsecli.SetParserEmail(e, args[0])
}

func (c *configureCmd) projectType(e *parsecli.Env, args []string) error {
	config, err := parsecli.ConfigFromDir(e.Root)
	if err != nil {
		return err
	}
	if len(args) > 1 {
		return stackerr.Newf("Invalid args: %v, only an optional project type argument is expected.", args)
	}
	validTypes := map[string]int{"parse": parsecli.ParseFormat}
	invertedTypes := map[int]string{parsecli.ParseFormat: "parse"}
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

	config.GetProjectConfig().Type = selectedProjectType
	if err := parsecli.StoreProjectConfig(e, config); err != nil {
		fmt.Fprintln(e.Err, "Could not save selected project type to project config")
		return err
	}
	fmt.Fprintf(e.Out, "Successfully set project type to: %v\n", invertedTypes[selectedProjectType])
	return nil
}

func NewConfigureCmd(e *parsecli.Env) *cobra.Command {
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
		Run:     parsecli.RunNoArgs(e, c.accountKey),
		Aliases: []string{"key"},
	}
	keyCmd.Flags().BoolVarP(&c.isDefault, "default", "d", c.isDefault,
		"Make this token a system default")
	cmd.AddCommand(keyCmd)

	emailCmd := &cobra.Command{
		Use:     "email [id]",
		Short:   "Configures the parser email for this project",
		Long:    "Configures the parser email for current project.",
		Run:     parsecli.RunWithArgs(e, c.parserEmail),
		Aliases: []string{"user"},
	}
	cmd.AddCommand(emailCmd)

	projectCmd := &cobra.Command{
		Use:   "project [type]",
		Short: "Set the project type to one among listed options",
		Long:  "Set the project type to one among listed options. For instance, 'parse'",
		Run:   parsecli.RunWithArgs(e, c.projectType),
	}
	cmd.AddCommand(projectCmd)

	hooksCmd := &cobra.Command{
		Use:   "hooks [filename]",
		Short: "Configure webhooks according to given config file",
		Long: `Configure webhooks for the app based on the given configuration csv file.
For more details read: https://parse.com/docs/js/guide#command-line-webhooks
`,
		Run:     parsecli.RunWithArgsClient(e, c.hooks.HooksCmd),
		Aliases: []string{"webhooks"},
	}
	hooksCmd.Flags().BoolVarP(&c.hooks.HooksStrict, "strict", "s", c.hooks.HooksStrict,
		"Configure hooks in strict mode, i.e., do not automatically fix errors.")
	hooksCmd.Flags().StringVarP(&c.hooks.BaseURL, "base", "b", c.hooks.BaseURL,
		`Base url to use while parsing the webhook url field.
If provided, the config file can have relative urls.`)
	cmd.AddCommand(hooksCmd)

	return cmd
}
