package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/facebookgo/clock"
	"github.com/facebookgo/ensure"
	"github.com/facebookgo/errgroup"
	"github.com/facebookgo/parse"
	"github.com/facebookgo/stackerr"
	"github.com/facebookgo/testname"
	"github.com/spf13/cobra"
)

func noOpCmd() *cobra.Command {
	var c cobra.Command
	c.SetOutput(ioutil.Discard)
	return &c
}

type Harness struct {
	T      testing.TB
	Out    bytes.Buffer
	Err    bytes.Buffer
	Clock  *clock.Mock
	env    *env
	remove []string
}

func (h *Harness) makeEmptyRoot() {
	var err error
	prefix := fmt.Sprintf("%s-", testname.Get("parse-cli-"))
	h.env.Root, err = ioutil.TempDir("", prefix)
	ensure.Nil(h.T, err)
	h.remove = append(h.remove, h.env.Root)
}

func (h *Harness) makeWithConfig(global string) {
	h.env.Root = makeDirWithConfig(h.T, global)
}

func (h *Harness) Stop() {
	for _, p := range h.remove {
		os.RemoveAll(p)
	}
}

func newHarness(t testing.TB) *Harness {
	te := Harness{
		T:     t,
		Clock: clock.NewMock(),
	}
	te.env = &env{
		Out:            &te.Out,
		Err:            &te.Err,
		Clock:          te.Clock,
		ParseAPIClient: &ParseAPIClient{apiClient: &parse.Client{}},
	}
	return &te
}

// makes a temp directory with the given global config.
func makeDirWithConfig(t testing.TB, global string) string {
	dir, err := ioutil.TempDir("", testname.Get("parse-cli-"))
	ensure.Nil(t, err)
	ensure.Nil(t, os.Mkdir(filepath.Join(dir, "config"), 0755))
	ensure.Nil(t, ioutil.WriteFile(
		filepath.Join(dir, legacyConfigFile),
		[]byte(global),
		os.FileMode(0600),
	))
	return dir
}

type transportFunc func(r *http.Request) (*http.Response, error)

func (t transportFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return t(r)
}

func jsonStr(t testing.TB, v interface{}) string {
	b, err := json.Marshal(v)
	ensure.Nil(t, err)
	return string(b)
}

func TestVersion(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	defer h.Stop()
	var c versionCmd
	err := c.run(h.env)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, h.Out.String(), version+"\n")
	ensure.DeepEqual(t, h.Err.String(), "")
}

// this just exists to make code coverage less noisy.
func TestRootCommand(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	defer h.Stop()
	parseRootCmd(h.env)
}

func TestErrorStringWithoutStack(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	defer h.Stop()
	h.env.ErrorStack = false
	const message = "hello world"
	actual := errorString(h.env, stackerr.New(message))
	ensure.StringContains(t, actual, message)
	ensure.StringDoesNotContain(t, actual, ".go")
}

type exitCode int

func TestErrorStringWithStack(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	defer h.Stop()
	h.env.ErrorStack = true
	const message = "hello world"
	actual := errorString(h.env, stackerr.New(message))
	ensure.StringContains(t, actual, message)
	ensure.StringContains(t, actual, ".go")
}

func TestRunNoArgsWithArg(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	defer h.Stop()
	h.env.Exit = func(i int) { panic(exitCode(i)) }
	func() {
		defer ensure.PanicDeepEqual(t, exitCode(1))
		r := runNoArgs(h.env, nil)
		r(noOpCmd(), []string{"foo"})
	}()
	ensure.StringContains(t, h.Err.String(), "unexpected arguments")
}

func TestRunNoArgsFuncError(t *testing.T) {
	t.Parallel()
	const message = "hello world"
	h := newHarness(t)
	defer h.Stop()
	h.env.Exit = func(i int) { panic(exitCode(i)) }
	func() {
		defer ensure.PanicDeepEqual(t, exitCode(1))
		r := runNoArgs(h.env, func(*env) error { return errors.New(message) })
		r(noOpCmd(), nil)
	}()
	ensure.StringContains(t, h.Err.String(), message)
}

func TestRunWithAppMultipleArgs(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	defer h.Stop()
	h.env.Exit = func(i int) { panic(exitCode(i)) }
	func() {
		defer ensure.PanicDeepEqual(t, exitCode(1))
		r := runWithClient(h.env, nil)
		r(noOpCmd(), []string{"foo", "bar"})
	}()
	ensure.StringContains(
		t,
		h.Err.String(),
		"only an optional app name is expected",
	)
}

func TestRunWithAppNonProjectDir(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	defer h.Stop()
	h.makeEmptyRoot()
	h.env.Exit = func(i int) { panic(exitCode(i)) }
	func() {
		defer ensure.PanicDeepEqual(t, exitCode(1))
		r := runWithClient(h.env, nil)
		r(noOpCmd(), nil)
	}()
	ensure.StringContains(
		t,
		h.Err.String(),
		"Command must be run inside a Parse project.",
	)
}

func TestRunWithAppNotFound(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	defer h.Stop()
	c := legacyConfig{Applications: map[string]*parseAppConfig{"a": {}}}
	h.makeWithConfig(jsonStr(t, c))
	h.env.Exit = func(i int) { panic(exitCode(i)) }
	func() {
		defer ensure.PanicDeepEqual(t, exitCode(1))
		r := runWithClient(h.env, nil)
		r(noOpCmd(), []string{"b"})
	}()
	ensure.StringContains(t, h.Err.String(), `App "b" wasn't found`)
}

