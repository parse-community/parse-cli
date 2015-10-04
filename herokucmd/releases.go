package herokucmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/ParsePlatform/parse-cli/parsecli"
	"github.com/bgentry/heroku-go"
	"github.com/facebookgo/stackerr"
	"github.com/spf13/cobra"
)

type releasesCmd struct{}

func (r *releasesCmd) printHerokuReleases(releases []heroku.Release, e *parsecli.Env) {
	w := new(tabwriter.Writer)
	w.Init(e.Out, 20, 4, 2, ' ', 0)
	fmt.Fprintln(w, "Version\tUpdate At\tDescription")
	for _, release := range releases {
		fmt.Fprintf(
			w,
			"%d\t%s\tDeployed by: %q. %s\n\n",
			release.Version,
			release.UpdatedAt,
			release.User.Email,
			release.Description,
		)
	}
	w.Flush()
}

func (r *releasesCmd) run(e *parsecli.Env, c *parsecli.Context) error {
	appConfig, ok := c.AppConfig.(*parsecli.HerokuAppConfig)
	if !ok {
		return stackerr.New("Invalid app config.")
	}
	lr := &heroku.ListRange{Field: "version", Max: 10, Descending: true}
	releases, err := e.HerokuAPIClient.ReleaseList(appConfig.HerokuAppID, lr)
	if err != nil {
		return stackerr.Wrap(err)
	}
	r.printHerokuReleases(releases, e)
	return nil
}

func NewReleasesCmd(e *parsecli.Env) *cobra.Command {
	r := &releasesCmd{}
	cmd := &cobra.Command{
		Use:   "releases [app]",
		Short: "Gets the Heroku releases for a Parse app",
		Long:  "Prints the last 10 releases made on Heroku",
		Run:   parsecli.RunWithClient(e, r.run),
	}
	return cmd
}
