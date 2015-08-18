package main

import (
	"fmt"
	"net/http"
	"net/url"
	"path"
	"runtime"
	"strings"

	"github.com/facebookgo/stackerr"
	"github.com/inconshreveable/go-update"
	"github.com/kardianos/osext"
	"github.com/spf13/cobra"
)

const (
	macCliDownloadURL     = "https://parse.com/downloads/cloud_code/cli/parse-osx/latest"
	unixCliDownloadURL    = "https://parse.com/downloads/cloud_code/cli/parse-linux/latest"
	windowsCliDownloadURL = "https://parse.com/downloads/cloud_code/cli/parse-windows/latest"
)

type updateCmd struct{}

func (u *updateCmd) latestVersion(e *env, downloadURL string) (string, error) {
	dURL, err := url.Parse(downloadURL)
	if err != nil {
		return "", stackerr.Wrap(err)
	}

	resp, err := e.Client.Do(&http.Request{Method: "HEAD", URL: dURL}, nil, nil)
	if err != nil {
		return "", nil // if unable to fetch latest cli version, do not abort!
	}

	base := path.Base(resp.Header.Get("Location"))
	base = strings.TrimSuffix(base, ".exe")
	//parse-os-2.0.2
	splits := strings.Split(base, "-")

	return splits[len(splits)-1], nil
}

func (u *updateCmd) updateCLI(e *env) (bool, error) {
	downloadURL := unixCliDownloadURL
	switch runtime.GOOS {
	case "windows":
		downloadURL = windowsCliDownloadURL
	case "darwin":
		downloadURL = macCliDownloadURL
	}

	latestVersion, err := u.latestVersion(e, downloadURL)
	if err != nil {
		return false, stackerr.Wrap(err)
	}

	if latestVersion == "" || latestVersion == version {
		return false, nil
	}

	exec, err := osext.Executable()
	if err != nil {
		return false, stackerr.Wrap(err)
	}

	fmt.Fprintf(e.Out, "Downloading binary from %s.\n", downloadURL)
	resp, err := http.Get(downloadURL)
	if err != nil {
		return false, stackerr.Newf("Update failed with error: %v", err)
	}
	defer resp.Body.Close()
	err = update.Apply(resp.Body, &update.Options{TargetPath: exec})
	if err != nil {
		return false, stackerr.Newf("Update failed with error: %v", err)
	}
	fmt.Fprintf(e.Out, "Successfully updated binary at: %s\n", exec)
	return true, nil
}

func (u *updateCmd) run(e *env) error {
	updated, err := u.updateCLI(e)
	if err != nil {
		return err
	}
	if !updated {
		fmt.Fprintf(e.Out, "Already using the latest cli version: %s\n", version)
	}
	return nil
}

func newUpdateCmd(e *env) *cobra.Command {
	var u updateCmd
	return &cobra.Command{
		Use:   "update",
		Short: "Updates this tool to the latest version",
		Long:  "Updates this tool to the latest version.",
		Run:   runNoArgs(e, u.run),
	}
}
