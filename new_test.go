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

	"github.com/ParsePlatform/parse-cli/parsecli"
	"github.com/facebookgo/ensure"
	"github.com/facebookgo/parse"
)

func newNewCmdHarness(t testing.TB) (*parsecli.Harness, *newCmd) {
	h := parsecli.NewHarness(t)
	h.MakeEmptyRoot()
	return h, &newCmd{}
}

func TestCloudCodeHelpMessage(t *testing.T) {
	t.Parallel()
	h := parsecli.NewHarness(t)
	defer h.Stop()

	n := &newCmd{}
	msg := n.cloudCodeHelpMessage(h.Env, &parsecli.App{ApplicationID: "a", MasterKey: "m"})
	ensure.StringContains(t,
		msg,
		fmt.Sprintf(
			`Your Cloud Code has been created at %s.

This includes a "Hello world" cloud function, so once you deploy,
you can test that it works, with`,
			h.Env.Root,
		),
	)
}

func TestGetCloudCodeDir(t *testing.T) {
	t.Parallel()
	h := parsecli.NewHarness(t)
	h.MakeEmptyRoot()
	defer h.Stop()

	n := &newCmd{}
	h.Env.In = ioutil.NopCloser(strings.NewReader("\n"))
	name, err := n.getCloudCodeDir(h.Env, "myapp", true)
	ensure.Nil(t, err)
	ensure.StringContains(t, h.Out.String(), "Now it's time to set up some Cloud Code")
	ensure.DeepEqual(t, name, "myapp")

	h.Out.Reset()
	h.Env.In = ioutil.NopCloser(strings.NewReader("otherApp\n"))
	name, err = n.getCloudCodeDir(h.Env, "myapp", true)
	ensure.Nil(t, err)
	ensure.StringContains(t, h.Out.String(), "Now it's time to set up some Cloud Code")
	ensure.DeepEqual(t, name, "otherApp")

	_, err = os.Create(filepath.Join(h.Env.Root, "otherApp"))
	ensure.Nil(t, err)
	h.Out.Reset()
	h.Env.In = ioutil.NopCloser(strings.NewReader("otherApp\n"))
	name, err = n.getCloudCodeDir(h.Env, "myapp", true)
	ensure.Err(t, err, regexp.MustCompile("already exists"))
	ensure.StringContains(t, h.Out.String(), "Now it's time to set up some Cloud Code")

	ensure.Nil(t, os.MkdirAll(filepath.Join(h.Env.Root, "myapp", "config"), 0755))
	_, err = os.Create(filepath.Join(h.Env.Root, "myapp", "config", "global.json"))
	ensure.Nil(t, err)
	h.Out.Reset()
	h.Env.In = ioutil.NopCloser(strings.NewReader("\n"))
	name, err = n.getCloudCodeDir(h.Env, "myapp", true)
	ensure.Err(t, err, regexp.MustCompile("you already have Cloud Code"))
	ensure.Nil(t, os.Remove(filepath.Join(h.Env.Root, "myapp", "config", "global.json")))

	h.Out.Reset()
	h.Env.In = ioutil.NopCloser(strings.NewReader("\n"))
	name, err = n.getCloudCodeDir(h.Env, "myapp", false)
	ensure.Err(t, err, regexp.MustCompile("a directory named: \"myapp\" already exists"))
}

func TestNewCmdDirs(t *testing.T) {
	t.Parallel()

	h, _ := newNewCmdHarness(t)
	defer h.Stop()

	ensure.Nil(t, parsecli.CloneSampleCloudCode(h.Env, true))

	var err error

	for _, newProjectFile := range parsecli.NewProjectFiles {
		_, err = os.Stat(filepath.Join(h.Env.Root, newProjectFile.Dirname))
		ensure.Nil(t, err)
	}
}

