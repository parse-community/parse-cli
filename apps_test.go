package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/facebookgo/ensure"
	"github.com/facebookgo/parse"
)

var (
	defaultCredentials      = credentials{email: "email", password: "password"}
	defaultTokenCredentials = credentials{email: "email", token: "token"}
	defaultApps             = apps{login: login{credentials: defaultCredentials}}
	defaultAppsWithToken    = apps{login: login{credentials: defaultTokenCredentials}}
)

func newTestApp(suffix string) *app {
	return &app{
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

func newAppHarness(t testing.TB) (*Harness, []*app) {
	h := newHarness(t)

	apps := []*app{
		newTestApp("A"),
		newTestApp("B"),
	}
	res := map[string][]*app{"results": apps}
	ht := transportFunc(func(r *http.Request) (*http.Response, error) {

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

	h.env.ParseAPIClient = &ParseAPIClient{apiClient: &parse.Client{Transport: ht}}
	return h, apps
}

func TestFetchAppKeys(t *testing.T) {
	t.Parallel()

	h, _ := newAppHarness(t)
	defer h.Stop()

	h.env.In = ioutil.NopCloser(strings.NewReader("email\npassword\n"))
	app, err := fetchAppKeys(h.env, "an-app")
	ensure.Nil(t, err)
	ensure.DeepEqual(t, app, newTestApp("an-app"))
}

func TestFetchApps(t *testing.T) {
	t.Parallel()

	h, apps := newAppHarness(t)
	defer h.Stop()

	a := defaultApps
	gotApps, err := a.restFetchApps(h.env)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, gotApps, apps)

	a.login.credentials.password = "invalid"
	_, err = a.restFetchApps(h.env)
	ensure.DeepEqual(t, err, errAuth)
}

func TestFetchAppsToken(t *testing.T) {
	t.Parallel()

	h, apps := newAppHarness(t)
	defer h.Stop()

	a := defaultAppsWithToken
	gotApps, err := a.restFetchApps(h.env)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, gotApps, apps)

	a.login.credentials.token = "invalid"
	_, err = a.restFetchApps(h.env)
	ensure.DeepEqual(t, err, errAuth)
}

func TestSelectAppWithRetries(t *testing.T) {
	t.Parallel()

	h, apps := newAppHarness(t)

	defer h.Stop()

	a := defaultApps
	h.env.In = ioutil.NopCloser(strings.NewReader("3\n1"))
	selectedApp, err := a.selectApp(apps, "test: ", h.env)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, h.Out.String(),
		`1:	A
2:	B
test: Invalid selection 3: must be between 1 and 2
1:	A
2:	B
test: `)
	ensure.DeepEqual(t, selectedApp, newTestApp("A"))
}

func TestSelectApp(t *testing.T) {
	t.Parallel()

	// apps = [B, A] & appNames=[A,B]
	h, apps := newAppHarness(t)
	apps[0], apps[1] = apps[1], apps[0]

	defer h.Stop()

	a := defaultApps
	h.env.In = ioutil.NopCloser(strings.NewReader("1"))
	selectedApp, err := a.selectApp(apps, "test: ", h.env)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, h.Out.String(),
		`1:	A
2:	B
test: `)
	ensure.DeepEqual(t, selectedApp, newTestApp("A"))
}

func TestGetAppName(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	defer h.Stop()

	a := defaultApps
	h.env.In = ioutil.NopCloser(strings.NewReader("\n"))
	appName, err := a.getAppName(h.env)
	ensure.Err(t, err, regexp.MustCompile("App name cannot be empty"))

	h.env.In = ioutil.NopCloser(strings.NewReader("hello\n"))
	appName, err = a.getAppName(h.env)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, appName, "hello")
}

func TestCreateApp(t *testing.T) {
	t.Parallel()

	h, _ := newAppHarness(t)
	defer h.Stop()

	a := defaultApps
	app, err := a.restCreateApp(h.env, "C")

	ensure.Nil(t, err)
	ensure.DeepEqual(t, app, newTestApp("C"))
}

func TestCreateNewApp(t *testing.T) {
	t.Parallel()

	h, _ := newAppHarness(t)
	defer h.Stop()

	a := defaultApps
	h.env.In = ioutil.NopCloser(strings.NewReader("D"))
	app, err := a.createApp(h.env, "")
	ensure.Nil(t, err)

	ensure.Nil(t, a.printApp(h.env, app))
	ensure.DeepEqual(t, h.Out.String(),
		`Please choose a name for your Parse app.
Note that this name will appear on the Parse website,
but it does not have to be the same as your mobile app's public name.
Name: Name				D
DashboardURL			https://api.example.com/dashboard/D
ApplicationID			applicationID.D
ClientKey			clientKey.D
JavaScriptKey			javaScriptKey.D
WindowsKey			windowsKey.D
WebhookKey			webhookKey.D
RestKey				restKey.D
MasterKey			masterKey.D
ClientPushEnabled		false
ClientClassCreationEnabled	true
RequireRevocableSessions	true
RevokeSessionOnPasswordChange	true
`)
}

func TestNoPanicAppCreate(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	defer h.Stop()

	ht := transportFunc(func(r *http.Request) (*http.Response, error) {
		ensure.DeepEqual(t, r.URL.Path, "/1/apps")
		return nil, errors.New("nil response panic")
	})
	h.env.ParseAPIClient = &ParseAPIClient{apiClient: &parse.Client{Transport: ht}}

	var apps apps
	app, err := apps.restCreateApp(h.env, "panic!")
	ensure.True(t, app == nil)
	ensure.Err(t, err, regexp.MustCompile("nil response panic"))
}
