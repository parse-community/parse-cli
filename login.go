package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bgentry/go-netrc/netrc"
	"github.com/bgentry/speakeasy"
	"github.com/facebookgo/jsonpipe"
	"github.com/facebookgo/stackerr"
	"github.com/mitchellh/go-homedir"
	"github.com/skratchdot/open-golang/open"
)

const keysURL = "https://www.parse.com/account_keys"

type credentials struct {
	email    string
	password string
	token    string
}

type login struct {
	credentials credentials
	tokenReader io.Reader
}

var (
	errAuth = errors.New(`Sorry, we do not have a user with this username and password.
If you do not remember your password, please follow instructions at:
  https://www.parse.com/login
to reset your password`)

	tokenErrMsgf = `Sorry, the account key: %q you provided is not valid.
Please follow instructions at %q to generate a new one.
`
	keyNotFound = regexp.MustCompile("Could not find access key")
	parseNetrc  = filepath.Join(".parse", "netrc")
)

func accessKeyNotFound(err error) bool {
	return keyNotFound.MatchString(err.Error())
}

func (l *login) populateCreds(e *env) error {
	if l.credentials.email != "" && l.credentials.password != "" {
		return nil
	}

	fmt.Fprint(e.Out, "Email: ")
	fmt.Fscanf(e.In, "%s\n", &l.credentials.email)

	var (
		password string
		err      error
	)
	if e.In == os.Stdin {
		password, err = speakeasy.Ask("Password (will be hidden): ")
		if err != nil {
			return err
		}
	} else {
		// NOTE: only for testing
		fmt.Fscanf(e.In, "%s\n", &password)
	}

	if password != "" {
		l.credentials.password = password
	}
	return nil
}

func (l *login) getTokensReader() (io.Reader, error) {
	if l.tokenReader != nil {
		return l.tokenReader, nil
	}
	homeDir, err := homedir.Dir()
	if err != nil {
		return nil, stackerr.Wrap(err)
	}
	location := filepath.Join(homeDir, parseNetrc)
	file, err := os.OpenFile(location, os.O_RDONLY, 0600)
	if err != nil {
		return nil, stackerr.Wrap(err)
	}
	l.tokenReader = file
	return file, nil
}

func (l *login) getTokenCredentials(e *env, email string) (*credentials, error) {
	reader, err := l.getTokensReader()
	if err != nil {
		return nil, stackerr.Wrap(err)
	}
	tokens, err := netrc.Parse(reader)
	if err != nil {
		return nil, stackerr.Wrap(err)
	}
	server, err := getHostFromURL(e.Server, email)
	if err != nil {
		return nil, err
	}
	machine := tokens.FindMachine(server)
	if machine != nil {
		return &credentials{
			token: machine.Password,
		}, nil
	}

	if email == "" {
		return nil, stackerr.Newf("Could not find access key for %q", server)
	}

	// check for system default account key for the given server
	// since we could not find account key for the given account (email)
	server, err = getHostFromURL(e.Server, "")
	if err != nil {
		return nil, err
	}
	machine = tokens.FindMachine(server)
	if machine != nil {
		return &credentials{
			token: machine.Password,
		}, nil
	}
	return nil, stackerr.Newf(
		`Could not find access key for email: %q,
and default access key not configured for %q
`,
		email,
		e.Server,
	)
}

func (l *login) updatedNetrcContent(
	e *env,
	content io.Reader,
	email string,
	credentials *credentials,
) ([]byte, error) {
	tokens, err := netrc.Parse(content)
	if err != nil {
		return nil, stackerr.Wrap(err)
	}

	server, err := getHostFromURL(e.Server, email)
	if err != nil {
		return nil, err
	}
	machine := tokens.FindMachine(server)
	if machine == nil {
		machine = tokens.NewMachine(server, "default", credentials.token, "")
	} else {
		machine.UpdatePassword(credentials.token)
	}

	updatedContent, err := tokens.MarshalText()
	if err != nil {
		return nil, stackerr.Wrap(err)
	}
	return updatedContent, nil
}

