package main

import (
	"io/ioutil"
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
			login email
			password token
		`,
	)
	credentials, err := l.getTokenCredentials(h.env)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, credentials.email, "email")
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
  login email
  password token0

machine  api.example.org
  login email
  password token
`,
		),
		&credentials{email: "email", token: "token"},
	)

	ensure.Nil(t, err)
	ensure.DeepEqual(t,
		string(updated),
		`machine api.example.com
  login email
  password token

machine  api.example.org
  login email
  password token
`,
	)

	h.env.Server = "https://api.example.org/1/"
	updated, err = l.updatedNetrcContent(h.env,
		strings.NewReader(
			`machine api.example.com
	login email
	password token
`,
		),
		&credentials{email: "email", token: "token"},
	)

	ensure.Nil(t, err)
	ensure.DeepEqual(t,
		string(updated),
		`machine api.example.com
	login email
	password token

machine api.example.org
	login email
	password token`,
	)
}

func TestLogin(t *testing.T) {
	t.Parallel()

	h, _ := newAppHarness(t)
	defer h.Stop()

	l := &login{tokenReader: strings.NewReader("")}

	h.env.In = ioutil.NopCloser(strings.NewReader("n\nemail\ntoken\n"))
	ensure.Nil(t, l.run(h.env))
	ensure.DeepEqual(t,
		h.Out.String(),
		`Please enter the email id you used to register with Parse
and an account key if you already generated it.
If you do not have an account key or would like to generate a new one,
please type: "y" to open the browser or "n" to continue: Email: Account Key: Successfully stored credentials.
`)

	h.env.In = ioutil.NopCloser(strings.NewReader("n\nemail\ninvalid\n"))
	ensure.Err(t, l.run(h.env), regexp.MustCompile("Please try again"))
	ensure.DeepEqual(t,
		h.Err.String(),
		`Sorry, we do not have a user with this email and account key.
Please follow instructions at https://www.parse.com/account/keys to generate a new account key.
`)
}
