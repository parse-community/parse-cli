package parsecmd

import (
	"fmt"
	"net/url"
	"sort"

	"github.com/ParsePlatform/parse-cli/parsecli"
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

func (j *jsSDKCmd) getAllJSSdks(e *parsecli.Env) ([]string, error) {
	u := &url.URL{
		Path: "jsVersions",
	}
	var response jsSDKVersion
	if _, err := e.ParseAPIClient.Get(u, &response); err != nil {
		return nil, stackerr.Wrap(err)
	}
	sort.Sort(naturalStrings(response.JS))
	return response.JS, nil
}

func (j *jsSDKCmd) printVersions(e *parsecli.Env, c *parsecli.Context) error {
	allVersions, err := j.getAllJSSdks(e)
	if err != nil {
		return err
	}

	currentVersion := c.Config.GetProjectConfig().Parse.JSSDK
	for _, version := range allVersions {
		prefix := "  "
		if currentVersion == version {
			prefix = "* "
		}
		fmt.Fprintf(e.Out, "%s %s\n", prefix, version)
	}
	return nil
}

func (j *jsSDKCmd) getVersion(e *parsecli.Env, c *parsecli.Context) error {
	if c.Config.GetProjectConfig().Parse.JSSDK == "" {
		return stackerr.New("JavaScript SDK version not set for this project.")
	}
	fmt.Fprintf(
		e.Out,
		"Current JavaScript SDK version is %s\n",
		c.Config.GetProjectConfig().Parse.JSSDK,
	)
	return nil
}

func (j *jsSDKCmd) setVersion(e *parsecli.Env, c *parsecli.Context) error {
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

	conf, err := parsecli.ConfigFromDir(e.Root)
	if err != nil {
		return err
	}
	conf.GetProjectConfig().Parse.JSSDK = j.newVersion
	if err := parsecli.StoreProjectConfig(e, conf); err != nil {
		return err
	}

	fmt.Fprintf(e.Out, "Current JavaScript SDK version is %s\n", conf.GetProjectConfig().Parse.JSSDK)
	return nil
}

// useLatestJSSDK is a utility method used by deploy & develop
// to write set jssdk version to latest available, if none set
func useLatestJSSDK(e *parsecli.Env) error {
	var j jsSDKCmd
	versions, err := j.getAllJSSdks(e)
	if err != nil {
		return err
	}

	if len(versions) == 0 {
		return stackerr.New("No JavaScript SDK version is available.")
	}

	config, err := parsecli.ConfigFromDir(e.Root)
	if err != nil {
		return err
	}

	config.GetProjectConfig().Parse.JSSDK = versions[0]
	return parsecli.StoreProjectConfig(e, config)
}

func (j *jsSDKCmd) run(e *parsecli.Env, c *parsecli.Context, args []string) error {
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

func NewJsSdkCmd(e *parsecli.Env) *cobra.Command {
	j := jsSDKCmd{}
	cmd := &cobra.Command{
		Use:   "jssdk [version]",
		Short: "Sets the Parse JavaScript SDK version to use in Cloud Code",
		Long: `Sets the Parse JavaScript SDK version to use in Cloud Code. ` +
			`Prints the version number currently set if none is specified.`,
		Run: parsecli.RunWithArgsClient(e, j.run),
	}
	cmd.Flags().BoolVarP(&j.all, "all", "a", j.all,
		"Shows all available JavaScript SDK version")
	return cmd
}
