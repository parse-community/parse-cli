package main

import (
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/facebookgo/ensure"
)

func TestConfigureAccountKey(t *testing.T) {
	t.Parallel()

	h := newTokenHarness(t)
	defer h.Stop()

	c := configureCmd{login: login{tokenReader: strings.NewReader("")}}

	h.env.In = ioutil.NopCloser(strings.NewReader("token\n"))
	ensure.Nil(t, c.accountKey(h.env))
	ensure.StringContains(
		t,
		h.Out.String(),
		`
Input your account key or press enter to generate a new one.
`)

	h.env.In = ioutil.NopCloser(strings.NewReader("invalid\n"))
	ensure.Err(t, c.accountKey(h.env), regexp.MustCompile("is not valid"))
	ensure.DeepEqual(t,
		h.Err.String(),
		"Could not store credentials. Please try again.\n\n",
	)

	h.env.Server = "http://api.parse.com/1/"
	c.tokenReader = strings.NewReader(
		`machine api.parse.com#email
			login default
			password token2
		`,
	)
	h.Err.Reset()
	h.env.In = ioutil.NopCloser(strings.NewReader("token\n"))
	ensure.Nil(t, c.accountKey(h.env))
	ensure.DeepEqual(t, h.Err.String(),
		`Note: this operation will overwrite the account key:
 "*oken"
for email: "email"
`)

	h.env.Server = "http://api.parse.com/1/"
	c.tokenReader = strings.NewReader(
		`machine api.parse.com#email
			login default
			password token2
		`,
	)
	c.isDefault = true
	h.Err.Reset()
	h.env.In = ioutil.NopCloser(strings.NewReader("token\n"))
	ensure.Nil(t, c.accountKey(h.env))
	ensure.DeepEqual(t, h.Err.String(), "")

	h.env.Server = "http://api.parse.com/1/"
	c.tokenReader = strings.NewReader(
		`machine api.parse.com
			login default
			password token2
		`,
	)
	c.isDefault = true
	h.Err.Reset()
	h.env.In = ioutil.NopCloser(strings.NewReader("token\n"))
	ensure.Nil(t, c.accountKey(h.env))
	ensure.DeepEqual(t, h.Err.String(), "Note: this operation will overwrite the default account key\n")

	h.env.Server = "http://api.parse.com/1/"
	c.tokenReader = strings.NewReader(
		`machine api.parse.com
			login default
			password token2
		`,
	)
	h.Err.Reset()
	c.isDefault = false
	h.env.In = ioutil.NopCloser(strings.NewReader("token\n"))
	ensure.Nil(t, c.accountKey(h.env))
	ensure.DeepEqual(t, h.Err.String(), "")
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

func TestProjectType(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	defer h.Stop()

	h.makeEmptyRoot()
	ensure.Nil(t, (&newCmd{}).cloneSampleCloudCode(h.env, false))

	c := &configureCmd{}
	err := c.projectType(h.env, []string{"1", "2"})
	ensure.Err(t, err, regexp.MustCompile("only an optional project type argument is expected"))

	h.env.In = ioutil.NopCloser(strings.NewReader("invalid\n"))
	err = c.projectType(h.env, nil)
	ensure.StringContains(t, h.Err.String(), "Invalid selection. Please enter a number")
	ensure.Err(t, err, regexp.MustCompile("Could not make a selection. Please try again."))
	h.Err.Reset()
	h.Out.Reset()

	h.env.In = ioutil.NopCloser(strings.NewReader("0\n"))
	err = c.projectType(h.env, nil)
	ensure.StringContains(t, h.Err.String(), "Please enter a number between 1 and")
	ensure.Err(t, err, regexp.MustCompile("Could not make a selection. Please try again."))
	h.Err.Reset()
	h.Out.Reset()

	h.env.In = ioutil.NopCloser(strings.NewReader("1\n"))
	err = c.projectType(h.env, nil)
	ensure.StringContains(t, h.Out.String(), "Successfully set project type to: parse")
	ensure.Nil(t, err)
}
