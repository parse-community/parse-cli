package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/ParsePlatform/parse-cli/parsecli"
	"github.com/facebookgo/clock"
	"github.com/facebookgo/stackerr"
	"github.com/spf13/cobra"
)

func main() {
	// some parts of apps.go are unable to handle
	// interrupts, this logic ensures we exit on system interrupts
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	go func() {
		<-interrupt
		os.Exit(1)
	}()

	e := parsecli.Env{
		Root:        os.Getenv("PARSE_ROOT"),
		Server:      os.Getenv("PARSE_SERVER"),
		ErrorStack:  os.Getenv("PARSE_ERROR_STACK") == "1",
		ParserEmail: os.Getenv("PARSER_EMAIL"),
		Out:         os.Stdout,
		Err:         os.Stderr,
		In:          os.Stdin,
		Exit:        os.Exit,
		Clock:       clock.New(),
	}
	if e.Root == "" {
		cur, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(e.Err, "Failed to get current directory:\n%s\n", err)
			os.Exit(1)
		}
		root := parsecli.GetProjectRoot(&e, cur)
		if parsecli.IsProjectDir(root) {
			e.Root = root
			config, err := parsecli.ConfigFromDir(root)
			if err != nil {
				fmt.Fprintln(e.Err, err)
				os.Exit(1)
			}
			e.Type = config.GetProjectConfig().Type
			if e.ParserEmail == "" {
				e.ParserEmail = config.GetProjectConfig().ParserEmail
			}
		} else {
			e.Type = parsecli.LegacyParseFormat
			e.Root = parsecli.GetLegacyProjectRoot(&e, cur)
		}
	}
	if e.Type != parsecli.LegacyParseFormat && e.Type != parsecli.ParseFormat {
		fmt.Fprintf(e.Err, "Unknown project type %d.\n", e.Type)
		os.Exit(1)
	}

	if e.Server == "" {
		e.Server = parsecli.DefaultBaseURL
	}

	apiClient, err := parsecli.NewParseAPIClient(&e)
	if err != nil {
		fmt.Fprintln(e.Err, err)
		os.Exit(1)
	}
	e.ParseAPIClient = apiClient

	var (
		rootCmd *cobra.Command
		command []string
	)
	switch e.Type {
	case parsecli.LegacyParseFormat, parsecli.ParseFormat:
		command, rootCmd = parseRootCmd(&e)
	}

	if len(command) == 0 || command[0] != "update" {
		message, err := checkIfSupported(&e, parsecli.Version, command...)
		if err != nil {
			fmt.Fprintln(e.Err, err)
			os.Exit(1)
		}
		if message != "" {
			fmt.Fprintln(e.Err, message)
		}
	}

	if err := rootCmd.Execute(); err != nil {
		// Error is already printed in Execute()
		os.Exit(1)
	}
}

func checkIfSupported(e *parsecli.Env, version string, args ...string) (string, error) {
	v := make(url.Values)
	v.Set("version", version)
	v.Set("other", strings.Join(args, " "))
	req := &http.Request{
		Method: "GET",
		URL:    &url.URL{Path: "supported", RawQuery: v.Encode()},
	}

	type result struct {
		warning string
		err     error
	}

	timeout := make(chan *result, 1)
	go func() {
		var res struct {
			Warning string `json:"warning"`
		}
		_, err := e.ParseAPIClient.Do(req, nil, &res)
		timeout <- &result{warning: res.Warning, err: err}
	}()

	select {
	case res := <-timeout:
		return res.warning, stackerr.Wrap(res.err)
	case <-time.After(time.Duration(500) * time.Millisecond):
		return "", nil
	}
	return "", nil
}
