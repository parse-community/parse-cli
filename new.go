package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"strconv"

	"github.com/ParsePlatform/parse-cli/herokucmd"
	"github.com/ParsePlatform/parse-cli/parsecli"
	"github.com/ParsePlatform/parse-cli/parsecmd"
	"github.com/facebookgo/parse"
	"github.com/facebookgo/stackerr"
	"github.com/spf13/cobra"
)

type newCmd struct {
	addApplication bool

	noCode       bool   // do not setup a Cloud Code project
	configOnly   bool   // create a Cloud Code project with only configuration, no code
	createNewApp bool   // create a new app
	parseAppName string // name of parse app
	codeLocation string // location of Cloud Code project
}

func (n *newCmd) curlCommand(e *parsecli.Env, app *parsecli.App) string {
	args := "{}"
	if e.Type == parsecli.HerokuFormat {
		args = `{"a": "Adventurous ", "b": "Parser"}`
	}

	return fmt.Sprintf(
		`curl -X POST \
 -H "X-Parse-Application-Id: %s" \
 -H "X-Parse-REST-API-Key: %s" \
 -H "Content-Type: application/json" \
 -d %s \
 https://parseapi.back4app.com/functions/hello
`,
		app.ApplicationID,
		app.RestKey,
		strconv.Quote(args),
	)
}

func (n *newCmd) cloudCodeHelpMessage(e *parsecli.Env, app *parsecli.App) string {
	code := "Cloud Code"
	if e.Type == parsecli.HerokuFormat {
		code = "server code"
	}
	return fmt.Sprintf(
		`Your %s has been created at %s.

This includes a "Hello world" cloud function, so once you deploy,
you can test that it works, with the printed curl command.

Next, you might want to deploy this code with:

	cd %s
	b4a deploy

Once deployed you can test that it works by running:
%s
`,
		code,
		e.Root,
		e.Root,
		n.curlCommand(e, app),
	)
}

func (n *newCmd) getCloudCodeDir(e *parsecli.Env, appName string, isNew bool) (string, error) {
	var cloudCodeDir string
	if isNew {
		fmt.Fprintf(e.Out,
			`Awesome! Now it's time to set up some Cloud Code for the app: %q,
Next we will create a directory to hold your Cloud Code.
Please enter the name to use for this directory,
or hit ENTER to use %q as the directory name.

Directory Name: `,
			appName,
			appName,
		)
	} else {
		fmt.Fprintf(e.Out,
			`Please enter the name of the folder where we can download the latest deployed
Cloud Code for your app %q

Directory Name: `,
			appName,
		)
	}

	fmt.Fscanf(e.In, "%s\n", &cloudCodeDir)
	cloudCodeDir = strings.TrimSpace(cloudCodeDir)
	if cloudCodeDir == "" {
		cloudCodeDir = appName
	}

	helpMsg := "Next time, please select the same app you selected now."
	if isNew {
		helpMsg = `Next time, please select to add Cloud Code to an 'existing' app
and select the app you just created.`
	}

	configFile := filepath.Join(e.Root, cloudCodeDir, parsecli.LegacyConfigFile)
	if _, err := os.Lstat(configFile); err == nil {
		return "", stackerr.Newf(
			`Sorry, we are unable to create Cloud Code at %s.
It seems that you already have Cloud Code at %s.
Please run "b4a new" again.
%s
Please choose a different name for your Cloud Code directory,
so it does not conflict with any other Cloud Code in the current directory.
`,
			cloudCodeDir,
			cloudCodeDir,
			helpMsg,
		)
	}

	cloudCodeDirInfo, err := os.Stat(filepath.Join(e.Root, cloudCodeDir))
	if err == nil {
		fileType := "file"
		if cloudCodeDirInfo.IsDir() {
			fileType = "directory"
		}
		return "", stackerr.Newf(`Sorry, we are unable to create Cloud Code at %s.
In the current directory a %s named: %q already exists.
Please run "b4a new" again.
%s
Please choose a different name for your Cloud Code directory,
so it does not conflict with any other Cloud Code in the current directory.
`,
			cloudCodeDir,
			fileType,
			cloudCodeDir,
			helpMsg,
		)
	}
	if !os.IsNotExist(err) {
		return "", stackerr.Wrap(err)
	}
	return cloudCodeDir, nil
}

