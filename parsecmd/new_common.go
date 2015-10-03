package parsecmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ParsePlatform/parse-cli/parsecli"
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

func (n *newCmd) curlCommand(app *parsecli.App) string {
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

func (n *newCmd) cloudCodeHelpMessage(e *parsecli.Env, app *parsecli.App) string {
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
		cloudCodeDir = "" // ensures that "parse new --init" inits the current directory
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
You can either set up a blank Cloud Code project or create a template project.
Please type [y] if you wish to set up a template Cloud Code project or [n] for blank project (y/n): `)
			} else {
				fmt.Fprint(e.Out, `
You can either set up a blank Cloud Code project or download the current deployed Cloud Code.
Please type [y] if you wish to download the current Cloud Code or [n] for blank project (y/n): `)
			}
			fmt.Fscanf(e.In, "%s\n", &decision)
			if strings.TrimSpace(decision) == "n" {
				n.configOnly = true
			}
		}

		dumpTemplate := false
		if !isNew && !n.configOnly {
			// if parse app was already created try to fetch cloud code and populate dir
			masterKey, err := appConfig.GetMasterKey(e)
			if err != nil {
				return false, err
			}
			e.ParseAPIClient = e.ParseAPIClient.WithCredentials(
				parse.MasterKey{
					ApplicationID: appConfig.GetApplicationID(),
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
		return dumpTemplate, parsecli.CloneSampleCloudCode(e, dumpTemplate)
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
		return useLatestJSSDK(e)
	}
	return nil
}

func (n *newCmd) run(e *parsecli.Env) error {
	apps := &parsecli.Apps{}
	addCmd := &addCmd{MakeDefault: true, apps: apps}

	if err := apps.Login.AuthUser(e); err != nil {
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

	e.Type = parsecli.ParseFormat
	appConfig := addCmd.getParseAppConfig(app)

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
