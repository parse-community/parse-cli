package parsecli

import (
	"regexp"
	"testing"

	"github.com/facebookgo/ensure"
	"github.com/facebookgo/errgroup"
	"github.com/facebookgo/parse"
	"github.com/facebookgo/stackerr"
)

func TestGetHostFromURL(t *testing.T) {
	urlStr := "https://api.parse.com/1/"
	host, err := getHostFromURL(urlStr, "")
	ensure.Nil(t, err)
	ensure.DeepEqual(t, host, "api.parse.com")

	urlStr = "https://api.parse.com/1/"
	host, err = getHostFromURL(urlStr, "yolo@earth.com")
	ensure.Nil(t, err)
	ensure.DeepEqual(t, host, "api.parse.com#yolo@earth.com")

	urlStr = "https://api.example.com:8080/1/"
	host, err = getHostFromURL(urlStr, "")
	ensure.Nil(t, err)
	ensure.DeepEqual(t, host, "api.example.com")

	urlStr = "https://api.example.com:8080/1/"
	host, err = getHostFromURL(urlStr, "yolo@earth.com")
	ensure.Nil(t, err)
	ensure.DeepEqual(t, host, "api.example.com#yolo@earth.com")

	urlStr = "api.example.com:8080:90"
	host, err = getHostFromURL(urlStr, "")
	ensure.Err(t, err, regexp.MustCompile("not a valid url"))
}

func TestLastFour(t *testing.T) {
	t.Parallel()

	ensure.DeepEqual(t, Last4(""), "")
	ensure.DeepEqual(t, Last4("a"), "a")
	ensure.DeepEqual(t, Last4("ab"), "ab")
	ensure.DeepEqual(t, Last4("abc"), "abc")
	ensure.DeepEqual(t, Last4("abcd"), "abcd")
	ensure.DeepEqual(t, Last4("abcdefg"), "***defg")
}

func TestErrorStringWithoutStack(t *testing.T) {
	t.Parallel()
	h := NewHarness(t)
	defer h.Stop()
	h.Env.ErrorStack = false
	const message = "hello world"
	actual := ErrorString(h.Env, stackerr.New(message))
	ensure.StringContains(t, actual, message)
	ensure.StringDoesNotContain(t, actual, ".go")
}

type exitCode int

func TestErrorStringWithStack(t *testing.T) {
	t.Parallel()
	h := NewHarness(t)
	defer h.Stop()
	h.Env.ErrorStack = true
	const message = "hello world"
	actual := ErrorString(h.Env, stackerr.New(message))
	ensure.StringContains(t, actual, message)
	ensure.StringContains(t, actual, ".go")
}

func TestErrorString(t *testing.T) {
	t.Parallel()
	h := NewHarness(t)
	defer h.Stop()
	apiErr := &parse.Error{Code: -1, Message: "Error\nMessage"}
	ensure.DeepEqual(t, `Error
Message`,
		ErrorString(h.Env, apiErr),
	)
}

func TestStackErrorString(t *testing.T) {
	t.Parallel()

	h := NewHarness(t)
	defer h.Stop()

	err := stackerr.New("error")

	h.Env.ErrorStack = false
	errStr := ErrorString(h.Env, err)

	ensure.DeepEqual(t, errStr, "error")

	h.Env.ErrorStack = true
	errStr = ErrorString(h.Env, err)

	ensure.StringContains(t, errStr, "error")
	ensure.StringContains(t, errStr, ".go")

	err = stackerr.Wrap(&parse.Error{Message: "message", Code: 1})
	h.Env.ErrorStack = false
	errStr = ErrorString(h.Env, err)

	ensure.DeepEqual(t, errStr, "message")

	h.Env.ErrorStack = true
	errStr = ErrorString(h.Env, err)

	ensure.StringContains(t, errStr, `parse: api error with code=1 and message="message`)
	ensure.StringContains(t, errStr, ".go")
}

func TestMultiErrorString(t *testing.T) {
	t.Parallel()

	h := NewHarness(t)
	defer h.Stop()

	err := errgroup.MultiError(
		[]error{
			stackerr.New("error"),
			stackerr.Wrap(&parse.Error{Message: "message", Code: 1}),
		},
	)

	h.Env.ErrorStack = false
	errStr := ErrorString(h.Env, err)

	ensure.DeepEqual(t, errStr, "multiple errors: error | message")

	h.Env.ErrorStack = true
	errStr = ErrorString(h.Env, err)

	ensure.StringContains(t, errStr, "multiple errors")
	ensure.StringContains(t, errStr, `parse: api error with code=1 and message="message"`)
	ensure.StringContains(t, errStr, ".go")
}
