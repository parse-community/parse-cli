package main

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"

	"github.com/facebookgo/clock"
	"github.com/spf13/cobra"
)

const (
	version        = "2.2.4"
	cloudDir       = "cloud"
	hostingDir     = "public"
	defaultBaseURL = "https://api.parse.com/1/"
)

var userAgent = fmt.Sprintf("parse-cli-%s-%s", runtime.GOOS, version)

type env struct {
	Root           string // project root
	Server         string // parse api server
	Type           int    // project type
	ParserEmail    string // email associated with developer parse account
	ErrorStack     bool
	Out            io.Writer
	Err            io.Writer
	In             io.Reader
	Exit           func(int)
	Clock          clock.Clock
	ParseAPIClient *ParseAPIClient
}

type client struct {
	Config    config
	AppName   string
	AppConfig appConfig
}

func main() {
	// some parts of apps.go are unable to handle
	// interrupts, this logic ensures we exit on system interrupts
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	go func() {
		<-interrupt
		os.Exit(1)
	}()

	e := env{
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
		root := getProjectRoot(&e, cur)
		if isProjectDir(root) {
			e.Root = root
			config, err := configFromDir(root)
			if err != nil {
				fmt.Fprintln(e.Err, err)
				os.Exit(1)
			}
			e.Type = config.getProjectConfig().Type
			if e.ParserEmail == "" {
				e.ParserEmail = config.getProjectConfig().ParserEmail
			}
		} else {
			e.Type = legacyParseFormat
			e.Root = getLegacyProjectRoot(&e, cur)
		}
	}
	if e.Type != legacyParseFormat && e.Type != parseFormat {
		fmt.Fprintf(e.Err, "Unknown project type %d.\n", e.Type)
		os.Exit(1)
	}

	if e.Server == "" {
		e.Server = defaultBaseURL
	}

	apiClient, err := newParseAPIClient(&e)
	if err != nil {
		fmt.Fprintln(e.Err, err)
		os.Exit(1)
	}
	e.ParseAPIClient = apiClient

	message, err := checkIfSupported(&e, version)
	if err != nil {
		fmt.Fprintln(e.Err, err)
		os.Exit(1)
	}
	if message != "" {
		fmt.Fprintln(e.Err, message)
	}

	var rootCmd *cobra.Command
	switch e.Type {
	case legacyParseFormat, parseFormat:
		rootCmd = parseRootCmd(&e)
	}
	if err := rootCmd.Execute(); err != nil {
		// Error is already printed in Execute()
		os.Exit(1)
	}
}
