package parsecmd

import (
	"regexp"
	"testing"

	"github.com/ParsePlatform/parse-cli/parsecli"
	"github.com/facebookgo/ensure"
)

func TestRunSymbolsCmd(t *testing.T) {
	t.Parallel()

	h := parsecli.NewHarness(t)
	defer h.Stop()

	s := &symbolsCmd{}
	ensure.Err(t, s.run(h.Env, nil), regexp.MustCompile("Please specify path"))
}
