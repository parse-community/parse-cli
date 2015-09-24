package main

import (
	"regexp"
	"testing"

	"github.com/facebookgo/ensure"
)

func TestRunSymbolsCmd(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	defer h.Stop()

	s := &symbolsCmd{}
	ensure.Err(t, s.run(h.env, nil), regexp.MustCompile("Please specify path"))
}
