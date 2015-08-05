package main

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/facebookgo/ensure"
)

func TestGenerateCheckValidArgs(t *testing.T) {
	t.Parallel()
	g := &generateCmd{}
	for arg := range validTypes {
		g.generateType = arg
		err := g.validateArgs()
		ensure.Nil(t, err)
	}
}

func TestGenerateCheckInvalidArgs(t *testing.T) {
	t.Parallel()
	g := generateCmd{generateType: "x"}
	err := g.validateArgs()
	ensure.Err(t, err, regexp.MustCompile(`type can only be one of `))
}

func newGenerateCmdHarness(t testing.TB, generateType string) (*Harness, *generateCmd) {
	h := newHarness(t)
	h.makeEmptyRoot()
	g := &generateCmd{generateType}
	return h, g
}

func TestGenerateEjsOutput(t *testing.T) {
	t.Parallel()
	h, g := newGenerateCmdHarness(t, expressEjs)
	defer h.Stop()
	g.run(h.env)

	output := fmt.Sprintf(`Creating directory %s.
Writing out sample file %s
Writing out sample file %s

Almost done! Please add this line to the top of your main.js:

	require('cloud/app.js');

`, filepath.Join(h.env.Root, "cloud", "views"),
		filepath.Join(h.env.Root, "cloud", "app.js"),
		filepath.Join(h.env.Root, "cloud", "views", "hello.ejs"))

	ensure.DeepEqual(t, h.Out.String(), output)
	ensure.DeepEqual(t, h.Err.String(), "")
}

func TestGenerateEjs(t *testing.T) {
	t.Parallel()
	h, g := newGenerateCmdHarness(t, expressEjs)
	defer h.Stop()
	g.run(h.env)

	cloudRoot := filepath.Join(h.env.Root, cloudDir)

	appJs := filepath.Join(cloudRoot, "app.js")
	content, err := ioutil.ReadFile(appJs)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, string(content), sampleAppJS)

	content, err = ioutil.ReadFile(filepath.Join(cloudRoot, "views", "hello.ejs"))
	ensure.Nil(t, err)
	ensure.DeepEqual(t, string(content), helloEJS)
}

func TestGenerateEjsExists(t *testing.T) {
	t.Parallel()
	h, g := newGenerateCmdHarness(t, expressEjs)
	defer h.Stop()
	g.run(h.env)

	ensure.Err(t, g.run(h.env), regexp.MustCompile("Please remove the above existing files and try again."))
}

func TestGenerateJadeOutput(t *testing.T) {
	t.Parallel()
	h, g := newGenerateCmdHarness(t, expressJade)
	defer h.Stop()
	g.run(h.env)

	output := fmt.Sprintf(`Creating directory %s.
Writing out sample file %s
Writing out sample file %s

Almost done! Please add this line to the top of your main.js:

	require('cloud/app.js');

`, filepath.Join(h.env.Root, "cloud", "views"),
		filepath.Join(h.env.Root, "cloud", "app.js"),
		filepath.Join(h.env.Root, "cloud", "views", "hello.jade"))

	ensure.DeepEqual(t, h.Out.String(), output)
	ensure.DeepEqual(t, h.Err.String(), "")
}

func TestGenerateJade(t *testing.T) {
	t.Parallel()
	h, g := newGenerateCmdHarness(t, expressJade)
	defer h.Stop()
	g.run(h.env)

	cloudRoot := filepath.Join(h.env.Root, cloudDir)

	appJs := filepath.Join(cloudRoot, "app.js")
	content, err := ioutil.ReadFile(appJs)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, string(content), strings.Replace(sampleAppJS, "ejs", "jade", -1))

	content, err = ioutil.ReadFile(filepath.Join(cloudRoot, "views", "hello.jade"))
	ensure.Nil(t, err)
	ensure.DeepEqual(t, string(content), helloJade)
}

func TestGenerateJadeExists(t *testing.T) {
	t.Parallel()
	h, g := newGenerateCmdHarness(t, expressJade)
	defer h.Stop()
	g.run(h.env)

	ensure.Err(t, g.run(h.env), regexp.MustCompile("Please remove the above existing files and try again."))
}
