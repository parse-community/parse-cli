package main

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"time"

	"github.com/facebookgo/clock"
	"github.com/facebookgo/errgroup"
	"github.com/facebookgo/parse"
	"github.com/facebookgo/stackerr"
	"github.com/spf13/cobra"
)

const (
	version        = "2.2.1"
	cloudDir       = "cloud"
	hostingDir     = "public"
	defaultBaseURL = "https://api.parse.com/1/"
)

var userAgent = fmt.Sprintf("parse-cli-%s-%s", runtime.GOOS, version)

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

// errorString returns the error string with our without the stack trace
// depending on the environment variable. this exists because we want plain
// messages for end users, but when we're working on the CLI we want the stack
// trace for debugging.
func errorString(e *env, err error) string {
	type hasUnderlying interface {
		HasUnderlying() error
	}

	parseErr := func(err error) error {
		if apiErr, ok := err.(*parse.Error); ok {
			return errors.New(apiErr.Message)
		}
		return err
	}

	lastErr := func(err error) error {
		if serr, ok := err.(*stackerr.Error); ok {
			if errs := stackerr.Underlying(serr); len(errs) != 0 {
				err = errs[len(errs)-1]
			}
		} else {
			if eu, ok := err.(hasUnderlying); ok {
				err = eu.HasUnderlying()
			}
		}

		return parseErr(err)
	}

	if !e.ErrorStack {
		if merr, ok := err.(errgroup.MultiError); ok {
			var multiError []error
			for _, ierr := range []error(merr) {
				multiError = append(multiError, lastErr(ierr))
			}
			err = errgroup.MultiError(multiError)
		} else {
			err = lastErr(err)
		}
		return parseErr(err).Error()
	}

	return err.Error()
}

type env struct {
	Root       string // project root
	Server     string // parse api server
	Type       int    // project type
	ErrorStack bool
	Out        io.Writer
	Err        io.Writer
	In         io.Reader
	Exit       func(int)
	Clock      clock.Clock
	Client     *Client
}

type client struct {
	Config    config
	AppName   string
	AppConfig appConfig
}

func newParseClient(e *env) (*Client, error) {
	baseURL, err := url.Parse(e.Server)
	if err != nil {
		return nil, stackerr.Newf("invalid server URL %q: %s", e.Server, err)
	}
	return &Client{
		client: &parse.Client{
			BaseURL: baseURL,
		},
	}, nil
}

func newClient(e *env, appName string) (*client, error) {
	config, err := configFromDir(e.Root)
	if err != nil {
		return nil, err
	}

	app, err := config.app(appName)
	if err != nil {
		return nil, err
	}

	masterKey, err := app.getMasterKey(e)
	if err != nil {
		return nil, err
	}
	e.Client = e.Client.WithCredentials(
		parse.MasterKey{
			ApplicationID: app.getApplicationID(),
			MasterKey:     masterKey,
		})

	return &client{
		AppName:   appName,
		AppConfig: app,
		Config:    config,
	}, nil
}

type cobraRun func(cmd *cobra.Command, args []string)

// runNoArgs wraps a run function that shouldn't get any arguments.
func runNoArgs(e *env, f func(*env) error) cobraRun {
	return func(cmd *cobra.Command, args []string) {
		if len(args) != 0 {
			fmt.Fprintf(e.Err, "unexpected arguments:%+v\n\n", args)
			cmd.Help()
			e.Exit(1)
		}
		if err := f(e); err != nil {
			fmt.Fprintln(e.Err, errorString(e, err))
			e.Exit(1)
		}
	}
}

// runWithArgs wraps a run function that can access arguments to cobraCmd
func runWithArgs(e *env, f func(*env, []string) error) cobraRun {
	return func(cmd *cobra.Command, args []string) {
		if err := f(e, args); err != nil {
			fmt.Fprintln(e.Err, errorString(e, err))
			e.Exit(1)
		}
	}
}

// runWithClient wraps a run function that should get an app, when the default is
// picked from the config in the current working directory.
func runWithClient(e *env, f func(*env, *client) error) cobraRun {
	return func(cmd *cobra.Command, args []string) {
		app := defaultKey
		if len(args) > 1 {
			fmt.Fprintf(
				e.Err,
				"unexpected arguments, only an optional app name is expected:%+v\n\n",
				args,
			)
			cmd.Help()
			e.Exit(1)
		}
		if len(args) == 1 {
			app = args[0]
		}
		cl, err := newClient(e, app)
		if err != nil {
			fmt.Fprintln(e.Err, errorString(e, err))
			e.Exit(1)
		}
		if err := f(e, cl); err != nil {
			fmt.Fprintln(e.Err, errorString(e, err))
			e.Exit(1)
		}
	}
}

