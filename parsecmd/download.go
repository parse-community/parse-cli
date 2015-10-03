package parsecmd

import (
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sync/atomic"

	"github.com/ParsePlatform/parse-cli/parsecli"
	"github.com/facebookgo/errgroup"
	"github.com/facebookgo/stackerr"
	"github.com/spf13/cobra"
)

type downloadCmd struct {
	release     *deployInfo
	destination string
	force       bool
}

var errNoFiles = errors.New("Nothing to download. Not yet deployed to the app.")

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
	e *parsecli.Env,
	destination string,
	release *deployInfo) error {
	var wg errgroup.Group

	maxParallel := make(chan struct{}, maxOpenFD)
	numFiles := len(release.Versions.Cloud) + len(release.Versions.Public)
	wg.Add(numFiles)

	var numErrors int32
	moveFile := func(destination, kind, file, checksum string) {
		defer func() {
			wg.Done()
			<-maxParallel
		}()

		err := os.MkdirAll(
			filepath.Join(e.Root, kind, filepath.Dir(file)),
			0755,
		)
		if err != nil {
			atomic.AddInt32(&numErrors, 1)
			wg.Error(stackerr.Wrap(err))
			return
		}

		err = os.Rename(
			filepath.Join(destination, kind, file),
			filepath.Join(e.Root, kind, file),
		)
		if err != nil {
			atomic.AddInt32(&numErrors, 1)
			wg.Error(stackerr.Wrap(err))
			return
		}

		err = d.verifyChecksum(
			filepath.Join(e.Root, kind, file),
			checksum,
		)
		if err != nil {
			atomic.AddInt32(&numErrors, 1)
			wg.Error(err)
			return
		}
	}

	for file, checksum := range release.Checksums.Cloud {
		maxParallel <- struct{}{}
		go moveFile(
			destination,
			parsecli.CloudDir,
			file,
			checksum,
		)
	}
	for file, checksum := range release.Checksums.Public {
		maxParallel <- struct{}{}
		go moveFile(
			destination,
			parsecli.HostingDir,
			file,
			checksum,
		)
	}

	if err := wg.Wait(); err != nil {
		// could not move a single file so no corruption:w
		if int(numErrors) == numFiles {
			fmt.Fprintf(
				e.Out,
				`Failed to download Cloud Code to
 %q
Try "parse download" and manually move contents from
the temporary download location.
`,
				e.Root,
			)
			return nil
		}

		fmt.Fprintf(
			e.Out,
			`Failed to download Cloud Code to
 %q

It might have corrupted contents, due to partially moved files.

Try "parse download" and manually move contents from
the temporary download location.
`,
			e.Root,
		)
		return err
	}
	return nil
}

func (d *downloadCmd) download(e *parsecli.Env, destination string, release *deployInfo) error {
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

		path := path.Join(destination, parsecli.HostingDir, file)
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

		path := path.Join(destination, parsecli.CloudDir, file)
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

func (d *downloadCmd) run(e *parsecli.Env, c *parsecli.Context) error {
	var err error

	latestRelease := d.release
	if latestRelease == nil {
		latestRelease, err = (&deployCmd{}).getPrevDeplInfo(e)
		if err != nil {
			return err
		}
		if len(latestRelease.Versions.Cloud) == 0 && len(latestRelease.Versions.Public) == 0 {
			return errNoFiles
		}
	}

	destination := d.destination
	if destination == "" {
		destination, err = ioutil.TempDir("", "parse_code_")
		if err != nil {
			return stackerr.Wrap(err)
		}
	}

	err = d.download(e, destination, latestRelease)
	if err != nil {
		fmt.Fprintln(e.Err, "Failed to download Cloud Code.")
		return stackerr.Wrap(err)
	}
	if !d.force {
		fmt.Fprintf(e.Out, "Successfully downloaded Cloud Code to %q.\n", destination)
		return nil
	}

	return stackerr.Wrap(d.moveFiles(e, destination, latestRelease))
}

func NewDownloadCmd(e *parsecli.Env) *cobra.Command {
	d := &downloadCmd{}
	cmd := &cobra.Command{
		Use:   "download [app]",
		Short: "Downloads the Cloud Code project",
		Long: `Downloads the Cloud Code project at a given location,
or at a temporary location if nothing is explicitly provided through the -l flag.
`,
		Run: parsecli.RunWithClient(e, d.run),
	}
	cmd.Flags().BoolVarP(&d.force, "force", "f", d.force,
		"Force will overwrite any files in the current project directory")
	cmd.Flags().StringVarP(&d.destination, "location", "l", d.destination,
		"Download Cloud Code project at the given location.")
	return cmd
}
