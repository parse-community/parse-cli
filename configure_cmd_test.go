package main

import (
	"io/ioutil"
	"regexp"
	"strings"
	"testing"

	"github.com/facebookgo/ensure"
)

func TestConfigureAcessToken(t *testing.T) {
	t.Parallel()

	h, _ := newAppHarness(t)
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
	ensure.Err(t, c.accountKey(h.env), regexp.MustCompile("Please try again"))
	ensure.DeepEqual(t,
		h.Err.String(),
		`Sorry, the account key you provided is not valid.
Please follow instructions at https://www.parse.com/account_keys to generate a new account key.
`,
	)
}