func TestRunWithAppNamed(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	defer h.Stop()
	const appName = "a"
	c := &parseConfig{
		Applications: map[string]*parseAppConfig{
			appName: {ApplicationID: "id", MasterKey: "token"},
		},
	}
	h.makeWithConfig(jsonStr(t, c))
	r := runWithClient(h.env, func(e *env, c *context) error {
		ensure.NotNil(t, c)
		return nil
	})
	r(noOpCmd(), []string{"a"})
}

func TestRunWithDefault(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	defer h.Stop()
	const appName = "a"
	c := &parseConfig{
		Applications: map[string]*parseAppConfig{
			appName:    {ApplicationID: "id", MasterKey: "token"},
			defaultKey: {Link: appName},
		},
	}
	h.makeWithConfig(jsonStr(t, c))
	r := runWithClient(h.env, func(e *env, c *context) error {
		ensure.NotNil(t, c)
		return nil
	})
	r(noOpCmd(), []string{"a"})
}

func TestRunWithAppError(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	defer h.Stop()
	const appName = "a"
	c := &parseConfig{
		Applications: map[string]*parseAppConfig{
			appName: {ApplicationID: "id", MasterKey: "token"},
		},
	}
	h.makeWithConfig(jsonStr(t, c))
	h.env.Exit = func(i int) { panic(exitCode(i)) }
	const message = "hello world"
	func() {
		defer ensure.PanicDeepEqual(t, exitCode(1))
		r := runWithClient(h.env, func(e *env, c *context) error {
			ensure.NotNil(t, c)
			return errors.New(message)
		})
		r(noOpCmd(), []string{"a"})
	}()
	ensure.StringContains(t, h.Err.String(), message)
}

func TestNewClientInvalidServerURL(t *testing.T) {
	t.Parallel()
	c, err := newParseAPIClient(&env{Server: ":"})
	ensure.True(t, c == nil)
	ensure.Err(t, err, regexp.MustCompile("invalid server URL"))
}

func TestErrorString(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	defer h.Stop()
	apiErr := &parse.Error{Code: -1, Message: "Error\nMessage"}
	ensure.DeepEqual(t, `Error
Message`,
		errorString(h.env, apiErr),
	)
}

func TestGetProjectRoot(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.makeEmptyRoot()
	defer h.Stop()

	ensure.Nil(t, os.Mkdir(filepath.Join(h.env.Root, "parse"), 0755))
	ensure.Nil(t, os.Mkdir(filepath.Join(h.env.Root, "parse", "config"), 0755))
	f, err := os.Create(filepath.Join(h.env.Root, "parse", legacyConfigFile))
	ensure.Nil(t, err)
	defer f.Close()
	ensure.Nil(t, os.Mkdir(filepath.Join(h.env.Root, "parse", "cloud"), 0755))
	ensure.Nil(t, os.Mkdir(filepath.Join(h.env.Root, "parse", "public"), 0755))
	ensure.Nil(t, os.MkdirAll(filepath.Join(h.env.Root, "parse", "cloud", "other", "config"), 0755))

	ensure.DeepEqual(t, getLegacyProjectRoot(h.env, h.env.Root), h.env.Root)

	ensure.DeepEqual(t, getLegacyProjectRoot(h.env, filepath.Join(h.env.Root, "parse", "config")), filepath.Join(h.env.Root, "parse"))

	ensure.DeepEqual(t, getLegacyProjectRoot(h.env, filepath.Join(h.env.Root, "parse", "cloud")), filepath.Join(h.env.Root, "parse"))

	ensure.DeepEqual(t, getLegacyProjectRoot(h.env, filepath.Join(h.env.Root, "parse", "public")), filepath.Join(h.env.Root, "parse"))

	ensure.DeepEqual(t, getLegacyProjectRoot(h.env, filepath.Join(h.env.Root, "parse", "cloud", "other")), filepath.Join(h.env.Root, "parse"))
}

func TestStackErrorString(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	defer h.Stop()

	err := stackerr.New("error")

	h.env.ErrorStack = false
	errStr := errorString(h.env, err)

	ensure.DeepEqual(t, errStr, "error")

	h.env.ErrorStack = true
	errStr = errorString(h.env, err)

	ensure.StringContains(t, errStr, "error")
	ensure.StringContains(t, errStr, ".go")

	err = stackerr.Wrap(&parse.Error{Message: "message", Code: 1})
	h.env.ErrorStack = false
	errStr = errorString(h.env, err)

	ensure.DeepEqual(t, errStr, "message")

	h.env.ErrorStack = true
	errStr = errorString(h.env, err)

	ensure.StringContains(t, errStr, `parse: api error with code=1 and message="message`)
	ensure.StringContains(t, errStr, ".go")
}

func TestMultiErrorString(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	defer h.Stop()

	err := errgroup.MultiError(
		[]error{
			stackerr.New("error"),
			stackerr.Wrap(&parse.Error{Message: "message", Code: 1}),
		},
	)

	h.env.ErrorStack = false
	errStr := errorString(h.env, err)

	ensure.DeepEqual(t, errStr, "multiple errors: error | message")

	h.env.ErrorStack = true
	errStr = errorString(h.env, err)

	ensure.StringContains(t, errStr, "multiple errors")
	ensure.StringContains(t, errStr, `parse: api error with code=1 and message="message"`)
	ensure.StringContains(t, errStr, ".go")
}
