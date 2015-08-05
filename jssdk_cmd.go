package main

import (
	"fmt"
	"net/url"
	"sort"

	"github.com/facebookgo/stackerr"
	"github.com/spf13/cobra"
)

type jsSDKVersion struct {
	JS []string `json:"js"`
}

type jsSDKCmd struct {
	all        bool
	newVersion string
}

func (j *jsSDKCmd) getAllJSSdks(e *env) ([]string, error) {
	u := &url.URL{
		Path: "jsVersions",
	}
	var response jsSDKVersion
	if _, err := e.Client.Get(u, &response); err != nil {
		return nil, stackerr.Wrap(err)
	}
	sort.Sort(naturalStrings(response.JS))
	return response.JS, nil
}

func (j *jsSDKCmd) printVersions(e *env, c *client) error {
	allVersions, err := j.getAllJSSdks(e)
	if err != nil {
		return err
	}

	currentVersion := c.Config.getProjectConfig().Parse.JSSDK
	for _, version := range allVersions {
		prefix := "  "
		if currentVersion == version {
			prefix = "* "
		}
		fmt.Fprintf(e.Out, "%s %s\n", prefix, version)
	}
	return nil
}

func (j *jsSDKCmd) getVersion(e *env, c *client) error {
	if c.Config.getProjectConfig().Parse.JSSDK == "" {
		return stackerr.New("JavaScript SDK version not set for this project.")
	}
	fmt.Fprintf(
		e.Out,
		"Current JavaScript SDK version is %s\n",
		c.Config.getProjectConfig().Parse.JSSDK,
	)
	return nil
}

func (j *jsSDKCmd) setVersion(e *env, c *client) error {
	allVersions, err := j.getAllJSSdks(e)
	if err != nil {
		return err
	}
	valid := false
	for _, version := range allVersions {
		if version == j.newVersion {
			valid = true
		}
	}
	if !valid {
		return stackerr.New("Invalid SDK version selected.")
	}

	conf, err := configFromDir(e.Root)
	if err != nil {
		return err
	}
	conf.getProjectConfig().Parse.JSSDK = j.newVersion
	if err := storeProjectConfig(e, conf); err != nil {
		return err
	}

	fmt.Fprintf(e.Out, "Current JavaScript SDK version is %s\n", conf.getProjectConfig().Parse.JSSDK)
	return nil
}

// useLatestJSSDK is a utility method used by deploy & develop
// to write set jssdk version to latest available, if none set
func useLatestJSSDK(e *env) error {
	var j jsSDKCmd
	versions, err := j.getAllJSSdks(e)
	if err != nil {
		return err
	}

	if len(versions) == 0 {
		return stackerr.New("No JavaScript SDK version is available.")
	}

	config, err := configFromDir(e.Root)
	if err != nil {
		return err
	}

	config.getProjectConfig().Parse.JSSDK = versions[0]
	return storeProjectConfig(e, config)
}

func (j *jsSDKCmd) run(e *env, c *client, args []string) error {
	switch {
	case j.all:
		return j.printVersions(e, c)
	default:
		if len(args) >= 1 {
			j.newVersion = args[0]
			return j.setVersion(e, c)
		}
		return j.getVersion(e, c)
	}
}

func newJsSdkCmd(e *env) *cobra.Command {
	j := jsSDKCmd{}
	cmd := &cobra.Command{
		Use:   "jssdk [version]",
		Short: "Sets the Parse JavaScript SDK version to use in Cloud Code",
		Long: `Sets the Parse JavaScript SDK version to use in Cloud Code. ` +
			`Prints the version number currently set if none is specified.`,
		Run: runWithArgsClient(e, j.run),
	}
	cmd.Flags().BoolVarP(&j.all, "all", "a", j.all,
		"Shows all available JavaScript SDK version")
	return cmd
}
