package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"path/filepath"

	"github.com/facebookgo/errgroup"
	"github.com/facebookgo/stackerr"
	"github.com/spf13/cobra"
)

type downloadCmd struct {
	force   bool
	release *deployInfo
}

func (d *downloadCmd) verifyChecksum(path, checksum string) error {
	file, err := os.Open(path)
	if err != nil {
		return stackerr.Wrap(err)
	}
	defer file.Close()

	h := md5.New()
	_, err = io.Copy(h, file)
	if err != nil {
		return stackerr.Wrap(err)
	}

	sum := fmt.Sprintf("%x", h.Sum(nil))
	if sum != checksum {
		return stackerr.Newf("Invalid checksum for %s", path)
	}

	return nil
}

func (d *downloadCmd) moveFiles(
	e *env,
	tempDir string,
	release *deployInfo) error {
	var wg errgroup.Group

	maxParallel := make(chan struct{}, maxOpenFD)
	wg.Add(len(release.Versions.Cloud) + len(release.Versions.Public))

	moveFile := func(tempDir, kind, file, checksum string) {
		defer func() {
			wg.Done()
			<-maxParallel
		}()

		err := os.MkdirAll(
			filepath.Join(e.Root, kind, filepath.Dir(file)),
			0755,
		)
		if err != nil {
			wg.Error(stackerr.Wrap(err))
			return
		}

		err = os.Rename(
			filepath.Join(tempDir, kind, file),
			filepath.Join(e.Root, kind, file),
		)
		if err != nil {
			wg.Error(stackerr.Wrap(err))
			return
		}

		err = d.verifyChecksum(
			filepath.Join(e.Root, kind, file),
			checksum,
		)
		if err != nil {
			wg.Error(err)
			return
		}
	}

	for file, checksum := range release.Checksums.Cloud {
		maxParallel <- struct{}{}
		go moveFile(
			tempDir,
			cloudDir,
			file,
			checksum,
		)
	}
	for file, checksum := range release.Checksums.Public {
		maxParallel <- struct{}{}
		go moveFile(
			tempDir,
			hostingDir,
			file,
			checksum,
		)
	}
	return wg.Wait()
}

func (d *downloadCmd) download(e *env, tempDir string, release *deployInfo) error {
	var wg errgroup.Group
	maxParallel := make(chan struct{}, maxOpenFD)
	wg.Add(len(release.Versions.Cloud) + len(release.Versions.Public))

	downloadHosted := func(file, version, checksum string) {
		defer func() {
			wg.Done()
			<-maxParallel
		}()

		v := make(url.Values)
		v.Set("version", version)
		v.Set("checksum", checksum)
		u := &url.URL{
			Path:     path.Join("hosted_files", file),
			RawQuery: v.Encode(),
		}
		var content []byte
		_, err := e.ParseAPIClient.Get(u, &content)
		if err != nil {
			wg.Error(err)
			return
		}

		path := path.Join(tempDir, hostingDir, file)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			wg.Error(stackerr.Wrap(err))
			return
		}
		if err := ioutil.WriteFile(path, content, 0644); err != nil {
			wg.Error(stackerr.Wrap(err))
			return
		}
		if err := d.verifyChecksum(path, checksum); err != nil {
			wg.Error(err)
			return
		}
	}

	downloadScript := func(file, version, checksum string) {
		defer func() {
			wg.Done()
			<-maxParallel
		}()

		v := make(url.Values)
		v.Set("version", version)
		v.Set("checksum", checksum)
		u := &url.URL{
			Path:     path.Join("scripts", file),
			RawQuery: v.Encode(),
		}
		var content string
		_, err := e.ParseAPIClient.Get(u, &content)
		if err != nil {
			wg.Error(err)
			return
		}

		path := path.Join(tempDir, cloudDir, file)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			wg.Error(stackerr.Wrap(err))
			return
		}
		if err := ioutil.WriteFile(path, []byte(content), 0644); err != nil {
			wg.Error(stackerr.Wrap(err))
			return
		}
	}

	for file, version := range release.Versions.Public {
		checksum, ok := release.Checksums.Public[file]
		if !ok {
			continue
		}
		maxParallel <- struct{}{}
		go downloadHosted(file, version, checksum)
	}

	for file, version := range release.Versions.Cloud {
		checksum, ok := release.Checksums.Cloud[file]
		if !ok {
			continue
		}
		maxParallel <- struct{}{}
		go downloadScript(file, version, checksum)
	}

	return wg.Wait()
}

func (d *downloadCmd) run(e *env, c *context) error {
	if d.release == nil {
		latestRelease, err := (&deployCmd{}).getPrevDeplInfo(e)
		if err != nil {
			return err
		}
		d.release = latestRelease
	}

	tempDir, err := ioutil.TempDir("", "parse_code_")
	if err != nil {
		return stackerr.Wrap(err)
	}

	err = d.download(e, tempDir, d.release)
	if err != nil {
		fmt.Fprintln(e.Err, "Failed to download Cloud Code.")
		return stackerr.Wrap(err)
	}
	if !d.force {
		fmt.Fprintf(e.Out, "Successfully downloaded Cloud Code to %q.\n", tempDir)
		return nil
	}

	err = d.moveFiles(e, tempDir, d.release)
	if err != nil {
		fmt.Fprintf(
			e.Out,
			`Failed to download Cloud Code to %q.
Sorry! but %s might have corrupted contents.
If you want to download Cloud Code from Parse,
try again without the "-f" option.
`,
			e.Root,
			e.Root,
		)
		return stackerr.Wrap(err)
	}
	return nil
}

func newDownloadCmd(e *env) *cobra.Command {
	d := &downloadCmd{}
	cmd := &cobra.Command{
		Use:   "download [app]",
		Short: "Downloads the Cloud Code project",
		Long:  "Downloads the Cloud Code project at a temporary location.",
		Run:   runWithClient(e, d.run),
	}
	cmd.Flags().BoolVarP(&d.force, "force", "f", d.force,
		"Force will overwrite any content in the current project directory")
	return cmd
}
