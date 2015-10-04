package parsecli

import (
	"errors"
	"io/ioutil"
	"testing"

	"github.com/facebookgo/ensure"
	"github.com/spf13/cobra"
)

func noOpCmd() *cobra.Command {
	var c cobra.Command
	c.SetOutput(ioutil.Discard)
	return &c
}

func TestRunNoArgsFuncError(t *testing.T) {
	t.Parallel()
	const message = "hello world"
	h := NewHarness(t)
	defer h.Stop()
	h.Env.Exit = func(i int) { panic(exitCode(i)) }
	func() {
		defer ensure.PanicDeepEqual(t, exitCode(1))
		r := RunNoArgs(h.Env, func(*Env) error { return errors.New(message) })
		r(noOpCmd(), nil)
	}()
	ensure.StringContains(t, h.Err.String(), message)
}

func TestRunWithAppMultipleArgs(t *testing.T) {
	t.Parallel()
	h := NewHarness(t)
	defer h.Stop()
	h.Env.Exit = func(i int) { panic(exitCode(i)) }
	func() {
		defer ensure.PanicDeepEqual(t, exitCode(1))
		r := RunWithClient(h.Env, nil)
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
	h := NewHarness(t)
	defer h.Stop()
	h.MakeEmptyRoot()
	h.Env.Exit = func(i int) { panic(exitCode(i)) }
	func() {
		defer ensure.PanicDeepEqual(t, exitCode(1))
		r := RunWithClient(h.Env, nil)
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
	h := NewHarness(t)
	defer h.Stop()
	c := legacyConfig{Applications: map[string]*ParseAppConfig{"a": {}}}
	h.MakeWithConfig(jsonStr(t, c))
	h.Env.Exit = func(i int) { panic(exitCode(i)) }
	func() {
		defer ensure.PanicDeepEqual(t, exitCode(1))
		r := RunWithClient(h.Env, nil)
		r(noOpCmd(), []string{"b"})
	}()
	ensure.StringContains(t, h.Err.String(), `App "b" wasn't found`)
}

func TestRunWithAppNamed(t *testing.T) {
	t.Parallel()
	h := NewHarness(t)
	defer h.Stop()
	const appName = "a"
	c := &ParseConfig{
		Applications: map[string]*ParseAppConfig{
			appName: {ApplicationID: "id", MasterKey: "token"},
		},
	}
	h.MakeWithConfig(jsonStr(t, c))
	r := RunWithClient(h.Env, func(e *Env, c *Context) error {
		ensure.NotNil(t, c)
		return nil
	})
	r(noOpCmd(), []string{"a"})
}

func TestRunWithDefault(t *testing.T) {
	t.Parallel()
	h := NewHarness(t)
	defer h.Stop()
	const appName = "a"
	c := &ParseConfig{
		Applications: map[string]*ParseAppConfig{
			appName:    {ApplicationID: "id", MasterKey: "token"},
			DefaultKey: {Link: appName},
		},
	}
	h.MakeWithConfig(jsonStr(t, c))
	r := RunWithClient(h.Env, func(e *Env, c *Context) error {
		ensure.NotNil(t, c)
		return nil
	})
	r(noOpCmd(), []string{"a"})
}

func TestRunWithAppError(t *testing.T) {
	t.Parallel()
	h := NewHarness(t)
	defer h.Stop()
	const appName = "a"
	c := &ParseConfig{
		Applications: map[string]*ParseAppConfig{
			appName: {ApplicationID: "id", MasterKey: "token"},
		},
	}
	h.MakeWithConfig(jsonStr(t, c))
	h.Env.Exit = func(i int) { panic(exitCode(i)) }
	const message = "hello world"
	func() {
		defer ensure.PanicDeepEqual(t, exitCode(1))
		r := RunWithClient(h.Env, func(e *Env, c *Context) error {
			ensure.NotNil(t, c)
			return errors.New(message)
		})
		r(noOpCmd(), []string{"a"})
	}()
	ensure.StringContains(t, h.Err.String(), message)
}
