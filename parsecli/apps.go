package parsecli

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/facebookgo/jsonpipe"
	"github.com/facebookgo/stackerr"
)

type App struct {
	Name                          string `json:"appName"`
	DashboardURL                  string `json:"dashboardURL"`
	ApplicationID                 string `json:"applicationId"`
	ClientKey                     string `json:"clientKey"`
	JavaScriptKey                 string `json:"javascriptKey"`
	WindowsKey                    string `json:"windowsKey"`
	WebhookKey                    string `json:"webhookKey"`
	RestKey                       string `json:"restKey"`
	MasterKey                     string `json:"masterKey"`
	ClientPushEnabled             bool   `json:"clientPushEnabled"`
	ClientClassCreationEnabled    bool   `json:"clientClassCreationEnabled"`
	RequireRevocableSessions      bool   `json:"requireRevocableSessions"`
	RevokeSessionOnPasswordChange bool   `json:"revokeSessionOnPasswordChange"`
}
type Apps struct {
	Login Login
}

const numRetries = 3

func allApps(apps []*App) []string {
	var appNames []string
	for _, app := range apps {
		appNames = append(appNames, app.Name)
	}
	sort.Strings(appNames)
	return appNames
}

func selectionString(appNames []string) string {
	var buffer bytes.Buffer
	for i, appName := range appNames {
		buffer.WriteString(fmt.Sprintf("%d:\t%s\n", i+1, appName))
	}
	return buffer.String()
}

func (a *Apps) PrintApp(e *Env, params *App) error {
	fmt.Fprintf(e.Out, "Properties of the app %q:\n", params.Name)

	w := new(tabwriter.Writer)
	w.Init(e.Out, 0, 8, 0, '\t', 0)
	fmt.Fprintf(w, "Name\t%s\n", params.Name)
	fmt.Fprintf(w, "DashboardURL\t%s\n", params.DashboardURL)
	fmt.Fprintf(w, "ApplicationID\t%s\n", params.ApplicationID)
	fmt.Fprintf(w, "ClientKey\t%s\n", params.ClientKey)
	fmt.Fprintf(w, "JavaScriptKey\t%s\n", params.JavaScriptKey)
	fmt.Fprintf(w, "WindowsKey\t%s\n", params.WindowsKey)
	fmt.Fprintf(w, "WebhookKey\t%s\n", params.WebhookKey)
	fmt.Fprintf(w, "RestKey\t%s\n", params.RestKey)
	fmt.Fprintf(w, "MasterKey\t%s\n", params.MasterKey)
	fmt.Fprintf(w, "ClientPushEnabled\t%t\n", params.ClientPushEnabled)
	fmt.Fprintf(w, "ClientClassCreationEnabled\t%t\n", params.ClientClassCreationEnabled)
	fmt.Fprintf(w, "RequireRevocableSessions\t%t\n", params.RequireRevocableSessions)
	fmt.Fprintf(w, "RevokeSessionOnPasswordChange\t%t\n", params.RevokeSessionOnPasswordChange)

	return stackerr.Wrap(w.Flush())
}

func getAuthHeaders(credentials *Credentials, headers http.Header) http.Header {
	if headers == nil {
		headers = make(http.Header)
	}
	if credentials.Token != "" {
		headers.Add("X-Parse-Account-Key", credentials.Token)
	} else {
		headers.Add("X-Parse-Email", credentials.Email)
		headers.Add("X-Parse-Password", credentials.Password)
	}
	return headers
}

func FetchAppKeys(e *Env, appID string) (*App, error) {
	l := &Login{}
	err := l.AuthUser(e, false)
	if err != nil {
		return nil, err
	}
	credentials := &l.Credentials

	req := &http.Request{
		Method: "GET",
		URL:    &url.URL{Path: path.Join("apps", appID)},
		Header: getAuthHeaders(credentials, nil),
	}
	res := &App{}

	if response, err := e.ParseAPIClient.Do(req, nil, res); err != nil {
		if response.StatusCode == http.StatusUnauthorized {
			return nil, errAuth
		}
		return nil, err
	}
	if res == nil {
		return nil, stackerr.Newf("Unable to fetch keys for %s.", appID)
	}
	return res, nil
}

