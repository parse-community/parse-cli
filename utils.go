package main

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/ParsePlatform/parse-cli/parsecli"
	"github.com/facebookgo/stackerr"
)

func getConfirmation(message string, e *parsecli.Env) bool {
	fmt.Fprintf(e.Out, message)
	var confirm string
	fmt.Fscanf(e.In, "%s\n", &confirm)
	lower := strings.ToLower(confirm)
	return lower != "" && strings.HasPrefix(lower, "y")
}

func validateURL(urlStr string) error {
	netURL, err := url.Parse(urlStr)
	if err != nil {
		return stackerr.Wrap(err)
	}

	if netURL.Scheme != "https" {
		return errors.New("Please enter a valid https url")
	}
	return nil
}
