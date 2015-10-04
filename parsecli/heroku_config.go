package parsecli

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"sort"

	"github.com/facebookgo/jsonpipe"
	"github.com/facebookgo/stackerr"
)

type HerokuAppConfig struct {
	ParseAppID        string `json:"parseAppId,omitempty"`
	MasterKey         string `json:"masterKey,omitempty"`
	HerokuAppID       string `json:"herokuAppId,omitempty"`
	HerokuAccessToken string `json:"herokuAccessToken,omitempty"`
	Link              string `json:"link,omitempty"`

	masterKey         string
	herokuAccessToken string
}

func (c *HerokuAppConfig) WithHiddenMasterKey(masterKey string) *HerokuAppConfig {
	c.masterKey = masterKey
	return c
}

func (c *HerokuAppConfig) WithHiddenAccessToken(accessToken string) *HerokuAppConfig {
	c.herokuAccessToken = accessToken
	return c
}

func (c *HerokuAppConfig) GetApplicationID() string {
	return c.ParseAppID
}

func (c *HerokuAppConfig) GetMasterKey(e *Env) (string, error) {
	if c.MasterKey != "" {
		return c.MasterKey, nil
	}
	if c.masterKey != "" {
		return c.masterKey, nil
	}
	app, err := FetchAppKeys(e, c.GetApplicationID())
	if err != nil {
		return "", err
	}
	c.masterKey = app.MasterKey
	return c.masterKey, nil
}

func (c *HerokuAppConfig) GetApplicationAuth(e *Env) (string, error) {
	if c.herokuAccessToken != "" {
		return c.herokuAccessToken, nil
	}

	var l Login
	_, err := l.AuthUserWithToken(e, true)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest(
		"POST",
		"herokuToken",
		ioutil.NopCloser(
			jsonpipe.Encode(
				map[string]string{
					"id": c.HerokuAppID,
				},
			),
		),
	)
	if err != nil {
		return "", stackerr.Wrap(err)
	}
	req.Header = make(http.Header)
	req.Header.Set("X-Parse-Application-Id", c.ParseAppID)
	req.Header.Set("X-Parse-Account-Key", l.Credentials.Token)

	resp, err := e.ParseAPIClient.RoundTrip(req)
	if err != nil {
		return "", stackerr.Wrap(err)
	}
	result := &struct {
		Token string `json:"token"`
	}{}
	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return "", stackerr.Wrap(err)
	}
	c.herokuAccessToken = result.Token
	return result.Token, nil
}

func (c *HerokuAppConfig) GetLink() string {
	return c.Link
}

type HerokuConfig struct {
	Applications  map[string]*HerokuAppConfig `json:"applications,omityempty"`
	ProjectConfig *ProjectConfig              `json:"-"`
}

func (c *HerokuConfig) AddAlias(name, link string) error {
	if _, found := c.Applications[name]; found {
		return stackerr.Newf("App %q has already been added.", name)
	}
	if _, found := c.Applications[link]; !found {
		return stackerr.Newf("App %q wasn't found.", link)
	}
	c.Applications[name] = &HerokuAppConfig{Link: link}
	return nil
}

func (c *HerokuConfig) SetDefaultApp(name string) error {
	delete(c.Applications, DefaultKey)
	return c.AddAlias(DefaultKey, name)
}

func (c *HerokuConfig) App(name string) (AppConfig, error) {
	ac, found := c.Applications[name]
	if !found {
		if name == DefaultKey {
			return nil, stackerr.Newf("No default app configured.")
		}
		return nil, stackerr.Newf("App %q wasn't found.", name)
	}
	if ac.Link != "" {
		return c.App(ac.Link)
	}
	return ac, nil
}

func (c *HerokuConfig) GetProjectConfig() *ProjectConfig {
	return c.ProjectConfig
}

func (c *HerokuConfig) GetDefaultApp() string {
	var defaultApp string
	if defaultKeyLink, ok := c.Applications[DefaultKey]; ok {
		defaultApp = defaultKeyLink.Link
	}
	return defaultApp
}

func (c *HerokuConfig) GetNumApps() int {
	return len(c.Applications)
}

var herokuAppNotFoundRegex = regexp.MustCompile("App not found")

func HerokuAppNotFound(err error) bool {
	return herokuAppNotFoundRegex.MatchString(err.Error())
}

func FetchHerokuAppName(id string, e *Env) (string, error) {
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("https://api.heroku.com/apps/%s", id),
		nil,
	)
	if err != nil {
		return "", stackerr.Wrap(err)
	}
	resp, err := e.ParseAPIClient.RoundTrip(req)
	if err != nil {
		return "", stackerr.Wrap(err)
	}
	app := struct {
		Name string `json:"name"`
	}{}
	if err := json.NewDecoder(resp.Body).Decode(&app); err != nil {
		return "", stackerr.Wrap(err)
	}
	return app.Name, nil
}

func (c *HerokuConfig) PrettyPrintApps(e *Env) {
	apps := c.Applications

	defaultApp := c.GetDefaultApp()

	var appNames []string
	for appName := range apps {
		appNames = append(appNames, appName)
	}
	sort.Strings(appNames)

	if len(appNames) == 0 {
		return
	}

	fmt.Fprintln(
		e.Out,
		"The following apps are associated with cloud code in the current directory:",
	)

	for _, appName := range appNames {
		if appName == DefaultKey {
			continue
		}
		if defaultApp == appName {
			fmt.Fprint(e.Out, "* ")
		} else {
			fmt.Fprint(e.Out, "  ")
		}
		fmt.Fprintf(e.Out, "%s", appName)

		config, _ := apps[appName]
		if config.GetLink() != "" {
			fmt.Fprintf(e.Out, " -> %s", config.GetLink())
		}
		herokuAppName, err := FetchHerokuAppName(config.HerokuAppID, e)
		if err != nil {
			if stackerr.HasUnderlying(err, stackerr.MatcherFunc(HerokuAppNotFound)) {
				herokuAppName = ""
			} else {
				herokuAppName = config.HerokuAppID
			}
		}
		fmt.Fprintf(e.Out, " (%q)\n", herokuAppName)
	}
}

func readHerokuConfigFile(path string) (*HerokuConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, stackerr.Wrap(err)
	}
	defer f.Close()
	var c HerokuConfig
	if err := json.NewDecoder(f).Decode(&c); err != nil {
		return nil, stackerr.Newf("Config file %q is not valid JSON.", path)
	}
	return &c, nil
}

func SetHerokuDefault(e *Env, newDefault, defaultApp string, herokuConfig *HerokuConfig) error {
	apps := herokuConfig.Applications
	if _, ok := apps[newDefault]; !ok {
		herokuConfig.PrettyPrintApps(e)
		return stackerr.Newf("Invalid application name \"%s\". Please select from the valid applications printed above.", newDefault)
	}

	if defaultApp == "" {
		apps[DefaultKey] = &HerokuAppConfig{Link: newDefault}
	} else {
		apps[DefaultKey].Link = newDefault
	}
	if err := StoreConfig(e, herokuConfig); err != nil {
		return err
	}
	fmt.Fprintf(e.Out, "Default app set to %s.\n", newDefault)
	return nil
}
