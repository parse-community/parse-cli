package herokucmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/ParsePlatform/parse-cli/parsecli"
	"github.com/facebookgo/stackerr"
)

const (
	nodeSampleURL  = "https://github.com/pavanka/parse-webhooks/releases/download/3.0.0/node-sample.zip"
	nodeSampleBase = "node-sample"
)

func recordDecision(e *parsecli.Env, decision string) {
	v := make(url.Values)
	v.Set("version", parsecli.Version)
	v.Set("decision", decision)
	req := &http.Request{
		Method: "GET",
		URL:    &url.URL{Path: "supported", RawQuery: v.Encode()},
	}
	e.ParseAPIClient.Do(req, nil, nil)
}

func PromptCreateWebhooks(e *parsecli.Env) (string, error) {
	selections := map[int]string{
		1: "Heroku (https://www.heroku.com)",
		2: "Back4App  (https://parse.com/docs/cloudcode/guide)",
	}

	msg := "Sorry! CLI supports only Parse Cloud Code and Heroku."

	projectType := 0
	for i := 0; i < 3; i++ {
		fmt.Fprintf(
			e.Out,
			`Which of these providers would you like use for running your server code:
%d) %s
%d) %s
Type 1 or 2 to make a selection: `,
			1, selections[1],
			2, selections[2],
		)
		fmt.Fscanf(e.In, "%d\n", &projectType)
		fmt.Fprintln(e.Out)
		switch projectType {
		case 1:
			recordDecision(e, "heroku")
			return "heroku", nil
		case 2:
			recordDecision(e, "parse")
			return "parse", nil
		}
		fmt.Fprintln(e.Err, msg)
	}
	return "", stackerr.New(msg)
}

func writeHerokuConfigs(e *parsecli.Env) error {
	err := parsecli.CreateConfigWithContent(
		filepath.Join(e.Root, parsecli.ParseProject),
		fmt.Sprintf(
			`{"project_type": %d}`,
			parsecli.HerokuFormat,
		),
	)
	if err != nil {
		return err
	}
	return parsecli.CreateConfigWithContent(
		filepath.Join(e.Root, parsecli.ParseLocal),
		"{}",
	)
}

func downloadNodeSample(e *parsecli.Env) (string, error) {
	tmpZip, err := ioutil.TempFile("", "parse-node-sample")
	if err != nil {
		return "", stackerr.Wrap(err)
	}

	resp, err := http.Get(nodeSampleURL)
	if err != nil {
		return "", stackerr.Wrap(err)
	}

	_, err = io.Copy(tmpZip, resp.Body)
	if err != nil {
		return "", stackerr.Wrap(err)
	}
	return tmpZip.Name(), nil
}

func setupNodeSample(e *parsecli.Env, dumpTemplate bool) error {
	if !dumpTemplate {
		err := os.MkdirAll(e.Root, 0755)
		if err != nil {
			return stackerr.Wrap(err)
		}
		return writeHerokuConfigs(e)
	}

	sampleCode, err := downloadNodeSample(e)
	if err != nil {
		return err
	}
	defer os.RemoveAll(sampleCode)

	err = unzip(sampleCode, nodeSampleBase, e.Root)
	if err != nil {
		return err
	}
	return writeHerokuConfigs(e)
}

func CloneNodeCode(e *parsecli.Env, isNew, onlyConfig bool, appConfig parsecli.AppConfig) (bool, error) {
	cloneTemplate := false
	if !isNew && !onlyConfig {
		authToken, err := appConfig.GetApplicationAuth(e)
		if err != nil {
			return false, err
		}
		herokuAppConfig, ok := appConfig.(*parsecli.HerokuAppConfig)
		if !ok {
			return false, stackerr.New("invalid heroku app config")
		}

		var gitURL string
		g := &gitInfo{}

		herokuAppName, err := parsecli.FetchHerokuAppName(herokuAppConfig.HerokuAppID, e)
		if err != nil {
			return false, err
		}
		gitURL = fmt.Sprintf("https://:%s@git.heroku.com/%s.git", authToken, herokuAppName)
		err = g.clone(gitURL, e.Root)
		if err != nil {
			fmt.Fprintf(e.Err, `Failed to fetch the latest deployed code from Heroku.
Please try "git clone %s %s".
Currently cloning the template project.
`,
				gitURL,
				e.Root,
			)
			cloneTemplate = true
		} else {
			isEmpty, err := g.isEmptyRepository(e.Root)
			if err != nil {
				return false, err
			}
			if isEmpty {
				if err := os.RemoveAll(e.Root); err != nil {
					return false, stackerr.Wrap(err)
				}
				cloneTemplate = true
			}
		}
	}
	cloneTemplate = (isNew || cloneTemplate) && !onlyConfig
	return cloneTemplate, setupNodeSample(e, cloneTemplate)
}
