package parsecli

import (
	"fmt"
	"net/http"

	"github.com/facebookgo/parse"
)

type Context struct {
	Config    Config
	AppName   string
	AppConfig AppConfig
}

func newContext(e *Env, appName string) (*Context, error) {
	config, err := ConfigFromDir(e.Root)
	if err != nil {
		return nil, err
	}

	app, err := config.App(appName)
	if err != nil {
		return nil, err
	}

	masterKey, err := app.GetMasterKey(e)
	if err != nil {
		return nil, err
	}
	e.ParseAPIClient = e.ParseAPIClient.WithCredentials(
		parse.MasterKey{
			ApplicationID: app.GetApplicationID(),
			MasterKey:     masterKey,
		},
	)

	if e.HerokuAPIClient != nil {
		authToken, err := app.GetApplicationAuth(e)
		if err != nil {
			return nil, err
		}
		headers := make(http.Header)
		headers.Add("Authorization", fmt.Sprintf("Bearer %s", authToken))
		e.HerokuAPIClient.AdditionalHeaders = headers
		e.HerokuAPIClient.UserAgent = UserAgent
	}

	return &Context{
		AppName:   appName,
		AppConfig: app,
		Config:    config,
	}, nil
}