func TestNewCmdContent(t *testing.T) {
	t.Parallel()

	h, _ := newNewCmdHarness(t)
	defer h.Stop()

	ensure.Nil(t, parsecli.CloneSampleCloudCode(h.Env, true))

	for _, newProjectFile := range parsecli.NewProjectFiles {
		content, err := ioutil.ReadFile(
			filepath.Join(h.Env.Root, newProjectFile.Dirname, newProjectFile.Filename),
		)
		ensure.Nil(t, err)
		ensure.DeepEqual(t, string(content), newProjectFile.Content)
	}

	content, err := ioutil.ReadFile(filepath.Join(h.Env.Root, parsecli.ParseProject))
	ensure.Nil(t, err)
	ensure.DeepEqual(t,
		string(content),
		fmt.Sprintf(
			`{
  "project_type" : %d,
  "parse": {"jssdk":""}
}`,
			parsecli.ParseFormat,
		),
	)

	content, err = ioutil.ReadFile(filepath.Join(h.Env.Root, parsecli.ParseLocal))
	ensure.Nil(t, err)
	ensure.DeepEqual(t, string(content), "{}")

}

func TestCurlCommand(t *testing.T) {
	t.Parallel()
	h := parsecli.NewHarness(t)
	defer h.Stop()

	n := &newCmd{}
	command := n.curlCommand(h.Env, &parsecli.App{ApplicationID: "AppID", RestKey: "RestKey"})
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
	h, _ := parsecli.NewAppHarness(t)
	defer h.Stop()

	h.Env.In = ioutil.NopCloser(strings.NewReader("email\npassword"))
	n := &newCmd{noCode: true, createNewApp: true, parseAppName: "yolo"}
	ensure.Nil(t, n.run(h.Env))
	ensure.StringContains(t, h.Out.String(), "Successfully created")
}

func TestSelectNewAppNoCode(t *testing.T) {
	t.Parallel()
	h, _ := parsecli.NewAppHarness(t)
	defer h.Stop()

	h.Env.In = ioutil.NopCloser(strings.NewReader("email\npassword\n"))
	n := &newCmd{noCode: true, parseAppName: "A"}
	ensure.Nil(t, n.run(h.Env))
	ensure.StringContains(t, h.Out.String(), "Successfully selected")
}

func TestShouldCreateNewApp(t *testing.T) {
	t.Parallel()
	h := parsecli.NewHarness(t)
	defer h.Stop()

	n := &newCmd{}

	h.Env.In = ioutil.NopCloser(strings.NewReader("new"))
	decision, err := n.promptCreateNewApp(h.Env, false)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, decision, "new")

	h.Env.In = ioutil.NopCloser(strings.NewReader("existing"))
	decision, err = n.promptCreateNewApp(h.Env, false)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, decision, "existing")

	h.Env.In = ioutil.NopCloser(strings.NewReader("other"))
	_, err = n.promptCreateNewApp(h.Env, false)
	ensure.Err(t, err, regexp.MustCompile("are the only valid options"))

	decision, err = n.promptCreateNewApp(h.Env, true)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, decision, "existing")

	n.createNewApp = true
	decision, err = n.promptCreateNewApp(h.Env, true)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, decision, "new")
}

func TestSetupAndConfigure(t *testing.T) {
	t.Parallel()
	h := parsecli.NewHarness(t)
	h.MakeEmptyRoot()
	defer h.Stop()

	n := &newCmd{}
	h.Env.Type = parsecli.ParseFormat
	h.Env.In = ioutil.NopCloser(strings.NewReader("\n"))
	code, err := n.setupSample(h.Env, "myapp", &parsecli.ParseAppConfig{ApplicationID: "a"}, true, false)
	ensure.Nil(t, err)
	ensure.True(t, code)

	type jsSDKVersion struct {
		JS []string `json:"js"`
	}
	ht := parsecli.TransportFunc(func(r *http.Request) (*http.Response, error) {
		ensure.DeepEqual(t, r.URL.Path, "/1/jsVersions")
		rows := jsSDKVersion{JS: []string{"1.5.0", "1.6.0"}}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(strings.NewReader(jsonStr(t, rows))),
		}, nil
	})
	h.Env.ParseAPIClient = &parsecli.ParseAPIClient{APIClient: &parse.Client{Transport: ht}}
	err = n.configureSample(
		&addCmd{MakeDefault: true},
		"yolo",
		(&parsecli.ParseAppConfig{ApplicationID: "a"}).WithInternalMasterKey("m"),
		nil,
		h.Env,
	)
	ensure.Nil(t, err)
}
