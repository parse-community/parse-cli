package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/facebookgo/ensure"
	"github.com/facebookgo/parse"
)

func newNewCmdHarness(t testing.TB) (*Harness, *newCmd) {
	h := newHarness(t)
	h.makeEmptyRoot()
	return h, &newCmd{}
}

func TestCloudCodeHelpMessage(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	defer h.Stop()

	n := &newCmd{}
	msg := n.cloudCodeHelpMessage(h.env, &app{ApplicationID: "a", MasterKey: "m"})
	ensure.StringContains(t,
		msg,
		fmt.Sprintf(
			`Your Cloud Code has been created at %s.
Next, you might want to deploy this code with "parse deploy".
This includes a "Hello world" cloud function, so once you deploy
you can test that it works, with:`,
			h.env.Root,
		),
	)
}

func TestGetCloudCodeDir(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.makeEmptyRoot()
	defer h.Stop()

	n := &newCmd{}
	h.env.In = ioutil.NopCloser(strings.NewReader("\n"))
	name, err := n.getCloudCodeDir(h.env, "myapp", true)
	ensure.Nil(t, err)
	ensure.StringContains(t, h.Out.String(), "Now it's time to set up some Cloud Code")
	ensure.DeepEqual(t, name, "myapp")

	h.Out.Reset()
	h.env.In = ioutil.NopCloser(strings.NewReader("otherApp\n"))
	name, err = n.getCloudCodeDir(h.env, "myapp", true)
	ensure.Nil(t, err)
	ensure.StringContains(t, h.Out.String(), "Now it's time to set up some Cloud Code")
	ensure.DeepEqual(t, name, "otherApp")

	_, err = os.Create(filepath.Join(h.env.Root, "otherApp"))
	ensure.Nil(t, err)
	h.Out.Reset()
	h.env.In = ioutil.NopCloser(strings.NewReader("otherApp\n"))
	name, err = n.getCloudCodeDir(h.env, "myapp", true)
	ensure.Err(t, err, regexp.MustCompile("already exists"))
	ensure.StringContains(t, h.Out.String(), "Now it's time to set up some Cloud Code")

	ensure.Nil(t, os.MkdirAll(filepath.Join(h.env.Root, "myapp", "config"), 0755))
	_, err = os.Create(filepath.Join(h.env.Root, "myapp", "config", "global.json"))
	ensure.Nil(t, err)
	h.Out.Reset()
	h.env.In = ioutil.NopCloser(strings.NewReader("\n"))
	name, err = n.getCloudCodeDir(h.env, "myapp", true)
	ensure.Err(t, err, regexp.MustCompile("you already have Cloud Code"))
	ensure.Nil(t, os.Remove(filepath.Join(h.env.Root, "myapp", "config", "global.json")))

	h.Out.Reset()
	h.env.In = ioutil.NopCloser(strings.NewReader("\n"))
	name, err = n.getCloudCodeDir(h.env, "myapp", false)
	ensure.Nil(t, err)
	ensure.StringContains(t, h.Out.String(), "folder where we can download the latest")
}

func TestNewCmdDirs(t *testing.T) {
	t.Parallel()

	h, n := newNewCmdHarness(t)
	defer h.Stop()

	ensure.Nil(t, n.cloneSampleCloudCode(h.env, true))

	var err error

	for _, newProjectFile := range newProjectFiles {
		_, err = os.Stat(filepath.Join(h.env.Root, newProjectFile.dirname))
		ensure.Nil(t, err)
	}
}

func TestNewCmdContent(t *testing.T) {
	t.Parallel()

	h, n := newNewCmdHarness(t)
	defer h.Stop()

	ensure.Nil(t, n.cloneSampleCloudCode(h.env, true))

	for _, newProjectFile := range newProjectFiles {
		content, err := ioutil.ReadFile(
			filepath.Join(h.env.Root, newProjectFile.dirname, newProjectFile.filename),
		)
		ensure.Nil(t, err)
		ensure.DeepEqual(t, string(content), newProjectFile.content)
	}

	content, err := ioutil.ReadFile(filepath.Join(h.env.Root, parseProject))
	ensure.Nil(t, err)
	ensure.DeepEqual(t,
		string(content),
		fmt.Sprintf(
			`{
  "project_type" : %d,
  "parse": {"jssdk":""}
}`,
			parseFormat,
		),
	)

	content, err = ioutil.ReadFile(filepath.Join(h.env.Root, parseLocal))
	ensure.Nil(t, err)
	ensure.DeepEqual(t, string(content), "{}")

}

