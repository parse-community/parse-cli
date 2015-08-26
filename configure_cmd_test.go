package main

import (
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/facebookgo/ensure"
)

func TestConfigureAccessToken(t *testing.T) {
	t.Parallel()

	h := newTokenHarness(t)
	defer h.Stop()

	c := configureCmd{login: login{tokenReader: strings.NewReader("")}}
	h.env.In = ioutil.NopCloser(strings.NewReader("token\n"))
	ensure.Nil(t, c.accountKey(h.env))
	ensure.DeepEqual(
		t,
		h.Out.String(),
		`
Input your account key or press enter to generate a new one.
Account Key: Successfully stored credentials.
`)
	h.env.In = ioutil.NopCloser(strings.NewReader("email\ninvalid\n"))
	ensure.Err(t, c.accountKey(h.env), regexp.MustCompile("is not valid"))
	ensure.DeepEqual(t,
		h.Err.String(),
		"Could not store credentials. Please try again.\n\n",
	)
}

func TestParserEmail(t *testing.T) {
	t.Parallel()

	h := newTokenHarness(t)
	h.makeEmptyRoot()
	defer h.Stop()

	var n newCmd
	ensure.Nil(t, n.createConfigWithContent(filepath.Join(h.env.Root, parseLocal), "{}"))
	ensure.Nil(t,
		n.createConfigWithContent(
			filepath.Join(h.env.Root, parseProject),
			`{"project_type": 1}`,
		),
	)

	var c configureCmd
	ensure.Nil(t, c.parserEmail(h.env, []string{"email2"}))
	ensure.DeepEqual(
		t,
		h.Out.String(),
		`Successfully configured email for current project to: "email2"
`,
	)

	ensure.Err(t, c.parserEmail(h.env, nil), regexp.MustCompile("Invalid args:"))
	ensure.Err(t, c.parserEmail(h.env, []string{"a", "b"}), regexp.MustCompile("Invalid args:"))
}
