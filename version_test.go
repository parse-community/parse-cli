package main

import (
	"testing"

	"github.com/ParsePlatform/parse-cli/parsecli"
	"github.com/facebookgo/ensure"
)

func TestVersion(t *testing.T) {
	t.Parallel()
	h := parsecli.NewHarness(t)
	defer h.Stop()
	var c versionCmd
	err := c.run(h.Env)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, h.Out.String(), parsecli.Version+"\n")
	ensure.DeepEqual(t, h.Err.String(), "")
}