func TestCurlCommand(t *testing.T) {
	t.Parallel()
	n := &newCmd{}
	command := n.curlCommand(&app{ApplicationID: "AppID", RestKey: "RestKey"})
	ensure.DeepEqual(t,
		command,
		`curl -X POST \
 -H "X-Parse-Application-Id: AppID" \
 -H "X-Parse-REST-API-Key: RestKey" \
 -H "Content-Type: application/json" \
 -d '{}' \
 https://api.parse.com/1/functions/hello
`)
}

func TestCreateNewAppNoCode(t *testing.T) {
	t.Parallel()
	h, _ := newAppHarness(t)
	defer h.Stop()

	h.env.In = ioutil.NopCloser(strings.NewReader("email\npassword"))
	n := &newCmd{noCode: true, createNewApp: true, parseAppName: "yolo"}
	ensure.Nil(t, n.run(h.env))
	ensure.StringContains(t, h.Out.String(), "Successfully created")
}

func TestSelectNewAppNoCode(t *testing.T) {
	t.Parallel()
	h, _ := newAppHarness(t)
	defer h.Stop()

	h.env.In = ioutil.NopCloser(strings.NewReader("email\npassword\n"))
	n := &newCmd{noCode: true, parseAppName: "A"}
	ensure.Nil(t, n.run(h.env))
	ensure.StringContains(t, h.Out.String(), "Successfully selected")
}

func TestShouldCreateNewApp(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	defer h.Stop()

	n := &newCmd{}

	h.env.In = ioutil.NopCloser(strings.NewReader("new"))
	decision, err := n.promptCreateNewApp(h.env, false)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, decision, "new")

	h.env.In = ioutil.NopCloser(strings.NewReader("existing"))
	decision, err = n.promptCreateNewApp(h.env, false)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, decision, "existing")

	h.env.In = ioutil.NopCloser(strings.NewReader("other"))
	_, err = n.promptCreateNewApp(h.env, false)
	ensure.Err(t, err, regexp.MustCompile("are the only valid options"))

	decision, err = n.promptCreateNewApp(h.env, true)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, decision, "existing")

	n.createNewApp = true
	decision, err = n.promptCreateNewApp(h.env, true)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, decision, "new")
}

func TestSetupAndConfigure(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.makeEmptyRoot()
	defer h.Stop()

	n := &newCmd{}
	h.env.Type = parseFormat
	h.env.In = ioutil.NopCloser(strings.NewReader("\n"))
	code, err := n.setupSample(h.env, "myapp", &parseAppConfig{ApplicationID: "a"}, true, false)
	ensure.Nil(t, err)
	ensure.True(t, code)

	ht := transportFunc(func(r *http.Request) (*http.Response, error) {
		ensure.DeepEqual(t, r.URL.Path, "/1/jsVersions")
		rows := jsSDKVersion{JS: []string{"1.5.0", "1.6.0"}}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(strings.NewReader(jsonStr(t, rows))),
		}, nil
	})
	h.env.ParseAPIClient = &ParseAPIClient{apiClient: &parse.Client{Transport: ht}}
	err = n.configureSample(
		&addCmd{MakeDefault: true},
		"yolo",
		&parseAppConfig{ApplicationID: "a", masterKey: "m"},
		nil,
		h.env,
	)
	ensure.Nil(t, err)

	d := &defaultCmd{}
	h.Out.Reset()
	ensure.Nil(t, d.run(h.env, nil))
	ensure.DeepEqual(t, h.Out.String(), "Current default app is yolo\n")
}
