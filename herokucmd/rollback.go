package herokucmd

import (
	"fmt"

	"github.com/ParsePlatform/parse-cli/parsecli"
	"github.com/bgentry/heroku-go"
	"github.com/facebookgo/stackerr"
	"github.com/spf13/cobra"
)

type rollbackCmd struct {
	ReleaseName string
}

func (r *rollbackCmd) getLastSecondVersion(e *parsecli.Env, name string) (int, error) {
	releases, err := e.HerokuAPIClient.ReleaseList(name, &heroku.ListRange{Field: "version", Max: 2, Descending: true})
	if err != nil {
		return 0, stackerr.Wrap(err)
	}
	return releases[1].Version, nil
}

func (r *rollbackCmd) run(e *parsecli.Env, c *parsecli.Context) error {
	appConfig, ok := c.AppConfig.(*parsecli.HerokuAppConfig)
	if !ok {
		return stackerr.New("Invalid Heroku app config")
	}
	releaseName := r.ReleaseName
	if releaseName == "" {
		lastSecond, err := r.getLastSecondVersion(e, appConfig.HerokuAppID)
		if err != nil {
			return err
		}
		releaseName = fmt.Sprintf("%d", lastSecond)
	}
	release, err := e.HerokuAPIClient.ReleaseRollback(appConfig.HerokuAppID, releaseName)
	if err != nil {
		return stackerr.Wrap(err)
	}
	fmt.Fprintf(
		e.Out,
		`%s
Current version: %d
`,
		release.Description,
		release.Version,
	)
	return nil
}

func NewRollbackCmd(e *parsecli.Env) *cobra.Command {
	r := rollbackCmd{}
	cmd := &cobra.Command{
		Use:   "rollback [app]",
		Short: "Rolls back the version for given app at Heroku",
		Long:  "Rolls back the version for given app at Heroku.",
		Run:   parsecli.RunWithClient(e, r.run),
	}
	cmd.Flags().StringVarP(&r.ReleaseName, "release", "r", r.ReleaseName, "The release to rollback to")
	return cmd
}
