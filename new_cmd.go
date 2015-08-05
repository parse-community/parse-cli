package main

import (
	"errors"
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
	fmt.Fprintf(e.Out,
		`Awesome! Now it's time to setup some Cloud Code for the app: %q,
Next we will create a directory to hold your Cloud Code.
Please enter the name to use for this directory,
or hit ENTER to use %q as the directory name.

Directory Name: `,
		appName,
		appName,
	)

	fmt.Scanf("%s\n", &cloudCodeDir)
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

func (n *newCmd) cloneSampleCloudCode(e *env, app *app, isNew bool) error {
	cloudCodeDir, err := n.getCloudCodeDir(e, app.Name, isNew)
	if err != nil {
		return err
	}
	e.Root = filepath.Join(e.Root, cloudCodeDir)

	if err != nil {
		return err
	}
	err = os.MkdirAll(e.Root, 0755)
	if err != nil {
		return stackerr.Wrap(err)
	}

	if err := os.Mkdir(filepath.Join(e.Root, configDir), 0755); err != nil {
		return stackerr.Wrap(err)
	}
	file, err := os.OpenFile(
		filepath.Join(e.Root, legacyConfigFile),
		os.O_RDWR|os.O_CREATE|os.O_TRUNC,
		0600,
	)
	if err != nil && !os.IsExist(err) {
		return stackerr.Wrap(err)
	}
	defer file.Close()
	if _, err := file.WriteString("{}"); err != nil {
		return stackerr.Wrap(err)
	}
	if err := file.Close(); err != nil {
		return stackerr.Wrap(err)
	}

	for _, info := range newProjectFiles {
		sampleDir := filepath.Join(e.Root, info.dirname)
		if _, err := os.Stat(sampleDir); err != nil {
			if !os.IsNotExist(err) {
				return stackerr.Wrap(err)
			}
			if err := os.Mkdir(sampleDir, 0755); err != nil {
				return stackerr.Wrap(err)
			}
		}

		sampleFile := filepath.Join(sampleDir, info.filename)
		if _, err := os.Stat(sampleFile); err != nil {
			if os.IsNotExist(err) {
				file, err := os.Create(sampleFile)
				if err != nil && !os.IsExist(err) {
					return stackerr.Wrap(err)
				}

				defer file.Close()

				if _, err := file.WriteString(info.content); err != nil {
					return stackerr.Wrap(err)
				}
				if err := file.Close(); err != nil {
					return stackerr.Wrap(err)
				}
			}
		}
	}
	fmt.Fprintf(e.Out, n.cloudCodeHelpMessage(e, app))
	return nil
}

func (n *newCmd) shouldCreateNewApp(e *env) string {
	var decision string
	fmt.Fprintf(e.Out,
		`Would you like to create a new app, or add Cloud Code to an existing app?
Type "new" or "existing": `)
	fmt.Fscanf(e.In, "%s\n", &decision)
	return strings.ToLower(decision)
}

func (n *newCmd) setupSample(e *env, app *app, isNew bool) error {
	_, err := os.Lstat(
		filepath.Join(e.Root, legacyConfigFile),
	)
	if err == nil {
		return stackerr.New(`Detected that you are already inside a Parse project.
Please refrain from creating a Parse project inside another Parse project.
`)
	}

	return n.cloneSampleCloudCode(e, app, isNew)
}

func (n *newCmd) configureSample(
	add *addCmd,
	app *app,
	e *env,
	args []string) error {
	if err := add.writeConfig(app, args, e, false); err != nil {
		return err
	}

	// at this point user has already set a default app for the project
	// use its properties to fetch latest jssdk version and set it in config
	config, err := configFromDir(e.Root)
	if err != nil {
		return err
	}
	defaultApp, err := config.app(config.getDefaultApp())
	if err != nil {
		return err
	}
	masterKey, err := defaultApp.getMasterKey(e)
	if err != nil {
		return err
	}
	e.Client = e.Client.WithCredentials(
		parse.MasterKey{
			ApplicationID: defaultApp.getApplicationID(),
			MasterKey:     masterKey,
		},
	)
	return useLatestJSSDK(e)
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
	decision := n.shouldCreateNewApp(e)
	if decision != "new" && decision != "existing" {
		return errors.New("`new` and `existing` are the only valid options")
	}

	isNew := false
	switch decision {
	case "new":
		isNew = true
		app, err = apps.createApp(e)
		if err != nil {
			return err
		}
	case "existing":
		app, err = addCmd.selectApp(e)
		if err != nil {
			return err
		}
	}

	if err := n.setupSample(e, app, isNew); err != nil {
		return err
	}
	return n.configureSample(addCmd, app, e, nil)
}

func newNewCmd(e *env) *cobra.Command {
	nc := newCmd{addApplication: true}
	cmd := &cobra.Command{
		Use:   "new",
		Short: "Creates a new Parse app and adds Cloud Code to an existing Parse app",
		Long:  `Creates a new Parse app and adds Cloud Code to an existing Parse app.`,
		Run:   runNoArgs(e, nc.run),
	}
	return cmd
}
