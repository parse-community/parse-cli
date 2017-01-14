package main

import (
	"fmt"
	"net/http"
	"net/url"
	"runtime"

	"github.com/ParsePlatform/parse-cli/parsecli"
	"github.com/facebookgo/stackerr"
	"github.com/inconshreveable/go-update"
	"github.com/kardianos/osext"
	"github.com/spf13/cobra"
)

const (
	macDownload       = "parse"
	windowsDownload   = "parse.exe"
	linuxDownload     = "parse_linux"
	linuxArmDownload  = "parse_linux_arm"
	downloadURLFormat = "https://github.com/back4app/parse-cli/releases/download/release_%s/%s"
)

type updateCmd struct{}

func (u *updateCmd) latestVersion(e *parsecli.Env) (string, error) {
	v := make(url.Values)
	v.Set("version", "latest")
	req := &http.Request{
		Method: "GET",
		URL:    &url.URL{Path: "supported", RawQuery: v.Encode()},
	}

	var res struct {
		Version string `json:"version"`
	}

	if _, err := e.ParseAPIClient.Do(req, nil, &res); err != nil {
		return "", stackerr.Wrap(err)
	}

	return res.Version, nil
}

func (u *updateCmd) getDownloadURL(e *parsecli.Env) (string, error) {
	ostype := runtime.GOOS
	arch := runtime.GOARCH

	latestVersion, err := u.latestVersion(e)
	if err != nil {
		return "", err
	}
	if latestVersion == "" || latestVersion == parsecli.Version {
		return "", nil
	}

	var downloadURL string
	switch ostype {
	case "darwin":
		downloadURL = fmt.Sprintf(downloadURLFormat, latestVersion, macDownload)
	case "windows":
		downloadURL = fmt.Sprintf(downloadURLFormat, latestVersion, windowsDownload)
	case "linux":
		if arch == "arm" {
			downloadURL = fmt.Sprintf(downloadURLFormat, latestVersion, linuxArmDownload)
		} else {
			downloadURL = fmt.Sprintf(downloadURLFormat, latestVersion, linuxDownload)
		}
	}
	return downloadURL, nil
}

func (u *updateCmd) updateCLI(e *parsecli.Env) (bool, error) {
	downloadURL, err := u.getDownloadURL(e)
	if err != nil {
		return false, err
	}
	if downloadURL == "" {
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
	err = update.Apply(resp.Body, update.Options{TargetPath: exec})
	if err != nil {
		return false, stackerr.Newf("Update failed with error: %v", err)
	}
	fmt.Fprintf(e.Out, "Successfully updated binary at: %s\n", exec)
	return true, nil
}

func (u *updateCmd) run(e *parsecli.Env) error {
	updated, err := u.updateCLI(e)
	if err != nil {
		return err
	}
	if !updated {
		fmt.Fprintf(e.Out, "Already using the latest cli version: %s\n", parsecli.Version)
	}
	return nil
}

func NewUpdateCmd(e *parsecli.Env) *cobra.Command {
	var u updateCmd
	return &cobra.Command{
		Use:   "update",
		Short: "Updates this tool to the latest version",
		Long:  "Updates this tool to the latest version.",
		Run:   parsecli.RunNoArgs(e, u.run),
	}
}
