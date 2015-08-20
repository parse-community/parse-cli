package main

import "github.com/facebookgo/parse"

type context struct {
	Config    config
	AppName   string
	AppConfig appConfig
}

func newContext(e *env, appName string) (*context, error) {
	config, err := configFromDir(e.Root)
	if err != nil {
		return nil, err
	}

	app, err := config.app(appName)
	if err != nil {
		return nil, err
	}

	masterKey, err := app.getMasterKey(e)
	if err != nil {
		return nil, err
	}
	e.ParseAPIClient = e.ParseAPIClient.WithCredentials(
		parse.MasterKey{
			ApplicationID: app.getApplicationID(),
			MasterKey:     masterKey,
		},
	)

	return &context{
		AppName:   appName,
		AppConfig: app,
		Config:    config,
	}, nil
}
