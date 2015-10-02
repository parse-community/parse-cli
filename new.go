package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	codeLocation string // location of cloud code project
}

var newProjectFiles = []struct {
	dirname, filename, content string
}{
	{"cloud", "main.js", sampleSource},
	{"public", "index.html", sampleHTML},
}

func (n *newCmd) curlCommand(app *app) string {
	return fmt.Sprintf(
		`curl -X POST \
 -H "X-Parse-Application-Id: %s" \
 -H "X-Parse-REST-API-Key: %s" \
 -H "Content-Type: application/json" \
 -d '{}' \
 https://api.parse.com/1/functions/hello
`,
		app.ApplicationID,
		app.RestKey,
	)
}

func (n *newCmd) cloudCodeHelpMessage(e *env, app *app) string {
	return fmt.Sprintf(
		`Your Cloud Code has been created at %s.
Next, you might want to deploy this code with "parse deploy".
This includes a "Hello world" cloud function, so once you deploy
you can test that it works, with:

%s
`,
		e.Root,
		n.curlCommand(app),
	)
}

func (n *newCmd) getCloudCodeDir(e *env, appName string, isNew bool) (string, error) {
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

	configFile := filepath.Join(e.Root, cloudCodeDir, legacyConfigFile)
	if _, err := os.Lstat(configFile); err == nil {
		return "", stackerr.Newf(
			`Sorry, we are unable to create Cloud Code at %s.
It seems that you already have Cloud Code at %s.
Please run "parse new" again.
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
		if !cloudCodeDirInfo.IsDir() {
			return "", stackerr.Newf(`Sorry, we are unable to create Cloud Code at %s.
In the current directory a file named: %s already exists.
Please run "parse new" again.
%s
Please choose a different name for your Cloud Code directory,
so it does not conflict with any other Cloud Code in the current directory.
`,
				cloudCodeDir,
				cloudCodeDir,
				helpMsg,
			)

		}
		return "", nil
	}
	if !os.IsNotExist(err) {
		return "", stackerr.Wrap(err)
	}
	return cloudCodeDir, nil
}

func (n *newCmd) createConfigWithContent(path, content string) error {
	file, err := os.OpenFile(
		path,
		os.O_RDWR|os.O_CREATE|os.O_TRUNC,
		0600,
	)
	if err != nil && !os.IsExist(err) {
		return stackerr.Wrap(err)
	}
	defer file.Close()
	if _, err := file.WriteString(content); err != nil {
		return stackerr.Wrap(err)
	}
	if err := file.Close(); err != nil {
		return stackerr.Wrap(err)
	}
	return nil
}

func (n *newCmd) promptCreateNewApp(e *env, nonInteractive bool) (string, error) {
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
	e *env,
	name string,
	appConfig appConfig,
	isNew bool,
	nonInteractive bool,
) (bool, error) {
	found := isProjectDir(getProjectRoot(e, e.Root))
	if !found {
		root := getLegacyProjectRoot(e, e.Root)
		_, err := os.Lstat(filepath.Join(root, legacyConfigFile))
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
		cloudCodeDir = "" // ensures that "parse new --init" inits the current directory
	} else {
		cloudCodeDir, err = n.getCloudCodeDir(e, name, isNew)
		if err != nil {
			return false, err
		}
	}
	e.Root = filepath.Join(e.Root, cloudCodeDir)

	switch e.Type {
	case parseFormat:
		dumpTemplate := false
		if !isNew && !n.configOnly {
			// if parse app was already created try to fetch cloud code and populate dir
			masterKey, err := appConfig.getMasterKey(e)
			if err != nil {
				return false, err
			}
			e.ParseAPIClient = e.ParseAPIClient.WithCredentials(
				parse.MasterKey{
					ApplicationID: appConfig.getApplicationID(),
					MasterKey:     masterKey,
				},
			)

			d := &downloadCmd{destination: e.Root}
			err = d.run(e, nil)
			if err != nil {
				if err == errNoFiles {
					dumpTemplate = true
				} else {
					fmt.Fprintln(
						e.Out,
						`
NOTE: If you like to fetch the latest deployed Cloud Code from Parse, 
you can use the "parse download" command after finishing the set up.
This will download Cloud Code to a temporary location.
`,
					)
				}
			}
		}
		dumpTemplate = (isNew || dumpTemplate) && !n.configOnly
		return dumpTemplate, n.cloneSampleCloudCode(e, dumpTemplate)
	}
	return false, stackerr.Newf("Unknown project type: %d", e.Type)
}

func (n *newCmd) configureSample(
	add *addCmd,
	name string,
	appConfig appConfig,
	args []string,
	e *env,
) error {
	if err := add.addSelectedApp(name, appConfig, args, e); err != nil {
		return err
	}

	masterKey, err := appConfig.getMasterKey(e)
	if err != nil {
		return err
	}
	e.ParseAPIClient = e.ParseAPIClient.WithCredentials(
		parse.MasterKey{
			ApplicationID: appConfig.getApplicationID(),
			MasterKey:     masterKey,
		},
	)

	if e.Type == parseFormat {
		return useLatestJSSDK(e)
	}
	return nil
}

func (n *newCmd) run(e *env) error {
	apps := &apps{}
	addCmd := &addCmd{MakeDefault: true, apps: apps}

	if err := apps.login.authUser(e); err != nil {
		return err
	}
	var (
		app *app
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
		app, err = apps.createApp(e, n.parseAppName, createRetries)
		if err != nil {
			return err
		}
		if n.noCode {
			fmt.Fprintln(e.Out, "Successfully created the app.")
			return apps.printApp(e, app)
		}
	case "existing", "e":
		app, err = addCmd.selectApp(e, n.parseAppName)
		if err != nil {
			return err
		}
		if n.noCode {
			fmt.Fprintln(e.Out, "Successfully selected the app.")
			return apps.printApp(e, app)
		}
	}

	e.Type = parseFormat
	appConfig := addCmd.getParseAppConfig(app)

	dumpTemplate, err := n.setupSample(e, app.Name, appConfig, isNew, nonInteractive)
	if err != nil {
		return err
	}
	if err := n.configureSample(addCmd, app.Name, appConfig, nil, e); err != nil {
		return err
	}
	if token := apps.login.credentials.token; token != "" {
		email, err := apps.login.authToken(e, token)
		if err != nil {
			return err
		}
		if err := (&configureCmd{}).parserEmail(e, []string{email}); err != nil {
			return err
		}
	}

	if dumpTemplate {
		fmt.Fprintf(e.Out, n.cloudCodeHelpMessage(e, app))
	}

	return nil
}

func newNewCmd(e *env) *cobra.Command {
	nc := newCmd{addApplication: true}
	cmd := &cobra.Command{
		Use:   "new",
		Short: "Adds Cloud Code to an existing Parse app, additional can create a new Parse app",
		Long: `Adds Cloud Code to an existing Parse app, additional can create a new Parse app.
You can also use it in non-interactive mode by using the various flags available.
`,
		Run: runNoArgs(e, nc.run),
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