func (a *Apps) RestFetchApps(e *Env) ([]*App, error) {
	req := &http.Request{
		Method: "GET",
		URL:    &url.URL{Path: "apps"},
		Header: getAuthHeaders(&a.Login.Credentials, nil),
	}

	var res struct {
		Results []*App `json:"results"`
	}

	if response, err := e.ParseAPIClient.Do(req, nil, &res); err != nil {
		if response.StatusCode == http.StatusUnauthorized {
			return nil, errAuth
		}
		return nil, err
	}
	return res.Results, nil
}

func (a *Apps) SelectApp(apps []*App, msg string, e *Env) (*App, error) {
	appNames := allApps(apps)
	appCount := len(apps)

	pos := -1
	var err error

	for {
		fmt.Fprintf(e.Out, "%s%s", selectionString(appNames), msg)

		var selected string
		fmt.Fscanf(e.In, "%s\n", &selected)
		if pos, err = strconv.Atoi(strings.TrimSpace(selected)); err != nil {
			pos = -1
			break
		}

		if pos > 0 && pos <= appCount {
			break
		} else {
			fmt.Fprintf(e.Out, "Invalid selection %d: must be between 1 and %d\n", pos, appCount)
			pos = -1
		}
	}

	if pos != -1 {
		appName := appNames[pos-1]
		for _, app := range apps {
			if app.Name == appName {
				return app, nil
			}
		}
	}
	return nil, stackerr.Newf("Please try again. Please select from among the listed apps.")
}

func (a *Apps) ShowApps(e *Env, appName string) error {
	apps, err := a.RestFetchApps(e)
	if err != nil {
		return err
	}
	if appName != "" {
		for _, app := range apps {
			if app.Name == appName {
				return a.PrintApp(e, app)
			}
		}
	}

	fmt.Fprintf(
		e.Out,
		"These are the apps you currently have access to:\n%s",
		selectionString(allApps(apps)),
	)
	return nil
}

func (a *Apps) getAppName(e *Env) (string, error) {
	var appName string
	fmt.Fprint(e.Out, `Please choose a name for your Parse app.
Note that this name will appear on the Back4App website,
but it does not have to be the same as your mobile app's public name.
Name: `)
	fmt.Fscanf(e.In, "%s\n", &appName)
	if appName == "" {
		return "", errors.New("App name cannot be empty")
	}
	return appName, nil
}

func (a *Apps) restCreateApp(e *Env, appName string) (*App, error) {
	req := &http.Request{
		Method: "POST",
		URL:    &url.URL{Path: "apps"},
		Header: getAuthHeaders(&a.Login.Credentials, nil),
		Body:   ioutil.NopCloser(jsonpipe.Encode(map[string]string{"appName": appName})),
	}

	var res App
	if response, err := e.ParseAPIClient.Do(req, nil, &res); err != nil {
		if response != nil && response.StatusCode == http.StatusUnauthorized {
			return nil, errAuth
		}
		return nil, err
	}

	return &res, nil
}

func (a *Apps) CreateApp(e *Env, givenName string, retries int) (*App, error) {
	apps, err := a.RestFetchApps(e)
	if err != nil {
		return nil, err
	}
	appNames := make(map[string]struct{})
	for _, app := range apps {
		appNames[app.Name] = struct{}{}
	}
	if retries <= 0 {
		retries = numRetries
	}
	for i := 0; i < retries; i++ {
		var appName string
		if givenName != "" {
			appName = givenName
		} else {
			appName, err = a.getAppName(e)
			if err != nil {
				return nil, err
			}
		}

		if _, ok := appNames[appName]; ok {
			fmt.Fprintf(e.Err,
				`Hey! you already created an app named %q.
The apps associated with your account:
%s
Please try creating an app with a different name next time.
`,
				appName,
				selectionString(allApps(apps)),
			)
			continue
		}

		app, err := a.restCreateApp(e, appName)
		if err != nil {
			return nil, err
		}
		return app, nil
	}
	return nil, errors.New("could not create new app")
}
