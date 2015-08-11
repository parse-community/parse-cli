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
	h.env.In = ioutil.NopCloser(strings.NewReader("n\nemail\ntoken\n"))
	ensure.Nil(t, c.accessToken(h.env))
	ensure.DeepEqual(t,
		h.Out.String(),
		`Please enter the email id you used to register with Parse
and an access token if you already generated it.
If you do not have an access token or would like to generate a new one,
please type: "y" to open the browser or "n" to continue: Email: Account Key: Successfully stored credentials.
`)

	h.env.In = ioutil.NopCloser(strings.NewReader("n\nemail\ninvalid\n"))
	ensure.Err(t, c.accessToken(h.env), regexp.MustCompile("Please try again"))
	ensure.DeepEqual(t,
		h.Err.String(),
		`Sorry, we do not have a user with this email and access token.
Please follow instructions at https://www.parse.com/account/account_keys to generate a new access token.
`)
}
