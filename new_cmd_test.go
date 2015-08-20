package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/facebookgo/ensure"
)

func newNewCmdHarness(t testing.TB) (*Harness, *newCmd) {
	h := newHarness(t)
	h.makeEmptyRoot()
	return h, &newCmd{}
}

func TestNewCmdDirs(t *testing.T) {
	t.Parallel()

	h, n := newNewCmdHarness(t)
	defer h.Stop()

	ensure.Nil(t, n.cloneSampleCloudCode(h.env, &app{Name: "test"}, false))

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

	ensure.Nil(t, n.cloneSampleCloudCode(h.env, &app{Name: "test"}, false))

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

func TestShouldCreateNewApp(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	defer h.Stop()

	n := &newCmd{}

	h.env.In = ioutil.NopCloser(strings.NewReader("new"))
	decision, err := n.promptCreateNewApp(h.env)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, decision, "new")

	h.env.In = ioutil.NopCloser(strings.NewReader("existing"))
	decision, err = n.promptCreateNewApp(h.env)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, decision, "existing")

	h.env.In = ioutil.NopCloser(strings.NewReader("other"))
	_, err = n.promptCreateNewApp(h.env)
	ensure.Err(t, err, regexp.MustCompile("are the only valid options"))
}