// runWithAppClient wraps a run function that should get an app
func runWithAppClient(e *env, f func(*env, *client) error) cobraRun {
	return func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Fprintf(e.Err, "please provide an app name\n\n")
			cmd.Help()
			e.Exit(1)
		}
		if len(args) > 1 {
			fmt.Fprintf(
				e.Err,
				"unexpected arguments, only an app name is expected:%+v\n\n",
				args,
			)
			cmd.Help()
			e.Exit(1)
		}
		app := args[0]
		cl, err := newClient(e, app)
		if err != nil {
			fmt.Fprintln(e.Err, errorString(e, err))
			e.Exit(1)
		}
		if err := f(e, cl); err != nil {
			fmt.Fprintln(e.Err, errorString(e, err))
			e.Exit(1)
		}
	}
}

// runWithArgsClient wraps a run function that should get an app, whee the default is
// picked from the config in the current working directory. It also passes args to the
// runner function
func runWithArgsClient(e *env, f func(*env, *client, []string) error) cobraRun {
	return func(cmd *cobra.Command, args []string) {
		app := defaultKey
		if len(args) > 1 {
			app = args[0]
			args = args[1:]
		}
		cl, err := newClient(e, app)
		if err != nil {
			fmt.Fprintln(e.Err, errorString(e, err))
			e.Exit(1)
		}
		if err := f(e, cl, args); err != nil {
			fmt.Fprintln(e.Err, errorString(e, err))
			e.Exit(1)
		}
	}
}

func rootCmd(e *env) *cobra.Command {
	c := &cobra.Command{
		Use: "parse",
		Long: fmt.Sprintf(
			`Parse Command Line Interface
Version %v
Copyright %d Parse, Inc.
http://parse.com`,
			version,
			time.Now().Year(),
		),
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	c.AddCommand(newAddCmd(e))
	c.AddCommand(newConfigureCmd(e))
	c.AddCommand(newDefaultCmd(e))
	c.AddCommand(newDeployCmd(e))
	c.AddCommand(newDevelopCmd(e))
	c.AddCommand(newFunctionHooksCmd(e))
	c.AddCommand(newGenerateCmd(e))
	c.AddCommand(newJsSdkCmd(e))
	c.AddCommand(newListCmd(e))
	c.AddCommand(newLogsCmd(e))
	c.AddCommand(newNewCmd(e))
	c.AddCommand(newReleasesCmd(e))
	c.AddCommand(newRollbackCmd(e))
	c.AddCommand(newSymbolsCmd(e))
	c.AddCommand(newTriggerHooksCmd(e))
	c.AddCommand(newUpdateCmd(e))
	c.AddCommand(newVersionCmd(e))

	if len(os.Args) <= 1 {
		return c
	}

	commands := []string{"help"}
	for _, command := range c.Commands() {
		commands = append(commands, command.Name())
	}

	args := make([]string, len(os.Args)-1)
	copy(args, os.Args[1:])

	if message := makeCorrections(commands, args); message != "" {
		fmt.Fprintln(e.Out, message)
	}
	c.SetArgs(args)

	return c
}

func getProjectRoot(e *env, cur string) string {
	if _, err := os.Stat(filepath.Join(cur, legacyConfigFile)); err == nil {
		return cur
	}

	root := cur
	base := filepath.Base(root)

	for base != "." && base != string(filepath.Separator) {
		base = filepath.Base(root)
		root = filepath.Dir(root)
		if base == "cloud" || base == "public" || base == "config" {
			if _, err := os.Stat(filepath.Join(root, legacyConfigFile)); err == nil {
				return root
			}
		}
	}

	return cur
}

func main() {
	// some parts of apps_cmd.go are unable to handle
	// interrupts, this logic ensures we exit on system interrupts
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	go func() {
		<-interrupt
		os.Exit(1)
	}()

	e := env{
		Root:       os.Getenv("PARSE_ROOT"),
		Server:     os.Getenv("PARSE_SERVER"),
		ErrorStack: os.Getenv("PARSE_ERROR_STACK") == "1",
		Out:        os.Stdout,
		Err:        os.Stderr,
		In:         os.Stdin,
		Exit:       os.Exit,
		Clock:      clock.New(),
	}
	if e.Root == "" {
		cur, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(e.Err, "Failed to get current directory:\n%s\n", err)
			os.Exit(1)
		}
		e.Root = getProjectRoot(&e, cur)
	}
	e.Type = legacy
	if e.Server == "" {
		e.Server = defaultBaseURL
	}
	client, err := newParseClient(&e)
	if err != nil {
		fmt.Fprintln(e.Err, err)
		os.Exit(1)
	}
	e.Client = client

	message, err := checkIfSupported(&e, version)
	if err != nil {
		fmt.Fprintln(e.Err, err)
		os.Exit(1)
	}
	if message != "" {
		fmt.Fprintln(e.Err, message)
	}

	if err := rootCmd(&e).Execute(); err != nil {
		// Error is already printed in Execute()
		os.Exit(1)
	}
}
