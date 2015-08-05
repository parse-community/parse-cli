package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/facebookgo/stackerr"
	"github.com/spf13/cobra"
)

const logFollowSleepDuration = time.Second

type parseTime struct {
	Type string `json:"__type"`
	ISO  string `json:"iso"`
}

type logResponse struct {
	Timestamp parseTime `json:"timestamp"`
	Message   string    `json:"message"`
}

type logsCmd struct {
	num    uint
	follow bool
	level  string
}

func (l *logsCmd) run(e *env, c *client) error {
	level := strings.ToUpper(l.level)
	if level != "INFO" && level != "ERROR" {
		return stackerr.Newf("invalid level: %q", l.level)
	}
	l.level = level
	numIsSet := true
	if l.num == 0 {
		numIsSet = false
		l.num = 10
	}

	lastTime, err := l.round(e, c, nil)
	if err != nil {
		return err
	}

	if !l.follow {
		return nil
	}
	if !numIsSet {
		l.num = 100 // force num to 100 for follow
	}

	ticker := e.Clock.Ticker(logFollowSleepDuration)
	defer ticker.Stop()
	for range ticker.C {
		lastTime, err = l.round(e, c, lastTime)
		if err != nil {
			return err
		}
	}
	return nil
}

func (l *logsCmd) round(e *env, c *client, startTime *parseTime) (*parseTime, error) {
	v := make(url.Values)
	v.Set("n", fmt.Sprint(l.num))
	v.Set("level", l.level)

	if startTime != nil {
		b, err := json.Marshal(startTime)
		if err != nil {
			return nil, stackerr.Wrap(err)
		}
		v.Set("startTime", string(b))
	}

	u := &url.URL{
		Path:     "scriptlog",
		RawQuery: v.Encode(),
	}
	var rows []logResponse
	if _, err := e.Client.Get(u, &rows); err != nil {
		return nil, stackerr.Wrap(err)
	}
	// logs come back in reverse
	for i := len(rows) - 1; i >= 0; i-- {
		fmt.Fprintln(e.Out, rows[i].Message)
	}
	if len(rows) > 0 {
		return &rows[0].Timestamp, nil
	}
	return startTime, nil
}

func newLogsCmd(e *env) *cobra.Command {
	l := logsCmd{
		level: "INFO",
	}
	cmd := &cobra.Command{
		Use:     "logs",
		Short:   "Prints out recent log messages",
		Long:    "Prints out recent log messages.",
		Run:     runWithClient(e, l.run),
		Aliases: []string{"log"},
	}
	cmd.Flags().UintVarP(&l.num, "num", "n", l.num,
		"The number of the messages to display")
	cmd.Flags().BoolVarP(&l.follow, "follow", "f", l.follow,
		"Emulates tail -f and streams new messages from the server")
	cmd.Flags().StringVarP(&l.level, "level", "l", l.level,
		"The log level to restrict to. Can be 'INFO' or 'ERROR'.")
	return cmd
}
