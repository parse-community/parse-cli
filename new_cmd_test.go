package main

import (
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

	_, err = os.Stat(filepath.Join(h.env.Root, configDir))
	ensure.Nil(t, err)
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

	content, err := ioutil.ReadFile(filepath.Join(h.env.Root, legacyConfigFile))
	ensure.Nil(t, err)
	ensure.DeepEqual(t, string(content), "{}")
}

func TestNewCmdConfigExists(t *testing.T) {
	t.Parallel()

	h, n := newNewCmdHarness(t)
	defer h.Stop()

	ensure.Nil(t, os.MkdirAll(filepath.Join(h.env.Root, "test", configDir), 0755))
	ensure.Nil(t,
		ioutil.WriteFile(filepath.Join(h.env.Root, "test", legacyConfigFile),
			[]byte("{}"),
			0600,
		),
	)

	h.env.In = ioutil.NopCloser(strings.NewReader("\n"))
	ensure.Err(t,
		n.cloneSampleCloudCode(h.env, &app{Name: "test"}, false),
		regexp.MustCompile("unable to create Cloud Code at test."),
	)
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

	h.env.In = ioutil.NopCloser(strings.NewReader("decide"))
	decision := n.shouldCreateNewApp(h.env)
	ensure.DeepEqual(t, decision, "decide")
}
