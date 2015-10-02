package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/facebookgo/stackerr"
	"github.com/spf13/cobra"
)

var (
	errInvalidFormat = errors.New(
		`
invalid format.
valid formats should look like:
[put|post],functionName,https_url
delete,functionName

[put|post],className:triggerName,https_url
delete,className:triggerName
`)

	errPostToPut = errors.New(
		`a hook with given name already exists: cannot create a new one.
`)

	errPutToPost = errors.New(
		`a hook with the given name does not exist yet: cannot update the url.
`)

	errNotExist = errors.New(
		`a hook with the given name does not exist. cannot delete it.
	`)
)

type configureCmd struct {
	login       login
	isDefault   bool
	tokenReader io.Reader // for testing

	// for hooks sub-command
	hooksStrict    bool
	baseURL        string
	baseWebhookURL *url.URL
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

type hookOperation struct {
	method   string
	function *functionHook
	trigger  *triggerHook
}

func (c *configureCmd) checkTriggerName(s string) error {
	switch strings.ToLower(s) {
	case "beforesave", "beforedelete", "aftersave", "afterdelete":
		return nil
	}
	return stackerr.Newf(
		`invalid trigger name: %v.
	This is the list of valid trigger names:
		beforeSave
		afterSave
		beforeDelete
		afterDelete
`,
		s,
	)
}

func (c *configureCmd) postOrPutHook(
	e *env,
	hooksOps []*hookOperation,
	fields ...string,
) (bool, []*hookOperation, error) {
	restOp := strings.ToUpper(fields[0])
	if restOp != "POST" && restOp != "PUT" {
		return false, nil, stackerr.Wrap(errInvalidFormat)
	}

	switch len(fields) {
	case 3:
		hooksOps = append(hooksOps, &hookOperation{
			method:   restOp,
			function: &functionHook{FunctionName: fields[1], URL: fields[2]},
		})
		return true, hooksOps, nil

	case 4:
		if err := c.checkTriggerName(fields[2]); err != nil {
			return false, nil, err
		}
		hooksOps = append(hooksOps, &hookOperation{
			method:  restOp,
			trigger: &triggerHook{ClassName: fields[1], TriggerName: fields[2], URL: fields[3]},
		})
		return true, hooksOps, nil
	}
	return false, nil, stackerr.Wrap(errInvalidFormat)

}

func (c *configureCmd) deleteHook(
	e *env,
	hooksOps []*hookOperation,
	fields ...string,
) (bool, []*hookOperation, error) {
	restOp := strings.ToUpper(fields[0])
	if restOp != "DELETE" {
		return false, nil, stackerr.Wrap(errInvalidFormat)
	}

	switch len(fields) {
	case 2:
		hooksOps = append(hooksOps, &hookOperation{
			method:   "DELETE",
			function: &functionHook{FunctionName: fields[1]},
		})
		return true, hooksOps, nil
	case 3:
		if err := c.checkTriggerName(fields[2]); err != nil {
			return false, nil, err
		}
		hooksOps = append(hooksOps, &hookOperation{
			method:  "DELETE",
			trigger: &triggerHook{ClassName: fields[1], TriggerName: fields[2]},
		})
		return true, hooksOps, nil
	}

	return false, nil, stackerr.Wrap(errInvalidFormat)
}

func (c *configureCmd) appendHookOperation(
	e *env,
	fields []string,
	hooks []*hookOperation,
) (bool, []*hookOperation, error) {
	if len(fields) == 0 {
		return false, hooks, nil
	}

	switch strings.ToLower(fields[0]) {
	case "post", "put":
		return c.postOrPutHook(e, hooks, fields...)

	case "delete":
		return c.deleteHook(e, hooks, fields...)
	}
	return false, nil, stackerr.Wrap(errInvalidFormat)
}

func (c *configureCmd) processHooksOperation(e *env, op string) ([]string, error) {
	op = strings.TrimSpace(op)
	if op == "" {
		return nil, nil
	}

	fields := strings.SplitN(op, ",", 3)
	switch restOp := strings.ToLower(fields[0]); restOp {
	case "post", "put":
		if len(fields) < 3 {
			return nil, stackerr.Wrap(errInvalidFormat)
		}
	case "delete":
		if len(fields) != 2 {
			return nil, stackerr.Wrap(errInvalidFormat)
		}
		subFields := strings.SplitN(fields[1], ":", 2)
		if len(subFields) == 2 {
			fields = append(fields, subFields[1])
			fields[1] = subFields[0]
		}
		return fields, nil
	default:
		return nil, stackerr.Wrap(errInvalidFormat)
	}

	if c.baseWebhookURL != nil {
		u, err := c.baseWebhookURL.Parse(fields[2])
		if err != nil {
			return nil, stackerr.Wrap(err)
		}
		fields[2] = u.String()
	}

	switch subFields := strings.SplitN(fields[1], ":", 2); len(subFields) {
	case 1:
		return fields, nil
	case 2:
		fields = append(fields, fields[2])
		fields[3] = fields[2]
		fields[2] = subFields[1]
		fields[1] = subFields[0]
		return fields, nil
	}
	return nil, stackerr.Wrap(errInvalidFormat)
}

func (c *configureCmd) createHooksOperations(
	e *env,
	reader io.Reader,
) ([]*hookOperation, error) {
	scanner := bufio.NewScanner(reader)
	var (
		hooksOps []*hookOperation
		added    bool
	)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		fields, err := c.processHooksOperation(e, scanner.Text())
		if err != nil {
			return nil, err
		}
		added, hooksOps, err = c.appendHookOperation(e, fields, hooksOps)
		if err != nil {
			return nil, err
		}
		if !added {
			fmt.Fprintf(e.Out, "Ignoring line: %d\n", lineNum)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, stackerr.Wrap(err)
	}
	return hooksOps, nil
}

func (c *configureCmd) checkStrictMode(restOp string, exists bool) (string, bool, error) {
	restOp = strings.ToUpper(restOp)
	if !exists {
		if restOp == "PUT" {
			if c.hooksStrict {
				return "", false, stackerr.Wrap(errPutToPost)
			}
			return "POST", true, nil
		}
		if restOp == "DELETE" {
			if c.hooksStrict {
				return "", false, stackerr.Wrap(errNotExist)
			}
			return "DELETE", true, nil
		}
	} else if restOp == "POST" {
		if c.hooksStrict {
			return "", false, stackerr.Wrap(errPostToPut)
		}
		return "PUT", true, nil
	}
	return restOp, false, nil
}

func (c *configureCmd) functionHookExists(e *env, name string) (bool, error) {
	functionsURL, err := url.Parse(path.Join(defaultFunctionsURL, name))
	if err != nil {
		return false, stackerr.Wrap(err)
	}
	var results struct {
		Results []*functionHook `json:"results,omitempty"`
	}
	_, err = e.ParseAPIClient.Get(functionsURL, &results)
	if err != nil {
		if strings.Contains(err.Error(), "is defined") {
			return false, nil
		}
		return false, stackerr.Wrap(err)
	}
	for _, result := range results.Results {
		if result.URL != "" && result.FunctionName == name {
			return true, nil
		}
	}
	return false, nil
}

func (c *configureCmd) deployFunctionHook(e *env, op *hookOperation) error {
	if op.function == nil {
		return stackerr.New("cannot deploy nil function hook")
	}
	exists, err := c.functionHookExists(e, op.function.FunctionName)
	if err != nil {
		return err
	}

	restOp, suppressed, err := c.checkStrictMode(op.method, exists)
	if err != nil {
		return err
	}

	function := &functionHooksCmd{Function: op.function}
	switch restOp {
	case "POST":
		return function.functionHooksCreate(e, nil)
	case "PUT":
		return function.functionHooksUpdate(e, nil)
	case "DELETE":
		if suppressed {
			return nil
		}
		return function.functionHooksDelete(e, nil)
	}
	return stackerr.Wrap(errInvalidFormat)
}

func (c *configureCmd) triggerHookExists(e *env, className, triggerName string) (bool, error) {
	triggersURL, err := url.Parse(path.Join(defaultTriggersURL, className, triggerName))
	if err != nil {
		return false, stackerr.Wrap(err)
	}
	var results struct {
		Results []*triggerHook `json:"results,omitempty"`
	}
	_, err = e.ParseAPIClient.Get(triggersURL, &results)
	if err != nil {
		if strings.Contains(err.Error(), "is defined") {
			return false, nil
		}
		return false, stackerr.Wrap(err)
	}
	for _, result := range results.Results {
		if result.URL != "" && result.ClassName == className && result.TriggerName == triggerName {
			return true, nil
		}
	}
	return false, nil
}

func (c *configureCmd) deployTriggerHook(e *env, op *hookOperation) error {
	if op.trigger == nil {
		return stackerr.New("cannot deploy nil trigger hook")
	}

	exists, err := c.triggerHookExists(e, op.trigger.ClassName, op.trigger.TriggerName)
	if err != nil {
		return err
	}
	restOp, suppressed, err := c.checkStrictMode(op.method, exists)
	if err != nil {
		return err
	}

	trigger := &triggerHooksCmd{Trigger: op.trigger, All: false}
	switch restOp {
	case "POST":
		return trigger.triggerHooksCreate(e, nil)
	case "PUT":
		return trigger.triggerHooksUpdate(e, nil)
	case "DELETE":
		if suppressed {
			return nil
		}
		return trigger.triggerHooksDelete(e, nil)
	}
	return stackerr.Wrap(errInvalidFormat)

}

func (c *configureCmd) deployWebhooksConfig(e *env, hooksOps []*hookOperation) error {
	for _, op := range hooksOps {
		if op.function == nil && op.trigger == nil {
			return stackerr.New("hook operation is neither a function, not a trigger.")
		}
		if op.function != nil && op.trigger != nil {
			return stackerr.New("a hook cannot be both a function and a trigger.")
		}
		if op.function != nil {
			if err := c.deployFunctionHook(e, op); err != nil {
				return err
			}
		} else {
			if err := c.deployTriggerHook(e, op); err != nil {
				return err
			}
		}
		fmt.Fprintln(e.Out)
	}
	return nil
}

func (c *configureCmd) parseBaseURL(e *env) error {
	if c.baseURL != "" {
		u, err := url.Parse(c.baseURL)
		if err != nil {
			fmt.Fprintln(e.Err, "Invalid base webhook url provided")
			return stackerr.Wrap(err)
		}
		if u.Scheme != "https" {
			return stackerr.New("Please provide a valid https url")
		}
		c.baseWebhookURL = u
	}
	return nil
}

func (c *configureCmd) hooksCmd(e *env, ctx *context, args []string) error {
	if len(args) > 1 {
		return fmt.Errorf("Invalid args: %v, only an optional hooks config file is expected.", args)
	}
	if err := c.parseBaseURL(e); err != nil {
		return err
	}

	reader := e.In
	if len(args) == 1 {
		file, err := os.Open(args[0])
		if err != nil {
			return stackerr.Wrap(err)
		}
		reader = ioutil.NopCloser(file)
	} else {
		fmt.Fprintln(e.Out, "Since a webhooks config file was not provided reading from stdin.")
	}
	hooksOps, err := c.createHooksOperations(e, reader)
	if err != nil {
		return err
	}
	err = c.deployWebhooksConfig(e, hooksOps)
	if err != nil {
		fmt.Fprintln(
			e.Out,
			"Failed to deploy the webhooks config. Please try again...",
		)
		return err
	}
	fmt.Fprintln(e.Out, "Successfully configured the given webhooks for the app.")
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

	hooksCmd := &cobra.Command{
		Use:   "hooks [filename]",
		Short: "Configure webhooks according to given config file",
		Long: `Configure webhooks for the app based on the given configuration csv file.
For more details read: https://parse.com/docs/js/guide#command-line-webhooks
`,
		Run:     runWithArgsClient(e, c.hooksCmd),
		Aliases: []string{"webhooks"},
	}
	hooksCmd.Flags().BoolVarP(&c.hooksStrict, "strict", "s", c.hooksStrict,
		"Configure hooks in strict mode, i.e., do not automatically fix errors.")
	hooksCmd.Flags().StringVarP(&c.baseURL, "base", "b", c.baseURL,
		`Base url to use while parsing the webhook url field.
If provided, the config file can have relative urls.`)
	cmd.AddCommand(hooksCmd)

	return cmd
}
