package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/facebookgo/errgroup"
	"github.com/facebookgo/parse"
	"github.com/facebookgo/stackerr"
)

var numbers = regexp.MustCompile("[0-9]+")

func numericLessThan(a, b string) bool {
	aLen, bLen := len(a), len(b)
	minLen := aLen
	if bLen < minLen {
		minLen = bLen
	}
	if minLen == 0 {
		if aLen == 0 {
			return bLen != 0
		}
		return false
	}

	i := 0
	for ; i < minLen && a[i] == b[i]; i++ {
	}
	if i == minLen {
		if aLen == bLen {
			return false
		}
		if aLen == minLen {
			return true
		}
		return false
	}

	a, b = a[i:], b[i:]
	aNos, bNos := numbers.FindAllStringIndex(a, 1), numbers.FindAllStringIndex(b, 1)

	if len(aNos) == 1 && aNos[0][0] == 0 &&
		len(bNos) == 1 && bNos[0][0] == 0 {
		aNum, err := strconv.Atoi(a[:aNos[0][1]])
		if err != nil {
			return a < b
		}
		bNum, err := strconv.Atoi(b[:bNos[0][1]])
		if err != nil {
			return a < b
		}
		if aNum != bNum {
			return aNum < bNum
		}
	}
	return a < b
}

type naturalStrings []string

func (v naturalStrings) Len() int {
	return len(v)
}

func (v naturalStrings) Less(i, j int) bool {
	return !numericLessThan(v[i], v[j]) // desc order
}

func (v naturalStrings) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

func getConfirmation(message string, e *env) bool {
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

func getHostFromURL(urlStr string) (string, error) {
	netURL, err := url.Parse(urlStr)
	if err != nil {
		return "", stackerr.Wrap(err)
	}
	server := regexp.MustCompile(`(.*):\d+$`).ReplaceAllString(netURL.Host, "$1")
	if server == "" {
		return "", stackerr.Newf("%s is not a valid url", urlStr)
	}
	return server, nil
}

func checkIfSupported(e *env, version string) (string, error) {
	v := make(url.Values)
	v.Set("version", version)
	req := &http.Request{
		Method: "GET",
		URL:    &url.URL{Path: "supported", RawQuery: v.Encode()},
	}

	var res struct {
		Warning string `json:"warning"`
	}

	if _, err := e.ParseAPIClient.Do(req, nil, &res); err != nil {
		return "", stackerr.Wrap(err)
	}
	return res.Warning, nil
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