func (n *newCmd) promptCreateNewApp(e *parsecli.Env, nonInteractive bool) (string, error) {
	if nonInteractive || n.noCode {
		if n.createNewApp {
			return "new", nil
		}
		return "existing", nil
	}

	msg := `"new" and "existing" are the only valid options.
Please try again ...`

	var decision string
	for i := 0; i < 3; i++ {
		fmt.Fprintf(e.Out,
			`Would you like to create a new app, or add Cloud Code to an existing app?
Type "(n)ew" or "(e)xisting": `,
		)
		fmt.Fscanf(e.In, "%s\n", &decision)
		decision = strings.ToLower(decision)

		if decision == "new" || decision == "n" || decision == "existing" || decision == "e" {
			return decision, nil
		}
		fmt.Fprintln(e.Err, msg)
	}
	return "", stackerr.New(msg)
}

func (n *newCmd) setupSample(
	e *parsecli.Env,
	name string,
	appConfig parsecli.AppConfig,
	isNew bool,
	nonInteractive bool,
) (bool, error) {
	found := parsecli.IsProjectDir(parsecli.GetProjectRoot(e, e.Root))
	if !found {
		root := parsecli.GetLegacyProjectRoot(e, e.Root)
		_, err := os.Lstat(filepath.Join(root, parsecli.LegacyConfigFile))
		found = err == nil
	}
	if found {
		return false, stackerr.New(
			`Detected that you are already inside a Parse project.
Please refrain from creating a Parse project inside another Parse project.
`,
		)
	}

	var (
		cloudCodeDir string
		err          error
	)

	if nonInteractive {
		cloudCodeDir = n.codeLocation
	} else if n.configOnly {
		cloudCodeDir = "" // ensures that "b4a new --init" inits the current directory
	} else {
		cloudCodeDir, err = n.getCloudCodeDir(e, name, isNew)
		if err != nil {
			return false, err
		}
	}
	e.Root = filepath.Join(e.Root, cloudCodeDir)

	switch e.Type {
	case parsecli.ParseFormat:
		if !n.configOnly {
			var decision string
			if isNew {
				fmt.Fprint(e.Out, `
You can either set up a blank project or create a sample Cloud Code project.
Please type "(b)lank" if you wish to setup a blank project, otherwise press ENTER: `)
			} else {
				fmt.Fprint(e.Out, `
You can either set up a blank project or download the current deployed Cloud Code.
Please type "(b)lank" if you wish to setup a blank project, otherwise press ENTER: `)
			}
			fmt.Fscanf(e.In, "%s\n", &decision)
			decision = strings.ToLower(strings.TrimSpace(decision))
			if decision != "" && decision == "b" || decision == "blank" {
				n.configOnly = true
			}
		}
		return parsecmd.CloneSampleCloudCode(e, isNew, n.configOnly, appConfig)
	case parsecli.HerokuFormat:
		if !n.configOnly {
			var decision string
			if isNew {
				fmt.Fprint(e.Out, `
You can either set up a blank project or create a sample Node.js project.
Please type "(b)lank" if you wish to setup a blank project, otherwise press ENTER: `)
			} else {
				fmt.Fprint(e.Out, `
You can either set up a blank project or download the server code currently deployed to Heroku.
Please type "(b)lank" if you wish to setup a blank project, otherwise press ENTER: `)
			}
			fmt.Fscanf(e.In, "%s\n", &decision)
			decision = strings.ToLower(strings.TrimSpace(decision))
			if decision != "" && decision == "b" || decision == "blank" {
				n.configOnly = true
			}
		}
		return herokucmd.CloneNodeCode(e, isNew, n.configOnly, appConfig)
	}
	return false, stackerr.Newf("Unknown project type: %d", e.Type)
}

func (n *newCmd) configureSample(
	add *addCmd,
	name string,
	appConfig parsecli.AppConfig,
	args []string,
	e *parsecli.Env,
) error {
	if err := add.addSelectedApp(name, appConfig, args, e); err != nil {
		return err
	}

	masterKey, err := appConfig.GetMasterKey(e)
	if err != nil {
		return err
	}
	e.ParseAPIClient = e.ParseAPIClient.WithCredentials(
		parse.MasterKey{
			ApplicationID: appConfig.GetApplicationID(),
			MasterKey:     masterKey,
		},
	)

	if e.Type == parsecli.ParseFormat {
		return parsecmd.UseLatestJSSDK(e)
	}
	return nil
}

