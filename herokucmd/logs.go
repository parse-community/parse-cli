package herokucmd

import (
	"io"
	"net/http"

	"github.com/ParsePlatform/parse-cli/parsecli"
	"github.com/bgentry/heroku-go"
	"github.com/facebookgo/stackerr"
	"github.com/spf13/cobra"
)

type logsCmd struct {
	tail bool
	num  int
}

func (h *logsCmd) run(e *parsecli.Env, ctx *parsecli.Context) error {
	hkConfig, ok := ctx.AppConfig.(*parsecli.HerokuAppConfig)
	if !ok {
		return stackerr.New("Unexpected config format")
	}

	opts := &heroku.LogSessionCreateOpts{}
	if h.num == 0 {
		h.num = 50
	}
	//source := "app"
	//opts.Source = &source
	opts.Lines = &h.num
	opts.Tail = &h.tail

	session, err := e.HerokuAPIClient.LogSessionCreate(
		hkConfig.HerokuAppID,
		opts,
	)
	if err != nil {
		return stackerr.Wrap(err)
	}
	resp, err := http.Get(session.LogplexURL)
	if err != nil {
		return stackerr.Wrap(err)
	}
	_, err = io.Copy(e.Out, resp.Body)
	return stackerr.Wrap(err)
}

func NewLogsCmd(e *parsecli.Env) *cobra.Command {
	h := &logsCmd{num: 50}
	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Fetch logs from heroku",
		Long:  "Fetch logs from heroku.",
		Run:   parsecli.RunWithClient(e, h.run),
	}
	cmd.Flags().BoolVarP(&h.tail, "follow", "f", h.tail, "Tail logs from server.")
	cmd.Flags().IntVarP(&h.num, "num", "n", h.num, "Number of log lines to fetch.")
	return cmd
}
