package main

import (
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"

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

func validateChecksum(path, checksum string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	hash := md5.New()
	_, err = io.Copy(hash, file)
	if err != nil {
		return err
	}

	sum := fmt.Sprintf("%x", hash.Sum(nil))
	if sum != checksum {
		return errors.New("Invalid checksum")
	}

	return nil
}

func (d *downloadCmd) moveFiles(e *env, tempDir string, release releaseFiles) error {
	var wg errgroup.Group
	maxParallel := make(chan struct{}, maxOpenFD)
	wg.Add(len(release.UserFiles[cloudDir]) + len(release.UserFiles[hostingDir]))

	moveFile := func(oldPath, newPath, checksum string) {
		defer func() {
			wg.Done()
			<-maxParallel
		}()

		err := validateChecksum(oldPath, checksum)
		if err != nil {
			wg.Error(err)
			return
		}
		err = os.Rename(oldPath, newPath)
		if err != nil {
			wg.Error(err)
			return
		}
		err = validateChecksum(newPath, checksum)
		if err != nil {
			wg.Error(err)
			return
		}
	}

	oldDir := fmt.Sprintf("%s/%s/", tempDir, cloudDir)
	newDir := fmt.Sprintf("%s/%s/", e.Root, cloudDir)
	for file, checksum := range release.Checksums[cloudDir] {
		maxParallel <- struct{}{}
		go moveFile(oldDir+file, newDir+file, checksum)
	}

	oldDir = fmt.Sprintf("%s/%s/", tempDir, hostingDir)
	newDir = fmt.Sprintf("%s/%s/", e.Root, hostingDir)
	for file, checksum := range release.Checksums[hostingDir] {
		maxParallel <- struct{}{}
		go moveFile(oldDir+file, newDir+file, checksum)
	}

	return wg.Wait()
}

func writeFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	err := os.MkdirAll(dir, os.ModeDir|os.ModePerm)
	if err != nil {
		return err
	}
	file, err := os.Create(path)
	defer file.Close()
	if err != nil {
		return err
	}
	_, err = file.Write(data)
	if err != nil {
		return err
	}
	return nil
}

func (d *downloadCmd) download(e *env, tempDir string, release releaseFiles) error {
	var wg errgroup.Group
	maxParallel := make(chan struct{}, maxOpenFD)
	wg.Add(len(release.UserFiles[cloudDir]) + len(release.UserFiles[hostingDir]))

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
			wg.Error(err)
			return
		}

		path := fmt.Sprintf("%s/%s/%s", tempDir, hostingDir, file)
		err = writeFile(path, result)
		if err != nil {
			wg.Error(err)
			return
		}
	}

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
			wg.Error(err)
			return
		}

		path := fmt.Sprintf("%s/%s/%s", tempDir, cloudDir, file)
		err = writeFile(path, []byte(result))
		if err != nil {
			wg.Error(err)
			return
		}
	}

	for file, version := range release.UserFiles[hostingDir] {
		maxParallel <- struct{}{}
		go downloadHosted(file, version)
	}

	for file, version := range release.UserFiles[cloudDir] {
		maxParallel <- struct{}{}
		go downloadScript(file, version)
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

	tempDir, err := ioutil.TempDir("", "parse-cli-download")
	if err != nil {
		return stackerr.Wrap(err)
	}

	err = d.download(e, tempDir, release)
	if err != nil {
		return stackerr.Wrap(err)
	}

	err = d.moveFiles(e, tempDir, release)
	if err != nil {
		return stackerr.Wrap(err)
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
