package main

import (
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/ParsePlatform/parse-cli/parsecli"
	"github.com/facebookgo/ensure"
)

func TestConfigureAccountKey(t *testing.T) {
	t.Parallel()

	h := parsecli.NewTokenHarness(t)
	defer h.Stop()

	c := configureCmd{login: parsecli.Login{TokenReader: strings.NewReader("")}}

	h.Env.In = ioutil.NopCloser(strings.NewReader("token\n"))
	ensure.Nil(t, c.accountKey(h.Env))
	ensure.StringContains(
		t,
		h.Out.String(),
		`
Input your account key or press ENTER to generate a new one.
`)

	h.Env.In = ioutil.NopCloser(strings.NewReader("invalid\n"))
	ensure.Err(t, c.accountKey(h.Env), regexp.MustCompile("is not valid"))
	ensure.DeepEqual(t,
		h.Err.String(),
		"Could not store credentials. Please try again.\n\n",
	)

	h.Env.Server = "http://api.parse.com/1/"
	c.tokenReader = strings.NewReader(
		`machine api.parse.com#email
			login default
			password token2
		`,
	)
	h.Err.Reset()
	h.Env.In = ioutil.NopCloser(strings.NewReader("token\n"))
	ensure.Nil(t, c.accountKey(h.Env))
	ensure.DeepEqual(t, h.Err.String(),
		`Note: this operation will overwrite the account key:
 "*oken"
for email: "email"
`)

	h.Env.Server = "http://api.parse.com/1/"
	c.tokenReader = strings.NewReader(
		`machine api.parse.com#email
			login default
			password token2
		`,
	)
	c.isDefault = true
	h.Err.Reset()
	h.Env.In = ioutil.NopCloser(strings.NewReader("token\n"))
	ensure.Nil(t, c.accountKey(h.Env))
	ensure.DeepEqual(t, h.Err.String(), "")

	h.Env.Server = "http://api.parse.com/1/"
	c.tokenReader = strings.NewReader(
		`machine api.parse.com
			login default
			password token2
		`,
	)
	c.isDefault = true
	h.Err.Reset()
	h.Env.In = ioutil.NopCloser(strings.NewReader("token\n"))
	ensure.Nil(t, c.accountKey(h.Env))
	ensure.DeepEqual(t, h.Err.String(), "Note: this operation will overwrite the default account key\n")

	h.Env.Server = "http://api.parse.com/1/"
	c.tokenReader = strings.NewReader(
		`machine api.parse.com
			login default
			password token2
		`,
	)
	h.Err.Reset()
	c.isDefault = false
	h.Env.In = ioutil.NopCloser(strings.NewReader("token\n"))
	ensure.Nil(t, c.accountKey(h.Env))
	ensure.DeepEqual(t, h.Err.String(), "")
}

func TestParserEmail(t *testing.T) {
	t.Parallel()

	h := parsecli.NewTokenHarness(t)
	h.MakeEmptyRoot()
	defer h.Stop()

	ensure.Nil(t, parsecli.CreateConfigWithContent(filepath.Join(h.Env.Root, parsecli.ParseLocal), "{}"))
	ensure.Nil(t,
		parsecli.CreateConfigWithContent(
			filepath.Join(h.Env.Root, parsecli.ParseProject),
			`{"project_type": 1}`,
		),
	)

	var c configureCmd
	ensure.Nil(t, c.parserEmail(h.Env, []string{"email2"}))
	ensure.DeepEqual(
		t,
		h.Out.String(),
		`Successfully configured email for current project to: "email2"
`,
	)

	ensure.Err(t, c.parserEmail(h.Env, nil), regexp.MustCompile("Invalid args:"))
	ensure.Err(t, c.parserEmail(h.Env, []string{"a", "b"}), regexp.MustCompile("Invalid args:"))
}

func TestProjectType(t *testing.T) {
	t.Parallel()
	h := parsecli.NewHarness(t)
	defer h.Stop()

	h.MakeEmptyRoot()
	ensure.Nil(t, parsecli.CloneSampleCloudCode(h.Env, false))

	c := &configureCmd{}
	err := c.projectType(h.Env, []string{"1", "2"})
	ensure.Err(t, err, regexp.MustCompile("only an optional project type argument is expected"))

	h.Env.In = ioutil.NopCloser(strings.NewReader("invalid\n"))
	err = c.projectType(h.Env, nil)
	ensure.StringContains(t, h.Err.String(), "Invalid selection. Please enter a number")
	ensure.Err(t, err, regexp.MustCompile("Could not make a selection. Please try again."))
	h.Err.Reset()
	h.Out.Reset()

	h.Env.In = ioutil.NopCloser(strings.NewReader("0\n"))
	err = c.projectType(h.Env, nil)
	ensure.StringContains(t, h.Err.String(), "Please enter a number between 1 and")
	ensure.Err(t, err, regexp.MustCompile("Could not make a selection. Please try again."))
	h.Err.Reset()
	h.Out.Reset()

	h.Env.In = ioutil.NopCloser(strings.NewReader("1\n"))
	err = c.projectType(h.Env, nil)
	ensure.StringContains(t, h.Out.String(), "Successfully set project type to: parse")
	ensure.Nil(t, err)
}
