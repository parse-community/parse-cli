package parsecmd

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/ParsePlatform/parse-cli/parsecli"
	"github.com/facebookgo/ensure"
)

func TestContDeploy(t *testing.T) {
	t.Parallel()

	h := createParseProject(t)
	defer h.Stop()

	deployer := deployFunc(func(parseVersion string,
		prevDeplInfo *deployInfo,
		forDevelop bool,
		e *parsecli.Env) (*deployInfo, error) {
		return &deployInfo{}, nil
	})

	done := make(chan struct{})
	go func() {
		h.Clock.Add(time.Second)
		close(done)
	}()

	first := make(chan struct{})
	(&developCmd{}).contDeploy(h.Env, deployer, first, done)
	_, opened := <-first
	ensure.False(t, opened)
}

func TestContDeployConfigErr(t *testing.T) {
	t.Parallel()

	h := createParseProject(t)
	defer h.Stop()

	ensure.Nil(t,
		ioutil.WriteFile(
			filepath.Join(h.Env.Root, parsecli.LegacyConfigFile),
			[]byte("}"),
			0600,
		),
	)
	h.Env.Type = parsecli.LegacyParseFormat

	deployer := deployFunc(func(parseVersion string,
		prevDeplInfo *deployInfo,
		forDevelop bool,
		e *parsecli.Env) (*deployInfo, error) {
		return &deployInfo{}, nil
	})

	done := make(chan struct{})
	go func() {
		h.Clock.Add(5 * time.Second)
		close(done)
	}()

	first := make(chan struct{})
	(&developCmd{}).contDeploy(h.Env, deployer, first, done)
	_, opened := <-first
	ensure.False(t, opened)

	ensure.StringContains(
		t,
		h.Err.String(),
		fmt.Sprintf(
			`Config malformed.
Please fix your config file in %s and try again.
`,
			filepath.Join(h.Env.Root, parsecli.LegacyConfigFile),
		),
	)
}

func TestHandleError(t *testing.T) {
	t.Parallel()
	h := parsecli.NewHarness(t)
	defer h.Stop()

	d := &developCmd{}
	netErr := &net.OpError{Err: errors.New("network")}
	sleep := func(d time.Duration) {}
	ensure.DeepEqual(t, d.handleError(h.Env, netErr, sleep), netErr)
	ensure.DeepEqual(t, h.Err.String(), "Flaky network. Waiting 20s before trying to fetch logs again.")

	h.Out.Reset()
	h.Err.Reset()
	otherErr := errors.New("other")
	ensure.DeepEqual(t, d.handleError(h.Env, otherErr, sleep), otherErr)
	ensure.DeepEqual(t, h.Err.Len(), 0)
}
