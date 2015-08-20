package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/bgentry/go-netrc/netrc"
	"github.com/bgentry/speakeasy"
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

	parseNetrc = filepath.Join(".parse", "netrc")
)

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

func (l *login) getTokenCredentials(e *env) (*credentials, error) {
	reader, err := l.getTokensReader()
	if err != nil {
		return nil, stackerr.Wrap(err)
	}
	tokens, err := netrc.Parse(reader)
	if err != nil {
		return nil, stackerr.Wrap(err)
	}
	server, err := getHostFromURL(e.Server)
	if err != nil {
		return nil, err
	}
	machine := tokens.FindMachine(server)
	if machine == nil {
		return nil, stackerr.Newf("could not find token for %s", server)
	}
	return &credentials{
		token: machine.Password,
	}, nil
}

func (l *login) updatedNetrcContent(
	e *env,
	content io.Reader,
	credentials *credentials,
) ([]byte, error) {
	tokens, err := netrc.Parse(content)
	if err != nil {
		return nil, stackerr.Wrap(err)
	}

	server, err := getHostFromURL(e.Server)
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

func (l *login) storeCredentials(e *env, credentials *credentials) error {
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

	content, err := l.updatedNetrcContent(e, file, credentials)

	file, err = os.OpenFile(location, os.O_WRONLY|os.O_TRUNC, 0600)
	_, err = file.Write(content)
	return stackerr.Wrap(err)
}

func (l *login) authUserWithToken(e *env) error {
	tokenCredentials, err := l.getTokenCredentials(e)

	if err != nil {
		return err
	}

	apps := &apps{login: login{credentials: *tokenCredentials}}
	_, err = apps.restFetchApps(e)
	if err == errAuth {
		fmt.Fprintln(e.Err,
			`Sorry, the token you configured is not valid.

`)
	}
	if err != nil {
		return stackerr.Wrap(err)
	}

	l.credentials = *tokenCredentials
	return nil
}

func (l *login) authUser(e *env) error {
	if l.authUserWithToken(e) == nil {
		return nil
	}

	apps := &apps{}
	fmt.Fprintln(
		e.Out,
		`To avoid typing the email and password everytime,
please type "parse configure token" and provide a valid access token.

Please log in to Parse using your email and password.`,
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

func (l *login) helpCreateToken(e *env) {
	var shouldOpen string
	fmt.Fscanf(e.In, "%s\n", &shouldOpen)
	if shouldOpen == "n" {
		fmt.Fprintf(e.Out,
			`Please open %q in the browser
and follow instructions to create an access token.
`,
			keysURL,
		)
		return
	}
	err := open.Run(keysURL)
	if err != nil {
		fmt.Fprintf(e.Err,
			`Sorry, we could not open %q in the browser.
Please open %q in the browser to create a new account key.
`,
			keysURL,
			keysURL,
		)
	}
}
