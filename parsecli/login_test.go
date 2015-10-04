package parsecli

import (
	"regexp"
	"strings"
	"testing"

	"github.com/facebookgo/ensure"
)

func TestPopulateCreds(t *testing.T) {
	t.Parallel()

	h := NewHarness(t)
	defer h.Stop()

	l := &Login{}
	h.Env.In = strings.NewReader("email\npassword\n")
	ensure.Nil(t, l.populateCreds(h.Env))
	ensure.DeepEqual(t, l.Credentials.Email, "email")
	ensure.DeepEqual(t, l.Credentials.Password, "password")
}

func TestGetTokenCredentials(t *testing.T) {
	t.Parallel()

	h := NewHarness(t)
	defer h.Stop()

	l := &Login{}
	h.Env.Server = "http://api.example.com/1/"

	l.TokenReader = strings.NewReader(
		`machine api.example.com
			login default
			password token
		`,
	)
	_, credentials, err := l.GetTokenCredentials(h.Env, "")
	ensure.Nil(t, err)
	ensure.DeepEqual(t, credentials.Token, "token")

	l.TokenReader = strings.NewReader(
		`machine api.example.com
			login default
			password token
		`,
	)
	h.Env.Server = "http://api.parse.com"
	_, credentials, err = l.GetTokenCredentials(h.Env, "")
	ensure.Err(t, err, keyNotFound)

	l = &Login{}
	h.Env.Server = "http://api.example.com/1/"

	l.TokenReader = strings.NewReader(
		`machine api.example.com#email
			login default
			password token
		`,
	)
	_, credentials, err = l.GetTokenCredentials(h.Env, "email")
	ensure.Nil(t, err)
	ensure.DeepEqual(t, credentials.Token, "token")

	l.TokenReader = strings.NewReader(
		`machine api.example.com#email
			login default
			password token
		`,
	)
	h.Env.Server = "http://api.parse.com"
	_, credentials, err = l.GetTokenCredentials(h.Env, "email")
	ensure.Err(t, err, keyNotFound)

	l = &Login{}
	h.Env.Server = "http://api.example.com/1/"

	l.TokenReader = strings.NewReader(
		`machine api.example.com#email
		login default
		password token1
machine api.example.com
		login default
		password token2
`,
	)
	_, credentials, err = l.GetTokenCredentials(h.Env, "email")
	ensure.Nil(t, err)
	ensure.DeepEqual(t, credentials.Token, "token1")

	l.TokenReader = strings.NewReader(
		`machine api.example.com#email
		login default
		password token1
machine api.example.com
		login default
		password token2
`,
	)

	_, credentials, err = l.GetTokenCredentials(h.Env, "xmail")
	ensure.Nil(t, err)
	ensure.DeepEqual(t, credentials.Token, "token2")
}

func TestAuthUserWithToken(t *testing.T) {
	t.Parallel()

	h := NewTokenHarness(t)
	defer h.Stop()

	l := &Login{}
	h.Env.ParserEmail = "email"
	h.Env.Server = "http://api.example.org/1/"

	l.TokenReader = strings.NewReader(
		`machine api.example.org#email
			login default
			password token
		`,
	)
	_, err := l.authUserWithToken(h.Env)
	ensure.Nil(t, err)

	h.Env.ParserEmail = "email2"

	l.TokenReader = strings.NewReader(
		`machine api.example.org#email2
			login default
			password token
		`,
	)
	_, err = l.authUserWithToken(h.Env)
	ensure.Err(t, err, regexp.MustCompile(`does not belong to "email2"`))

	h.Env.ParserEmail = ""

	l.TokenReader = strings.NewReader(
		`machine api.example.org
			login default
			password token2
		`,
	)
	_, err = l.authUserWithToken(h.Env)
	ensure.Err(t, err, regexp.MustCompile("provided is not valid"))

}

func TestUpdatedNetrcContent(t *testing.T) {
	t.Parallel()

	h := NewHarness(t)
	defer h.Stop()

	l := &Login{}

	h.Env.Server = "https://api.example.com/1/"
	updated, err := l.updatedNetrcContent(
		h.Env,
		strings.NewReader(
			`machine api.example.com#email
  login default
  password token0

machine  api.example.org
  login default
  password token
`,
		),
		"email",
		&Credentials{Token: "token"},
	)

	ensure.Nil(t, err)
	ensure.DeepEqual(t,
		string(updated),
		`machine api.example.com#email
  login default
  password token

machine  api.example.org
  login default
  password token
`,
	)

	h.Env.Server = "https://api.example.org/1/"
	updated, err = l.updatedNetrcContent(h.Env,
		strings.NewReader(
			`machine api.example.com
	login default
	password token
`,
		),
		"email",
		&Credentials{Token: "token"},
	)

	ensure.Nil(t, err)
	ensure.DeepEqual(t,
		string(updated),
		`machine api.example.com
	login default
	password token

machine api.example.org#email
	login default
	password token`,
	)
}
