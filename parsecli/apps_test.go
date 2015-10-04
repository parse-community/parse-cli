package parsecli

import (
	"errors"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/facebookgo/ensure"
	"github.com/facebookgo/parse"
)

var (
	defaultCredentials      = Credentials{Email: "email", Password: "password"}
	defaultTokenCredentials = Credentials{Email: "email", Token: "token"}
	defaultApps             = Apps{Login: Login{Credentials: defaultCredentials}}
	defaultAppsWithToken    = Apps{Login: Login{Credentials: defaultTokenCredentials}}
)

func TestFetchAppKeys(t *testing.T) {
	t.Parallel()

	h, _ := NewAppHarness(t)
	defer h.Stop()

	h.Env.In = ioutil.NopCloser(strings.NewReader("email\npassword\n"))
	app, err := FetchAppKeys(h.Env, "an-app")
	ensure.Nil(t, err)
	ensure.DeepEqual(t, app, newTestApp("an-app"))
}

func TestFetchApps(t *testing.T) {
	t.Parallel()

	h, apps := NewAppHarness(t)
	defer h.Stop()

	a := defaultApps
	gotApps, err := a.RestFetchApps(h.Env)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, gotApps, apps)

	a.Login.Credentials.Password = "invalid"
	_, err = a.RestFetchApps(h.Env)
	ensure.DeepEqual(t, err, errAuth)
}

func TestFetchAppsToken(t *testing.T) {
	t.Parallel()

	h, apps := NewAppHarness(t)
	defer h.Stop()

	a := defaultAppsWithToken
	gotApps, err := a.RestFetchApps(h.Env)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, gotApps, apps)

	a.Login.Credentials.Token = "invalid"
	_, err = a.RestFetchApps(h.Env)
	ensure.DeepEqual(t, err, errAuth)
}

func TestSelectAppWithRetries(t *testing.T) {
	t.Parallel()

	h, apps := NewAppHarness(t)

	defer h.Stop()

	a := defaultApps
	h.Env.In = ioutil.NopCloser(strings.NewReader("3\n1"))
	selectedApp, err := a.SelectApp(apps, "test: ", h.Env)
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
	h, apps := NewAppHarness(t)
	apps[0], apps[1] = apps[1], apps[0]

	defer h.Stop()

	a := defaultApps
	h.Env.In = ioutil.NopCloser(strings.NewReader("1"))
	selectedApp, err := a.SelectApp(apps, "test: ", h.Env)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, h.Out.String(),
		`1:	A
2:	B
test: `)
	ensure.DeepEqual(t, selectedApp, newTestApp("A"))
}

func TestGetAppName(t *testing.T) {
	t.Parallel()

	h := NewHarness(t)
	defer h.Stop()

	a := defaultApps
	h.Env.In = ioutil.NopCloser(strings.NewReader("\n"))
	appName, err := a.getAppName(h.Env)
	ensure.Err(t, err, regexp.MustCompile("App name cannot be empty"))

	h.Env.In = ioutil.NopCloser(strings.NewReader("hello\n"))
	appName, err = a.getAppName(h.Env)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, appName, "hello")
}

func TestCreateApp(t *testing.T) {
	t.Parallel()

	h, _ := NewAppHarness(t)
	defer h.Stop()

	a := defaultApps
	app, err := a.restCreateApp(h.Env, "C")

	ensure.Nil(t, err)
	ensure.DeepEqual(t, app, newTestApp("C"))
}

func TestCreateNewApp(t *testing.T) {
	t.Parallel()

	h, _ := NewAppHarness(t)
	defer h.Stop()

	a := defaultApps
	h.Env.In = ioutil.NopCloser(strings.NewReader("D"))
	app, err := a.CreateApp(h.Env, "", 0)
	ensure.Nil(t, err)

	ensure.Nil(t, a.PrintApp(h.Env, app))
	ensure.DeepEqual(t, h.Out.String(),
		`Please choose a name for your Parse app.
Note that this name will appear on the Parse website,
but it does not have to be the same as your mobile app's public name.
Name: Properties of the app "D":
Name				D
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

func TestCreateNewAppNameTaken(t *testing.T) {
	t.Parallel()

	h, _ := NewAppHarness(t)
	defer h.Stop()

	a := defaultApps

	h.Env.In = ioutil.NopCloser(strings.NewReader("A\nD"))
	_, err := a.CreateApp(h.Env, "", 0)
	ensure.Nil(t, err)
	ensure.StringContains(t, h.Err.String(), "already created an app")
}

func TestNoPanicAppCreate(t *testing.T) {
	t.Parallel()
	h := NewHarness(t)
	defer h.Stop()

	ht := TransportFunc(func(r *http.Request) (*http.Response, error) {
		ensure.DeepEqual(t, r.URL.Path, "/1/apps")
		return nil, errors.New("nil response panic")
	})
	h.Env.ParseAPIClient = &ParseAPIClient{APIClient: &parse.Client{Transport: ht}}

	var apps Apps
	app, err := apps.restCreateApp(h.Env, "panic!")
	ensure.True(t, app == nil)
	ensure.Err(t, err, regexp.MustCompile("nil response panic"))
}
