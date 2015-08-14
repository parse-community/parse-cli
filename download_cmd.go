package main

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/facebookgo/stackerr"
	"github.com/spf13/cobra"
)

type downloadCmd struct{}

type releaseFiles struct {
	UserFiles map[string]map[string]string `json:"userFiles"`
	Checksums map[string]map[string]string `json:"checksums"`
}

type fileMap map[string][]byte

func (d *downloadCmd) writeFiles(files fileMap) error {
	for path, data := range files {
		dir := filepath.Dir(path)
		err := os.MkdirAll(dir, os.ModeDir|os.ModePerm)
		if err != nil {
			return stackerr.Wrap(err)
		}
		file, err := os.Create(path)
		defer file.Close()
		if err != nil {
			return stackerr.Wrap(err)
		}
		_, err = file.Write(data)
		if err != nil {
			return stackerr.Wrap(err)
		}
	}
	return nil
}

func (d *downloadCmd) downloadScripts(e *env, release releaseFiles, files fileMap) error {
	dir := fmt.Sprintf("%s/%s/", e.Root, cloudDir)
	for file, version := range release.UserFiles[cloudDir] {
		v := url.Values{}
		v.Set("checksum", release.Checksums[cloudDir][file])
		v.Set("version", version)
		u := &url.URL{
			Path:     fmt.Sprintf("scripts/%s", file),
			RawQuery: v.Encode(),
		}
		var result string
		_, err := e.Client.Get(u, &result)
		if err != nil {
			return stackerr.Wrap(err)
		}
		files[dir+file] = []byte(result)
	}
	return nil
}

func (d *downloadCmd) downloadHosted(e *env, release releaseFiles, files fileMap) error {
	dir := fmt.Sprintf("%s/%s/", e.Root, hostingDir)
	for file, version := range release.UserFiles[hostingDir] {
		v := url.Values{}
		v.Set("checksum", release.Checksums[hostingDir][file])
		v.Set("version", version)
		u := &url.URL{
			Path:     fmt.Sprintf("hosted_files/%s", file),
			RawQuery: v.Encode(),
		}
		var result []byte
		_, err := e.Client.Get(u, &result)
		if err != nil {
			return stackerr.Wrap(err)
		}
		files[dir+file] = result
	}
	return nil
}

func (d *downloadCmd) run(e *env, c *client) error {
	var release releaseFiles

	u := &url.URL{
		Path: "deploy",
	}
	_, err := e.Client.Get(u, &release)
	if err != nil {
		return stackerr.Wrap(err)
	}

	files := make(fileMap)

	err = d.downloadHosted(e, release, files)
	if err != nil {
		return err
	}

	err = d.downloadScripts(e, release, files)
	if err != nil {
		return err
	}

	err = d.writeFiles(files)
	if err != nil {
		return err
	}

	fmt.Fprintln(e.Out, "Download successful.")
	return nil
}

func newDownloadCmd(e *env) *cobra.Command {
	d := &downloadCmd{}
	cmd := &cobra.Command{
		Use:   "download [app]",
		Short: "Downloads the app files.",
		Long:  "Downloads the cloud code for a Parse App.",
		Run:   runWithClient(e, d.run),
	}
	return cmd
}
