package parsecli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/facebookgo/clock"
	"github.com/facebookgo/ensure"
	"github.com/facebookgo/parse"
	"github.com/facebookgo/testname"
)

const (
	Version        = "2.2.8"
	CloudDir       = "cloud"
	HostingDir     = "public"
	DefaultBaseURL = "https://api.parse.com/1/"
)

var UserAgent = fmt.Sprintf("parse-cli-%s-%s", runtime.GOOS, Version)

type Env struct {
	Root           string // project root
	Server         string // parse api server
	Type           int    // project type
	ParserEmail    string // email associated with developer parse account
	ErrorStack     bool
	Out            io.Writer
	Err            io.Writer
	In             io.Reader
	Exit           func(int)
	Clock          clock.Clock
	ParseAPIClient *ParseAPIClient
}

type Harness struct {
	T      testing.TB
	Out    bytes.Buffer
	Err    bytes.Buffer
	Clock  *clock.Mock
	Env    *Env
	remove []string
}

func (h *Harness) MakeEmptyRoot() {
	var err error
	prefix := fmt.Sprintf("%s-", testname.Get("parse-cli-"))
	h.Env.Root, err = ioutil.TempDir("", prefix)
	ensure.Nil(h.T, err)
	h.remove = append(h.remove, h.Env.Root)
}

func (h *Harness) MakeWithConfig(global string) {
	h.Env.Root = makeDirWithConfig(h.T, global)
}

func (h *Harness) Stop() {
	for _, p := range h.remove {
		os.RemoveAll(p)
	}
}

func NewHarness(t testing.TB) *Harness {
	te := Harness{
		T:     t,
		Clock: clock.NewMock(),
	}
	te.Env = &Env{
		Out:            &te.Out,
		Err:            &te.Err,
		Clock:          te.Clock,
		ParseAPIClient: &ParseAPIClient{APIClient: &parse.Client{}},
	}
	return &te
}

// makes a temp directory with the given global config.
func makeDirWithConfig(t testing.TB, global string) string {
	dir, err := ioutil.TempDir("", testname.Get("parse-cli-"))
	ensure.Nil(t, err)
	ensure.Nil(t, os.Mkdir(filepath.Join(dir, "config"), 0755))
	ensure.Nil(t, ioutil.WriteFile(
		filepath.Join(dir, LegacyConfigFile),
		[]byte(global),
		os.FileMode(0600),
	))
	return dir
}

type TransportFunc func(r *http.Request) (*http.Response, error)

func (t TransportFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return t(r)
}

func NewTokenHarness(t testing.TB) *Harness {
	h := NewHarness(t)
	ht := TransportFunc(func(r *http.Request) (*http.Response, error) {
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
	h.Env.ParseAPIClient = &ParseAPIClient{APIClient: &parse.Client{Transport: ht}}
	return h
}

func NewAppHarness(t testing.TB) (*Harness, []*App) {
	h := NewHarness(t)

	apps := []*App{
		newTestApp("A"),
		newTestApp("B"),
	}
	res := map[string][]*App{"results": apps}
	ht := TransportFunc(func(r *http.Request) (*http.Response, error) {

		email := r.Header.Get("X-Parse-Email")
		password := r.Header.Get("X-Parse-Password")
		token := r.Header.Get("X-Parse-Account-Key")
		if !((email == "email" && password == "password") || (token == "token")) {
			return &http.Response{
				StatusCode: http.StatusUnauthorized,
				Body:       ioutil.NopCloser(strings.NewReader(`{"error": "incorrect credentials"}`)),
			}, nil
		}

		switch r.URL.Path {
		case "/1/apps":
			if r.Method == "GET" {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       ioutil.NopCloser(strings.NewReader(jsonStr(t, &res))),
				}, nil

			}

			if r.Method != "POST" || r.Body == nil {
				return &http.Response{
					StatusCode: http.StatusNotFound,
				}, errors.New("unknown resource")
			}

			var params map[string]string
			if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
				}, err
			}
			details, err := json.Marshal(newTestApp(params["appName"]))
			if err != nil {
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
				}, err
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(bytes.NewReader(details)),
			}, nil

		case "/1/apps/an-app":
			ensure.DeepEqual(t, r.Method, "GET")
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(strings.NewReader(jsonStr(t, newTestApp("an-app")))),
			}, nil
		default:
			return &http.Response{
				StatusCode: http.StatusNotFound,
			}, nil
		}
	})

	h.Env.ParseAPIClient = &ParseAPIClient{APIClient: &parse.Client{Transport: ht}}
	return h, apps
}

func newTestApp(suffix string) *App {
	return &App{
		Name:                          suffix,
		DashboardURL:                  fmt.Sprintf("https://api.example.com/dashboard/%s", suffix),
		ApplicationID:                 fmt.Sprintf("applicationID.%s", suffix),
		ClientKey:                     fmt.Sprintf("clientKey.%s", suffix),
		JavaScriptKey:                 fmt.Sprintf("javaScriptKey.%s", suffix),
		WindowsKey:                    fmt.Sprintf("windowsKey.%s", suffix),
		WebhookKey:                    fmt.Sprintf("webhookKey.%s", suffix),
		RestKey:                       fmt.Sprintf("restKey.%s", suffix),
		MasterKey:                     fmt.Sprintf("masterKey.%s", suffix),
		ClientPushEnabled:             false,
		ClientClassCreationEnabled:    true,
		RequireRevocableSessions:      true,
		RevokeSessionOnPasswordChange: true,
	}
}

func jsonStr(t testing.TB, v interface{}) string {
	b, err := json.Marshal(v)
	ensure.Nil(t, err)
	return string(b)
}
