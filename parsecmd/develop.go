package parsecmd

import (
	"fmt"
	"net"
	"time"

	"github.com/ParsePlatform/parse-cli/parsecli"
	"github.com/spf13/cobra"
)

const (
	maxLogRetries        = 50
	developFollowNumLogs = 25
	networkErrorsWait    = 20
	otherErrorsWait      = 10
)

type developCmd struct {
	deployInterval time.Duration // The number of seconds between deploy
	mustFetch      bool          // If set, prevDeployInfo will always be fetched from server
	Verbose        bool          // If set, will print details about deploy in addition to server logs
}

type deployFunc func(parseVersion string,
	prevDeplInfo *deployInfo,
	forDevelop bool,
	e *parsecli.Env) (*deployInfo, error)

func (d *developCmd) contDeploy(e *parsecli.Env, deployer deployFunc, first, done chan struct{}) {
	if d.deployInterval == 0 {
		d.deployInterval = time.Second
	}

	var prevDeplInfo *deployInfo
	ticker := e.Clock.Ticker(d.deployInterval)
	defer ticker.Stop()

	latestError := false
	for {
		select {
		case <-ticker.C:
		case <-done:
			return
		}
		config, err := parsecli.ConfigFromDir(e.Root)
		if err != nil {
			if !latestError {
				latestError = true
			}
			if latestError {
				fmt.Fprintf(
					e.Err,
					`Config malformed.
Please fix your config file in %s and try again.
`,
					parsecli.GetConfigFile(e),
				)
			}
			if first != nil { // first deploy aborted
				close(first)
				first = nil
			}
			continue
		}
		latestError = false
		newDeplInfo, _ := deployer(config.GetProjectConfig().Parse.JSSDK, prevDeplInfo, true, e)
		if !d.mustFetch {
			prevDeplInfo = newDeplInfo
		}
		if first != nil { // first deploy finished
			close(first)
			first = nil
		}
	}
}

func (d *developCmd) handleError(e *parsecli.Env, err error, sleep func(time.Duration)) error {
	if err == nil {
		return nil
	}

	if _, ok := err.(*net.OpError); ok {
		fmt.Fprintf(
			e.Err,
			"Flaky network. Waiting %ds before trying to fetch logs again.",
			networkErrorsWait,
		)
		sleep(networkErrorsWait * time.Second)
	} else {
		sleep(otherErrorsWait * time.Second)
	}
	return err
}

func (d *developCmd) run(e *parsecli.Env, c *parsecli.Context) error {
	first := make(chan struct{})
	go d.contDeploy(e,
		deployFunc((&deployCmd{Verbose: d.Verbose}).deploy),
		first,
		make(chan struct{}))
	<-first

	var err error
	for i := 0; i < maxLogRetries; i++ {
		l := &logsCmd{num: 1, level: "INFO"}

		// we want to fetch only the latest log line after first deploy
		// there after num is reset to 25 in log tailer for efficiency
		err = d.handleError(e, l.run(e, c), e.Clock.Sleep)
		if err != nil {
			continue
		}

		l.num = developFollowNumLogs
		l.follow = true
		err = d.handleError(e, l.run(e, c), e.Clock.Sleep)
		if err == nil {
			return nil
		}
	}
	return err
}

func NewDevelopCmd(e *parsecli.Env) *cobra.Command {
	d := &developCmd{deployInterval: time.Second}
	cmd := &cobra.Command{
		Use:   "develop app",
		Short: "Monitors for changes to code and deploys, also tails parse logs",
		Long: `Monitors for changes to source files and uploads updated files to Parse. ` +
			`This will also monitor the parse INFO log for any new log messages and write ` +
			`out updates to the terminal. This requires an app to be provided, to ` +
			`avoid running develop on production apps accidently.`,
		Run: parsecli.RunWithAppClient(e, d.run),
	}
	cmd.Flags().DurationVarP(&d.deployInterval, "interval", "i", d.deployInterval, "Number of seconds between deploys.")
	cmd.Flags().BoolVarP(&d.mustFetch, "fetch", "f", d.mustFetch, "Always fetch previous deployment info from server")
	cmd.Flags().BoolVarP(&d.Verbose, "verbose", "v", d.Verbose, "Control verbosity of cmd line logs")

	return cmd
}
