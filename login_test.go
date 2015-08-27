package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/facebookgo/ensure"
	"github.com/facebookgo/parse"
)

func newTokenHarness(t testing.TB) *Harness {
	h := newHarness(t)
	ht := transportFunc(func(r *http.Request) (*http.Response, error) {
		ensure.DeepEqual(t, r.URL.Path, "/1/accountkey")
		ensure.DeepEqual(t, r.Method, "POST")

		key := &struct {
			AccountKey string `json:"accountKey"`
		}{}
		ensure.Nil(t, json.NewDecoder(ioutil.NopCloser(r.Body)).Decode(key))

		if key.AccountKey != "token" {
			return &http.Response{
				StatusCode: http.StatusUnauthorized,
				Body:       ioutil.NopCloser(strings.NewReader(`{"error": "incorrect token"}`)),
			}, nil
		}

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(strings.NewReader(`{"email": "email"}`)),
		}, nil
	})
	h.env.ParseAPIClient = &ParseAPIClient{apiClient: &parse.Client{Transport: ht}}
	return h
}

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
	_, credentials, err := l.getTokenCredentials(h.env, "")
	ensure.Nil(t, err)
	ensure.DeepEqual(t, credentials.token, "token")

	l.tokenReader = strings.NewReader(
		`machine api.example.com
			login default
			password token
		`,
	)
	h.env.Server = "http://api.parse.com"
	_, credentials, err = l.getTokenCredentials(h.env, "")
	ensure.Err(t, err, keyNotFound)

	l = &login{}
	h.env.Server = "http://api.example.com/1/"

	l.tokenReader = strings.NewReader(
		`machine api.example.com#email
			login default
			password token
		`,
	)
	_, credentials, err = l.getTokenCredentials(h.env, "email")
	ensure.Nil(t, err)
	ensure.DeepEqual(t, credentials.token, "token")

	l.tokenReader = strings.NewReader(
		`machine api.example.com#email
			login default
			password token
		`,
	)
	h.env.Server = "http://api.parse.com"
	_, credentials, err = l.getTokenCredentials(h.env, "email")
	ensure.Err(t, err, keyNotFound)

	l = &login{}
	h.env.Server = "http://api.example.com/1/"

	l.tokenReader = strings.NewReader(
		`machine api.example.com#email
		login default
		password token1
machine api.example.com
		login default
		password token2
`,
	)
	_, credentials, err = l.getTokenCredentials(h.env, "email")
	ensure.Nil(t, err)
	ensure.DeepEqual(t, credentials.token, "token1")

	l.tokenReader = strings.NewReader(
		`machine api.example.com#email
		login default
		password token1
machine api.example.com
		login default
		password token2
`,
	)

	_, credentials, err = l.getTokenCredentials(h.env, "xmail")
	ensure.Nil(t, err)
	ensure.DeepEqual(t, credentials.token, "token2")
}

func TestAuthUserWithToken(t *testing.T) {
	t.Parallel()

	h := newTokenHarness(t)
	defer h.Stop()

	l := &login{}
	h.env.ParserEmail = "email"
	h.env.Server = "http://api.example.org/1/"

	l.tokenReader = strings.NewReader(
		`machine api.example.org#email
			login default
			password token
		`,
	)
	_, err := l.authUserWithToken(h.env)
	ensure.Nil(t, err)

	h.env.ParserEmail = "email2"

	l.tokenReader = strings.NewReader(
		`machine api.example.org#email2
			login default
			password token
		`,
	)
	_, err = l.authUserWithToken(h.env)
	ensure.Err(t, err, regexp.MustCompile(`does not belong to "email2"`))

	h.env.ParserEmail = ""

	l.tokenReader = strings.NewReader(
		`machine api.example.org
			login default
			password token2
		`,
	)
	_, err = l.authUserWithToken(h.env)
	ensure.Err(t, err, regexp.MustCompile("provided is not valid"))

}

func TestUpdatedNetrcContent(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	defer h.Stop()

	l := &login{}

	h.env.Server = "https://api.example.com/1/"
	updated, err := l.updatedNetrcContent(
		h.env,
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
		&credentials{token: "token"},
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

	h.env.Server = "https://api.example.org/1/"
	updated, err = l.updatedNetrcContent(h.env,
		strings.NewReader(
			`machine api.example.com
	login default
	password token
`,
		),
		"email",
		&credentials{token: "token"},
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
