package herokucmd

import (
	"fmt"

	"github.com/ParsePlatform/parse-cli/parsecli"
	"github.com/facebookgo/stackerr"
	"github.com/spf13/cobra"
)

type downloadCmd struct{}

func (h *downloadCmd) run(e *parsecli.Env, ctx *parsecli.Context) error {
	appConfig, ok := ctx.AppConfig.(*parsecli.HerokuAppConfig)
	if !ok {
		return stackerr.New("expected heroku project config type")
	}

	authToken, err := appConfig.GetApplicationAuth(e)
	if err != nil {
		return err
	}

	herokuAppName, err := parsecli.FetchHerokuAppName(appConfig.HerokuAppID, e)
	if err != nil {
		return err
	}
	gitURL := fmt.Sprintf("https://:%s@git.heroku.com/%s.git", authToken, herokuAppName)
	return (&gitInfo{}).pull(e, gitURL)
}

func NewDownloadCmd(e *parsecli.Env) *cobra.Command {
	d := &downloadCmd{}
	cmd := &cobra.Command{
		Use:   "download",
		Short: "Downloads the latest deployed server code from Heroku servers",
		Long:  "Downloads the latest deployed server code from Heroku servers.",
		Run:   parsecli.RunWithClient(e, d.run),
	}
	return cmd
}
