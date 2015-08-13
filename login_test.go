package main

import (
	"regexp"
	"strings"
	"testing"

	"github.com/facebookgo/ensure"
)

func TestPopulateCreds(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	defer h.Stop()

	l := &login{}
	h.env.In = strings.NewReader("email\npassword\n")
	ensure.Nil(t, l.populateCreds(h.env))
	ensure.DeepEqual(t, l.credentials.email, "email")
	ensure.DeepEqual(t, l.credentials.password, "password")
}

func TestGetTokenCredentials(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	defer h.Stop()

	l := &login{}
	h.env.Server = "http://api.example.com/1/"

	l.tokenReader = strings.NewReader(
		`machine api.example.com
			login default
			password token
		`,
	)
	credentials, err := l.getTokenCredentials(h.env)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, credentials.token, "token")

	h.env.Server = "http://api.parse.com"
	credentials, err = l.getTokenCredentials(h.env)
	ensure.Err(t, err, regexp.MustCompile("could not find token for"))
}

func TestAuthUserWithToken(t *testing.T) {
	t.Parallel()

	h, _ := newAppHarness(t)
	defer h.Stop()

	l := &login{}
	h.env.Server = "http://api.example.org/1/"

	l.tokenReader = strings.NewReader(
		`machine api.example.org
			login email
			password token
		`,
	)
	ensure.Nil(t, l.authUserWithToken(h.env))
}

func TestUpdatedNetrcContent(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	defer h.Stop()

	l := &login{}

	h.env.Server = "https://api.example.com/1/"
	updated, err := l.updatedNetrcContent(h.env,
		strings.NewReader(
			`machine api.example.com
  login default
  password token0

machine  api.example.org
  login default
  password token
`,
		),
		&credentials{token: "token"},
	)

	ensure.Nil(t, err)
	ensure.DeepEqual(t,
		string(updated),
		`machine api.example.com
  login default
  password token

machine  api.example.org
  login default
  password token
`,
	)

	h.env.Server = "https://api.example.org/1/"
	updated, err = l.updatedNetrcContent(h.env,
		strings.NewReader(
			`machine api.example.com
	login default
	password token
`,
		),
		&credentials{token: "token"},
	)

	ensure.Nil(t, err)
	ensure.DeepEqual(t,
		string(updated),
		`machine api.example.com
	login default
	password token

machine api.example.org
	login default
	password token`,
	)
}