func (n *newCmd) run(e *parsecli.Env) error {
	apps := &parsecli.Apps{}
	addCmd := &addCmd{MakeDefault: true, apps: apps}

	if err := apps.Login.AuthUser(e, false); err != nil {
		return err
	}
	var (
		app *parsecli.App
		err error
	)

	nonInteractive := n.parseAppName != "" && n.codeLocation != ""
	decision, err := n.promptCreateNewApp(e, nonInteractive)
	if err != nil {
		return err
	}
	isNew := false

	switch decision {
	case "new", "n":
		isNew = true
		var createRetries int
		if n.noCode {
			createRetries = 1
		}
		// we pass retries so that even in non interactive mode we can create an app,
		// and failure does not print 3 times to the screen
		app, err = apps.CreateApp(e, n.parseAppName, createRetries)
		if err != nil {
			return err
		}
		if n.noCode {
			fmt.Fprintln(e.Out, "Successfully created the app.")
			return apps.PrintApp(e, app)
		}
	case "existing", "e":
		app, err = addCmd.selectApp(e, n.parseAppName)
		if err != nil {
			return err
		}
		if n.noCode {
			fmt.Fprintln(e.Out, "Successfully selected the app.")
			return apps.PrintApp(e, app)
		}
	}

	projectType, err := herokucmd.PromptCreateWebhooks(e)
	if err != nil {
		return err
	}

	var appConfig parsecli.AppConfig
	switch projectType {
	case "heroku":
		e.Type = parsecli.HerokuFormat
		var newHerokuApp bool
		newHerokuApp, appConfig, err = herokucmd.GetLinkedHerokuAppConfig(app, e)
		if err != nil {
			return err
		}
		isNew = isNew || newHerokuApp

	case "parse":
		e.Type = parsecli.ParseFormat
		appConfig = parsecmd.GetParseAppConfig(app)
	}

	dumpTemplate, err := n.setupSample(e, app.Name, appConfig, isNew, nonInteractive)
	if err != nil {
		return err
	}
	if err := n.configureSample(addCmd, app.Name, appConfig, nil, e); err != nil {
		return err
	}
	if token := apps.Login.Credentials.Token; token != "" {
		email, err := apps.Login.AuthToken(e, token)
		if err != nil {
			return err
		}
		if err := parsecli.SetParserEmail(e, email); err != nil {
			return err
		}
	}

	if dumpTemplate {
		fmt.Fprintf(e.Out, n.cloudCodeHelpMessage(e, app))
	}

	return nil
}

func NewNewCmd(e *parsecli.Env) *cobra.Command {
	nc := newCmd{addApplication: true}
	cmd := &cobra.Command{
		Use:   "new",
		Short: "Adds Cloud Code to an existing Parse app, additional can create a new Parse app",
		Long: `Adds Cloud Code to an existing Parse app, additional can create a new Parse app.
You can also use it in non-interactive mode by using the various flags available.
`,
		Run: parsecli.RunNoArgs(e, nc.run),
	}
	cmd.Flags().BoolVarP(&nc.configOnly, "init", "i", nc.configOnly,
		"Create a Cloud Code project only with configuration.")
	cmd.Flags().BoolVarP(&nc.noCode, "nocode", "n", nc.noCode,
		"Do not set up a Cloud Code project for the app. Typically used in conjunction with 'create' option")
	cmd.Flags().BoolVarP(&nc.createNewApp, "create", "c", nc.createNewApp,
		"Set this flag to true if you want to create a new Parse app.")
	cmd.Flags().StringVarP(&nc.parseAppName, "app", "a", nc.parseAppName,
		"Name of the Parse app you want to create or set up Cloud Code project for.")
	cmd.Flags().StringVarP(&nc.codeLocation, "loc", "l", nc.codeLocation,
		"Location, relative to the current directory, at which the Cloud Code project will be set up.")
	return cmd
}
