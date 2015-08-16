package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

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

	if _, err := e.Client.Do(req, nil, &res); err != nil {
		return "", stackerr.Wrap(err)
	}
	return res.Warning, nil
}
