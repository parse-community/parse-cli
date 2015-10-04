package parsecmd

import (
	"fmt"
	"net/url"

	"github.com/ParsePlatform/parse-cli/parsecli"
	"github.com/facebookgo/stackerr"
	"github.com/spf13/cobra"
)

type rollbackCmd struct {
	ReleaseName string
}

type rollbackInfo struct {
	ReleaseName string `json:"releaseName,omitEmpty"`
}

func (r *rollbackCmd) run(e *parsecli.Env, c *parsecli.Context) error {
	var req rollbackInfo
	message := "previous release"
	if r.ReleaseName != "" {
		message = r.ReleaseName
		req.ReleaseName = r.ReleaseName
	}

	fmt.Fprintf(e.Out, "Rolling back to %s\n", message)

	var response rollbackInfo
	if _, err := e.ParseAPIClient.Post(&url.URL{Path: "deploy"},
		&req, &response); err != nil {
		return stackerr.Newf("Rollback failed with %s", stackerr.Wrap(err))
	}
	fmt.Fprintf(e.Out, "Rolled back to version %s\n", response.ReleaseName)
	return nil
}

func NewRollbackCmd(e *parsecli.Env) *cobra.Command {
	r := rollbackCmd{}
	cmd := &cobra.Command{
		Use:   "rollback [app]",
		Short: "Rolls back the version for the given app",
		Long:  "Rolls back the version for the given app.",
		Run:   parsecli.RunWithClient(e, r.run),
	}
	cmd.Flags().StringVarP(&r.ReleaseName, "release", "r", r.ReleaseName,
		"Provides an optional release to rollback to. If no release is provided, rolls back to the previous release.")
	return cmd
}
