package main

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

type app struct {
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

type apps struct {
	login login
}

const numRetries = 3

func allApps(apps []*app) []string {
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

func (a *apps) printApp(e *env, params *app) error {
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

func getAuthHeaders(credentials *credentials, headers http.Header) http.Header {
	if headers == nil {
		headers = make(http.Header)
	}
	headers.Add("X-Parse-Email", credentials.email)
	if credentials.token != "" {
		headers.Add("X-Parse-Account-Key", credentials.token)
	} else {
		headers.Add("X-Parse-Password", credentials.password)
	}
	return headers
}

func fetchAppKeys(e *env, appID string) (*app, error) {
	l := &login{}
	err := l.authUser(e)
	if err != nil {
		return nil, err
	}
	credentials := &l.credentials

	req := &http.Request{
		Method: "GET",
		URL:    &url.URL{Path: path.Join("apps", appID)},
		Header: getAuthHeaders(credentials, nil),
	}
	res := &app{}

	if response, err := e.Client.Do(req, nil, res); err != nil {
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

func (a *apps) restFetchApps(e *env) ([]*app, error) {
	req := &http.Request{
		Method: "GET",
		URL:    &url.URL{Path: "apps"},
		Header: getAuthHeaders(&a.login.credentials, nil),
	}

	var res struct {
		Results []*app `json:"results"`
	}

	if response, err := e.Client.Do(req, nil, &res); err != nil {
		if response.StatusCode == http.StatusUnauthorized {
			return nil, errAuth
		}
		return nil, err
	}
	return res.Results, nil
}

func (a *apps) selectApp(apps []*app, msg string, e *env) (*app, error) {
	appNames := allApps(apps)
	appCount := len(apps)

	pos := -1
	var err error

	for {
		fmt.Fprintf(e.Out, "%s%s", selectionString(appNames), msg)

		var selected string
		fmt.Fscanf(e.In, "%s", &selected)
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

func (a *apps) showApps(e *env) error {
	apps, err := a.restFetchApps(e)
	if err != nil {
		return err
	}
	fmt.Fprintln(e.Out, "The following apps are currently owned by you:")
	app, err := a.selectApp(apps,
		"Select an app to view its properties: ",
		e,
	)
	if err != nil {
		return err
	}
	fmt.Fprintf(e.Out, "Properties of app: %q\n", app.Name)
	return a.printApp(e, app)
}

func (a *apps) getAppName(e *env) (string, error) {
	var appName string
	fmt.Fprint(e.Out, `Please choose a name for your Parse app.
Note that this name will appear on the Parse website,
but it does not have to be the same as your mobile app's public name.
Name: `)
	fmt.Fscanf(e.In, "%s\n", &appName)
	if appName == "" {
		return "", errors.New("App name cannot be empty")
	}
	return appName, nil
}

func (a *apps) restCreateApp(e *env, appName string) (*app, error) {
	req := &http.Request{
		Method: "POST",
		URL:    &url.URL{Path: "apps"},
		Header: make(http.Header),
		Body:   ioutil.NopCloser(jsonpipe.Encode(map[string]string{"appName": appName})),
	}

	req.Header.Add("X-Parse-Email", a.login.credentials.email)
	req.Header.Add("X-Parse-Password", a.login.credentials.password)

	var res app
	if response, err := e.Client.Do(req, nil, &res); err != nil {
		if response != nil && response.StatusCode == http.StatusUnauthorized {
			return nil, errAuth
		}
		return nil, err
	}

	return &res, nil
}

func (a *apps) createApp(e *env) (*app, error) {
	apps, err := a.restFetchApps(e)
	if err != nil {
		return nil, err
	}
	appNames := make(map[string]struct{})
	for _, app := range apps {
		appNames[app.Name] = struct{}{}
	}
	for i := 0; i < numRetries; i++ {
		appName, err := a.getAppName(e)
		if err != nil {
			return nil, err
		}
		if _, ok := appNames[appName]; ok {
			fmt.Fprintf(e.Out,
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
