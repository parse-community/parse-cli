package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

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
func runWithClient(e *env, f func(*env, *context) error) cobraRun {
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
		cl, err := newContext(e, app)
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
func runWithAppClient(e *env, f func(*env, *context) error) cobraRun {
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
		cl, err := newContext(e, app)
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
func runWithArgsClient(e *env, f func(*env, *context, []string) error) cobraRun {
	return func(cmd *cobra.Command, args []string) {
		app := defaultKey
		if len(args) > 1 {
			app = args[0]
			args = args[1:]
		}
		cl, err := newContext(e, app)
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
