package main

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sync"

	"github.com/facebookgo/errgroup"
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
	var wg errgroup.Group
	maxParallel := make(chan struct{}, maxOpenFD)
	wg.Add(len(files))

	writeFile := func(path string, data []byte) {
		defer func() {
			wg.Done()
			<-maxParallel
		}()

		dir := filepath.Dir(path)
		err := os.MkdirAll(dir, os.ModeDir|os.ModePerm)
		if err != nil {
			wg.Error(stackerr.Wrap(err))
			return
		}
		file, err := os.Create(path)
		defer file.Close()
		if err != nil {
			wg.Error(stackerr.Wrap(err))
			return
		}
		_, err = file.Write(data)
		if err != nil {
			wg.Error(stackerr.Wrap(err))
			return
		}
	}

	for path, data := range files {
		maxParallel <- struct{}{}
		go writeFile(path, data)
	}

	return wg.Wait()
}

func (d *downloadCmd) downloadScripts(e *env, release releaseFiles, files fileMap) error {
	dir := fmt.Sprintf("%s/%s/", e.Root, cloudDir)
	var wg errgroup.Group
	maxParallel := make(chan struct{}, maxOpenFD)
	wg.Add(len(release.UserFiles[cloudDir]))
	var mutex sync.Mutex

	downloadScript := func(file, version string) {
		defer func() {
			wg.Done()
			<-maxParallel
		}()

		v := url.Values{}
		v.Set("checksum", release.Checksums[cloudDir][file])
		v.Set("version", version)
		u := &url.URL{
			Path:     fmt.Sprintf("scripts/%s", file),
			RawQuery: v.Encode(),
		}
		var result string
		_, err := e.ParseAPIClient.Get(u, &result)
		if err != nil {
			wg.Error(stackerr.Wrap(err))
			return
		}
		mutex.Lock()
		files[dir+file] = []byte(result)
		defer mutex.Unlock()
	}

	for file, version := range release.UserFiles[cloudDir] {
		maxParallel <- struct{}{}
		go downloadScript(file, version)
	}

	return wg.Wait()
}

func (d *downloadCmd) downloadHosted(e *env, release releaseFiles, files fileMap) error {
	dir := fmt.Sprintf("%s/%s/", e.Root, hostingDir)
	var wg errgroup.Group
	maxParallel := make(chan struct{}, maxOpenFD)
	wg.Add(len(release.UserFiles[hostingDir]))
	var mutex sync.Mutex

	downloadHosted := func(file, version string) {
		defer func() {
			wg.Done()
			<-maxParallel
		}()

		v := url.Values{}
		v.Set("checksum", release.Checksums[hostingDir][file])
		v.Set("version", version)
		u := &url.URL{
			Path:     fmt.Sprintf("hosted_files/%s", file),
			RawQuery: v.Encode(),
		}
		var result []byte
		_, err := e.ParseAPIClient.Get(u, &result)
		if err != nil {
			wg.Error(stackerr.Wrap(err))
			return
		}
		mutex.Lock()
		files[dir+file] = result
		defer mutex.Unlock()
	}

	for file, version := range release.UserFiles[hostingDir] {
		maxParallel <- struct{}{}
		go downloadHosted(file, version)
	}

	return wg.Wait()
}

func (d *downloadCmd) run(e *env, c *context) error {
	var release releaseFiles

	u := &url.URL{
		Path: "deploy",
	}
	_, err := e.ParseAPIClient.Get(u, &release)
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
