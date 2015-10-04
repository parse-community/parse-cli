package herokucmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ParsePlatform/parse-cli/parsecli"
	"github.com/ParsePlatform/parse-cli/webhooks"
	"github.com/bgentry/heroku-go"
	"github.com/facebookgo/stackerr"
	"github.com/spf13/cobra"
)

type deployCmd struct {
	Force       bool
	Description string
}

func fetchHerokuAppInfo(e *parsecli.Env, appConfig *parsecli.HerokuAppConfig) (*heroku.App, error) {
	herokuAppInfo, err := e.HerokuAPIClient.AppInfo(appConfig.HerokuAppID)
	if err != nil {
		return nil, stackerr.Newf(
			"Unable to fetch app info for: %q from heroku.",
			appConfig.HerokuAppID,
		)
	}
	return herokuAppInfo, nil
}

func setHerokuConfigs(
	e *parsecli.Env,
	appConfig *parsecli.HerokuAppConfig,
	options map[string]*string,
) error {
	varsSet, err := e.HerokuAPIClient.ConfigVarUpdate(
		appConfig.HerokuAppID,
		options,
	)
	if err != nil {
		stackerr.New("Unable to set config vars to heroku.")
	}
	fmt.Fprintln(e.Out, "Successfully set the following vars to heroku:")
	for key, val := range varsSet {
		if key != "HOOKS_URL" {
			val = parsecli.Last4(val)
		}
		fmt.Fprintln(e.Out, key, val)
	}
	return nil
}

func (d *deployCmd) deploy(e *parsecli.Env, ctx *parsecli.Context) error {
	appConfig, ok := ctx.AppConfig.(*parsecli.HerokuAppConfig)
	if !ok {
		return stackerr.New("Invalid config type.")
	}

	authToken, err := ctx.AppConfig.GetApplicationAuth(e)
	if err != nil {
		return err
	}
	appInfo, err := fetchHerokuAppInfo(e, appConfig)
	if err != nil {
		return err
	}

	// set appropriate heroku config vars
	appKeys, err := parsecli.FetchAppKeys(e, appConfig.ParseAppID)
	if err != nil {
		return err
	}
	fmt.Fprintln(e.Out, "This Node.js webhooks project will be deployed to Heroku.")
	err = setHerokuConfigs(e,
		appConfig,
		map[string]*string{
			"HOOKS_URL":         &appInfo.WebURL,
			"PARSE_APP_ID":      &appConfig.ParseAppID,
			"PARSE_MASTER_KEY":  &appKeys.MasterKey,
			"PARSE_WEBHOOK_KEY": &appKeys.WebhookKey,
		},
	)
	if err != nil {
		return err
	}

	// push to heroku git
	g := gitInfo{
		repo:        e.Root,
		authToken:   authToken,
		remote:      fmt.Sprintf("git.heroku.com/%s.git", appInfo.Name),
		description: d.Description,
	}
	err = g.whichGit()
	if err != nil {
		return err
	}
	err = g.isGitRepo(e)
	if err != nil {
		err = g.init(e)
		if err != nil {
			return err
		}
	}
	dirty, err := g.isDirty(e)
	if err != nil {
		return err
	}

	if dirty {
		err := g.commit(e)
		if err != nil {
			return err
		}
	}
	if err := g.push(e, d.Force); err != nil {
		return err
	}

	webhooksConfig := filepath.Join(e.Root, "webhooks.json")
	if _, err := os.Lstat(webhooksConfig); err == nil {
		h := &webhooks.Hooks{BaseURL: appInfo.WebURL}
		return h.HooksCmd(e, ctx, []string{webhooksConfig})
	}

	return nil
}

func NewDeployCmd(e *parsecli.Env) *cobra.Command {
	var d deployCmd
	cmd := &cobra.Command{
		Use:   "deploy [app]",
		Short: "Deploys server code to Heroku",
		Long:  "Deploys server code to Heroku.",
		Run:   parsecli.RunWithClient(e, d.deploy),
	}
	cmd.Flags().BoolVarP(&d.Force, "force", "f", d.Force,
		"Forcefully deploy current contents even if they differ hugely from existing deploy.")
	cmd.Flags().StringVarP(&d.Description, "description", "d", d.Description,
		"Description is used as git commit message, if repo is dirty, before pushing to Heroku remote.")
	return cmd
}
