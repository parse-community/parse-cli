package parsecli

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/facebookgo/errgroup"
	"github.com/facebookgo/parse"
	"github.com/facebookgo/stackerr"
)

const (
	SampleSource = `
// Use Parse.Cloud.define to define as many cloud functions as you want.
// For example:
Parse.Cloud.define("hello", function(request, response) {
  response.success("Hello world!");
});
`

	SampleHTML = `
<html>
  <head>
    <title>My ParseApp site</title>
    <style>
    body { font-family: Helvetica, Arial, sans-serif; }
    div { width: 800px; height: 400px; margin: 40px auto; padding: 20px; border: 2px solid #5298fc; }
    h1 { font-size: 30px; margin: 0; }
    p { margin: 40px 0; }
    em { font-family: monospace; }
    a { color: #5298fc; text-decoration: none; }
    </style>
  </head>
  <body>
    <div>
      <h1>Congratulations! You're already hosting with Parse.</h1>
      <p>To get started, edit this file at <em>public/index.html</em> and start adding static content.</p>
      <p>If you want something a bit more dynamic, delete this file and check out <a href="https://parse.com/docs/hosting_guide#webapp">our hosting docs</a>.</p>
    </div>
  </body>
</html>
`
)

func getHostFromURL(urlStr, email string) (string, error) {
	netURL, err := url.Parse(urlStr)
	if err != nil {
		return "", stackerr.Wrap(err)
	}
	server := regexp.MustCompile(`(.*):\d+$`).ReplaceAllString(netURL.Host, "$1")
	if server == "" {
		return "", stackerr.Newf("%s is not a valid url", urlStr)
	}
	if email != "" {
		return fmt.Sprintf("%s#%s", server, email), nil
	}
	return server, nil
}

func Last4(str string) string {
	l := len(str)
	if l > 4 {
		return fmt.Sprintf("%s%s", strings.Repeat("*", l-4), str[l-4:l])
	}
	return str
}

// errorString returns the error string with our without the stack trace
// depending on the Environment variable. this exists because we want plain
// messages for end users, but when we're working on the CLI we want the stack
// trace for debugging.
func ErrorString(e *Env, err error) string {
	type hasUnderlying interface {
		HasUnderlying() error
	}

	parseErr := func(err error) error {
		if apiErr, ok := err.(*parse.Error); ok {
			return errors.New(apiErr.Message)
		}
		return err
	}

	lastErr := func(err error) error {
		if serr, ok := err.(*stackerr.Error); ok {
			if errs := stackerr.Underlying(serr); len(errs) != 0 {
				err = errs[len(errs)-1]
			}
		} else {
			if eu, ok := err.(hasUnderlying); ok {
				err = eu.HasUnderlying()
			}
		}

		return parseErr(err)
	}

	if !e.ErrorStack {
		if merr, ok := err.(errgroup.MultiError); ok {
			var multiError []error
			for _, ierr := range []error(merr) {
				multiError = append(multiError, lastErr(ierr))
			}
			err = errgroup.MultiError(multiError)
		} else {
			err = lastErr(err)
		}
		return parseErr(err).Error()
	}

	return err.Error()
}

func CreateConfigWithContent(path, content string) error {
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

var NewProjectFiles = []struct {
	Dirname, Filename, Content string
}{
	{"cloud", "main.js", SampleSource},
	{"public", "index.html", SampleHTML},
}

func CloneSampleCloudCode(e *Env, dumpTemplate bool) error {
	err := os.MkdirAll(e.Root, 0755)
	if err != nil {
		return stackerr.Wrap(err)
	}

	err = CreateConfigWithContent(
		filepath.Join(e.Root, ParseProject),
		fmt.Sprintf(
			`{
  "project_type" : %d,
  "parse": {"jssdk":""}
}`,
			ParseFormat,
		),
	)
	if err != nil {
		return err
	}
	err = CreateConfigWithContent(
		filepath.Join(e.Root, ParseLocal),
		"{}",
	)
	if err != nil {
		return err
	}

	// no need to set up the template code
	if !dumpTemplate {
		return nil
	}

	for _, info := range NewProjectFiles {
		sampleDir := filepath.Join(e.Root, info.Dirname)
		if _, err := os.Stat(sampleDir); err != nil {
			if !os.IsNotExist(err) {
				return stackerr.Wrap(err)
			}
			if err := os.Mkdir(sampleDir, 0755); err != nil {
				return stackerr.Wrap(err)
			}
		}

		sampleFile := filepath.Join(sampleDir, info.Filename)
		if _, err := os.Stat(sampleFile); err != nil {
			if os.IsNotExist(err) {
				file, err := os.Create(sampleFile)
				if err != nil && !os.IsExist(err) {
					return stackerr.Wrap(err)
				}

				defer file.Close()

				if _, err := file.WriteString(info.Content); err != nil {
					return stackerr.Wrap(err)
				}
				if err := file.Close(); err != nil {
					return stackerr.Wrap(err)
				}
			}
		}
	}
	return nil
}