func (l *login) storeCredentials(e *env, email string, credentials *credentials) error {
	if l.tokenReader != nil {
		// tests should not store credentials
		return nil
	}

	homeDir, err := homedir.Dir()
	if err != nil {
		return stackerr.Wrap(err)
	}

	location := filepath.Join(homeDir, parseNetrc)
	if err := os.MkdirAll(filepath.Dir(location), 0755); err != nil {
		return stackerr.Wrap(err)
	}
	file, err := os.OpenFile(location, os.O_RDONLY|os.O_CREATE, 0600)
	if err != nil {
		return stackerr.Wrap(err)
	}
	content, err := l.updatedNetrcContent(e, file, email, credentials)
	if err != nil {
		return err
	}

	file, err = os.OpenFile(location, os.O_WRONLY|os.O_TRUNC, 0600)
	_, err = file.Write(content)
	return stackerr.Wrap(err)
}

func (l *login) authToken(e *env, token string) (string, error) {
	req := &http.Request{
		Method: "POST",
		URL:    &url.URL{Path: "accountkey"},
		Body: ioutil.NopCloser(
			jsonpipe.Encode(
				map[string]string{
					"accountKey": token,
				},
			),
		),
	}

	res := &struct {
		Email string `json:"email"`
	}{}
	if response, err := e.ParseAPIClient.Do(req, nil, res); err != nil {
		if response.StatusCode == http.StatusUnauthorized {
			return "", stackerr.Newf(tokenErrMsgf, last4(token), keysURL)
		}
		return "", stackerr.Wrap(err)
	}

	if e.ParserEmail != "" && res.Email != e.ParserEmail {
		return "", stackerr.Newf("Account key %q does not belong to %q", last4(token), e.ParserEmail)
	}
	return res.Email, nil
}

func (l *login) authUserWithToken(e *env) (string, error) {
	tokenCredentials, err := l.getTokenCredentials(e, e.ParserEmail)
	if err != nil {
		if stackerr.HasUnderlying(err, stackerr.MatcherFunc(accessKeyNotFound)) {
			fmt.Fprintln(e.Err, errorString(e, err))
		}
		return "", err
	}

	email, err := l.authToken(e, tokenCredentials.token)
	if err != nil {
		fmt.Fprintf(e.Err, "Account key could not be used.\nError: %s\n\n", errorString(e, err))
		return "", err
	}

	l.credentials = *tokenCredentials
	return email, nil
}

func (l *login) authUser(e *env) error {
	_, err := l.authUserWithToken(e)
	if err == nil {
		return nil
	}

	// user never created an account key: educate them
	if stackerr.HasUnderlying(err, stackerr.MatcherFunc(os.IsNotExist)) {
		fmt.Fprintln(
			e.Out,
			`We've changed the way the CLI works.
To save time logging in, you should create an account key.
`)
	}

	apps := &apps{}
	fmt.Fprintln(
		e.Out,
		`Type "parse configure accountkey" to create a new account key.

Please login to Parse using your email and password.`,
	)
	for i := 0; i < numRetries; i++ {
		err := l.populateCreds(e)
		if err != nil {
			return err
		}
		apps.login.credentials = l.credentials
		_, err = apps.restFetchApps(e)
		if err == nil {
			return nil
		}

		if i == numRetries-1 && err != nil {
			return err
		}
		if err != errAuth {
			fmt.Fprintf(e.Err, "Got error: %s", errorString(e, err))
		}
		fmt.Fprintf(e.Err, "%s\nPlease try again...\n", err)
		l.credentials.password = ""
	}
	return errAuth
}

func (l *login) helpCreateToken(e *env) (string, error) {
	for i := 0; i < 4; i++ {
		fmt.Fprintln(e.Out, "\nInput your account key or press enter to generate a new one.")
		fmt.Fprintf(e.Out, `Account Key: `)

		var token string
		fmt.Fscanf(e.In, "%s\n", &token)
		token = strings.TrimSpace(token)
		if token != "" {
			return token, nil
		}

		err := open.Run(keysURL)
		if err != nil {
			fmt.Fprintf(e.Err,
				`Sorry, we couldn’t open the browser for you.
Go here to generate an account key: %q
`,
				keysURL,
			)
		}
	}
	return "", stackerr.New("Account key cannot be empty. Please try again.")
}
