package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/facebookgo/errgroup"
	"github.com/facebookgo/jsonpipe"
	"github.com/facebookgo/stackerr"
	"github.com/facebookgo/symwalk"
	"github.com/spf13/cobra"
)

const (
	maxOpenFD   = 24
	parseIgnore = ".parseignore"
)

type deployCmd struct {
	Description string
	Force       bool
	Verbose     bool
	Retries     int
	wait        func(int) time.Duration
}

func (d *deployCmd) getSourceFiles(
	dirName string,
	suffixes map[string]struct{},
	e *env,
) ([]string, []string, error) {
	ignoreFile := filepath.Join(e.Root, parseIgnore)

	content, err := ioutil.ReadFile(ignoreFile)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, nil, stackerr.Wrap(err)
		}
		content = nil
	}
	matcher, errors := parseIgnoreMatcher(content)
	if errors != nil && d.Verbose {
		fmt.Fprintf(e.Err,
			"Error compiling the parseignore file:\n%s\n",
			ignoreErrors(errors, e),
		)
	}

	ignoredSet := make(map[string]struct{})
	err = symwalk.Walk(dirName, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			ignoredSet[path] = struct{}{}
		}
		return nil
	})
	if err != nil {
		return nil, nil, stackerr.Wrap(err)
	}

	var selected []string
	errors, err = parseIgnoreWalk(matcher,
		dirName,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			ok := len(suffixes) == 0
			if !ok {
				_, ok = suffixes[filepath.Ext(path)]
			}
			if ok && !info.IsDir() {
				selected = append(selected, path)
				delete(ignoredSet, path)
			}
			return nil
		})
	if err != nil {
		return nil, nil, stackerr.Wrap(err)
	}

	if len(errors) != 0 && d.Verbose {
		fmt.Fprintf(e.Err,
			"Encountered the following errors while matching patterns:\n%s\n",
			ignoreErrors(errors, e),
		)
	}

	var ignored []string
	for file := range ignoredSet {
		ignored = append(ignored, file)
	}
	sort.Strings(selected)
	sort.Strings(ignored)

	return selected, ignored, nil
}

func (d *deployCmd) computeChecksums(files []string,
	normalizeName func(string) string) (map[string]string, error) {
	var wg errgroup.Group
	maxParallel := make(chan struct{}, maxOpenFD)
	wg.Add(len(files))
	var mutex sync.Mutex

	checksums := make(map[string]string)

	computeChecksum := func(name string) {
		defer func() {
			wg.Done()
			<-maxParallel
		}()

		file, err := os.Open(name)
		defer file.Close()
		if err != nil {
			wg.Error(stackerr.Wrap(err))
			return
		}
		h := md5.New()
		if _, err := io.Copy(h, file); err != nil {
			wg.Error(stackerr.Wrap(err))
			return
		}
		if err := file.Close(); err != nil {
			wg.Error(stackerr.Wrap(err))
			return
		}
		mutex.Lock()
		checksums[normalizeName(name)] = fmt.Sprintf("%x", h.Sum(nil))
		defer mutex.Unlock()
	}

	for _, file := range files {
		maxParallel <- struct{}{}
		go computeChecksum(file)
	}

	err := wg.Wait()
	if err != nil {
		return checksums, err
	}

	return checksums, nil
}

func (d *deployCmd) uploadFile(filename, endpoint string, e *env,
	normalizeName func(string) string) (string, error) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(
		"POST",
		endpoint,
		ioutil.NopCloser(
			jsonpipe.Encode(
				map[string]interface{}{
					"name":    normalizeName(filename),
					"content": content,
				},
			),
		),
	)
	if err != nil {
		return "", stackerr.Wrap(err)
	}

	mimeType := mime.TypeByExtension(filepath.Ext(filename))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	req.Header.Add("Content-Type", mimeType)

	var res struct {
		Version string `json:"version"`
	}

	if _, err := e.ParseAPIClient.Do(req, nil, &res); err != nil {
		return "", stackerr.Wrap(err)
	}

	if res.Version == "" {
		return "", stackerr.Newf("Malformed response when trying to upload %s", filename)
	}
	return res.Version, nil
}

type uploader struct {
	DirName       string
	Suffixes      map[string]struct{}
	EndPoint      string
	Env           *env
	PrevChecksums map[string]string
	PrevVersions  map[string]string
}

func (d *deployCmd) uploadSourceFiles(u *uploader) (map[string]string,
	map[string]string, error) {
	sourceFiles, ignoredFiles, err := d.getSourceFiles(filepath.Join(u.Env.Root, u.DirName), u.Suffixes, u.Env)
	if err != nil {
		return nil, nil, err
	}

	namePrefixLen := len(filepath.Join(u.Env.Root, u.DirName, "1")) - 1
	normalizeName := func(name string) string {
		name = filepath.ToSlash(filepath.Clean(name))
		return name[namePrefixLen:]
	}

	currentChecksums, err := d.computeChecksums(sourceFiles, normalizeName)
	if err != nil {
		return nil, nil, err
	}

	var mutex sync.Mutex
	maxParallel := make(chan struct{}, maxOpenFD)
	var wg errgroup.Group
	currentVersions := make(map[string]string)

	uploadFile := func(sourceFile string) {
		defer func() {
			wg.Done()
			<-maxParallel
		}()
		version, err := d.uploadFile(sourceFile, u.EndPoint, u.Env, normalizeName)
		if err != nil {
			wg.Error(err)
			return
		}
		mutex.Lock()
		currentVersions[normalizeName(sourceFile)] = version
		defer mutex.Unlock()
	}
	changed := false
	var changedFiles []string
	for _, sourceFile := range sourceFiles {
		if !d.Force { // if not forced, verify changed content using checksums
			name := normalizeName(sourceFile)
			var noUpload bool
			if prevChecksum, ok := u.PrevChecksums[name]; ok {
				noUpload = prevChecksum == currentChecksums[name]
			}
			if prevVersion, ok := u.PrevVersions[name]; ok && noUpload {
				currentVersions[name] = prevVersion
				continue
			}
		}
		changed = true
		wg.Add(1)

		changedFiles = append(changedFiles, sourceFile)

		maxParallel <- struct{}{}
		go uploadFile(sourceFile)
	}
	if changed && d.Verbose {
		var message string
		switch u.DirName {
		case "cloud":
			message = "scripts"
		case "public":
			message = "hosting"
		}
		fmt.Fprintf(u.Env.Out,
			`Uploading recent changes to %s...
The following files will be uploaded:
%s
`,
			message,
			strings.Join(changedFiles, "\n"),
		)
		if len(ignoredFiles) != 0 {
			fmt.Fprintln(u.Env.Out, "The following files will be ignored:")
			for _, file := range ignoredFiles {
				fmt.Fprintln(u.Env.Out, file)
			}
		}
	}
	if err := wg.Wait(); err != nil {
		return nil, nil, err
	}
	return currentChecksums, currentVersions, nil
}

type deployFileData struct {
	Cloud  map[string]string `json:"cloud"`
	Public map[string]string `json:"public"`
}

type deployInfo struct {
	ReleaseName  string         `json:"releaseName,omitempty"`
	Description  string         `json:"description,omitempty"`
	ParseVersion string         `json:"parseVersion,omitempty"`
	Checksums    deployFileData `json:"checksums,omitempty"`
	Versions     deployFileData `json:"userFiles,omitempty"`
	Warning      string         `json:"warning,omitempty"`
}

func (d *deployCmd) makeNewRelease(info *deployInfo, e *env) (deployInfo, error) {
	var res deployInfo
	u := url.URL{
		Path: "deploy",
	}
	_, err := e.ParseAPIClient.Post(&u, info, &res)
	if err != nil {
		return res, stackerr.Wrap(err)
	}
	return res, nil
}

func (d *deployCmd) getPrevDeplInfo(e *env) (*deployInfo, error) {
	prevDeplInfo := &deployInfo{}
	if _, err := e.ParseAPIClient.Get(&url.URL{Path: "deploy"}, prevDeplInfo); err != nil {
		return nil, stackerr.Wrap(err)
	}
	legacy := len(prevDeplInfo.Checksums.Cloud) == 0 &&
		len(prevDeplInfo.Checksums.Public) == 0 &&
		len(prevDeplInfo.Versions.Cloud) == 0 &&
		len(prevDeplInfo.Versions.Public) == 0
	if legacy {
		var res struct {
			ReleaseName  string            `json:"releaseName,omitempty"`
			Description  string            `json:"description,omitempty"`
			ParseVersion string            `json:"parseVersion,omitempty"`
			Checksums    map[string]string `json:"checksums,omitempty"`
			Versions     map[string]string `json:"userFiles,omitempty"`
		}
		if _, err := e.ParseAPIClient.Get(&url.URL{Path: "deploy"}, &res); err != nil {
			return nil, stackerr.Wrap(err)
		}
		prevDeplInfo.ReleaseName = res.ReleaseName
		prevDeplInfo.Description = res.Description
		prevDeplInfo.ParseVersion = res.ParseVersion
		prevDeplInfo.Checksums.Cloud = res.Checksums
		prevDeplInfo.Versions.Cloud = res.Versions
	}
	return prevDeplInfo, nil
}

func (d *deployCmd) deploy(
	parseVersion string,
	prevDeplInfo *deployInfo,
	forDevelop bool,
	e *env) (*deployInfo, error) {
	if parseVersion == "" {
		fmt.Fprintln(e.Err,
			"JS SDK version not set, setting it to latest available JS SDK version",
		)
		if err := useLatestJSSDK(e); err != nil {
			return nil, err
		}
	}

	if d.Verbose {
		fmt.Fprintln(e.Out, "Uploading source files")
	}
	if prevDeplInfo == nil {
		var err error
		prevDeplInfo, err = d.getPrevDeplInfo(e)
		if err != nil {
			return nil, err
		}
	}

	scriptChecksums, scriptVersions, err := d.uploadSourceFiles(&uploader{
		DirName: "cloud",
		Suffixes: map[string]struct{}{
			".js":   {},
			".ejs":  {},
			".jade": {},
		},
		EndPoint:      "scripts",
		PrevChecksums: prevDeplInfo.Checksums.Cloud,
		PrevVersions:  prevDeplInfo.Versions.Cloud,
		Env:           e})
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	hostedChecksums, hostedVersions, err := d.uploadSourceFiles(&uploader{
		DirName:       "public",
		Suffixes:      map[string]struct{}{},
		EndPoint:      "hosted_files",
		PrevChecksums: prevDeplInfo.Checksums.Public,
		PrevVersions:  prevDeplInfo.Versions.Public,
		Env:           e})
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	if len(scriptChecksums) == 0 && len(hostedChecksums) == 0 {
		return nil, stackerr.New("No files to upload")
	}

	if d.Verbose {
		fmt.Fprintln(e.Out, "Finished uploading files")
	}

	noDiff := reflect.DeepEqual(scriptChecksums, prevDeplInfo.Checksums.Cloud) &&
		reflect.DeepEqual(scriptVersions, prevDeplInfo.Versions.Cloud) &&
		reflect.DeepEqual(hostedChecksums, prevDeplInfo.Checksums.Public) &&
		reflect.DeepEqual(hostedVersions, prevDeplInfo.Versions.Public)

	noDiff = noDiff && (parseVersion == prevDeplInfo.ParseVersion)

	if noDiff {
		if d.Verbose {
			fmt.Fprintln(e.Out, "Not creating a release because no files have changed")
		}
		return prevDeplInfo, nil
	}

	if parseVersion == "" {
		parseVersion = prevDeplInfo.ParseVersion
	}

	newDeployInfo := &deployInfo{
		ParseVersion: parseVersion,
		Checksums:    deployFileData{Cloud: scriptChecksums, Public: hostedChecksums},
		Versions:     deployFileData{Cloud: scriptVersions, Public: hostedVersions},
		Description:  d.Description,
	}

	res, err := d.makeNewRelease(newDeployInfo, e)
	if err != nil {
		if forDevelop {
			// if the release failed but we are in develop mode, we want to return
			// the old release information with new checksums so we do not keep
			// uploading a broken release.
			prevDeplInfo.Checksums = newDeployInfo.Checksums
			return prevDeplInfo, err
		}
		return nil, err
	}

	if forDevelop {
		fmt.Fprintln(e.Out, "Your changes are now live.")
	} else {
		if res.Warning != "" {
			fmt.Fprintln(e.Err, res.Warning)
		}
		fmt.Fprintf(e.Out, "New release is named %s (using Parse JavaScript SDK v%s)\n", res.ReleaseName, res.ParseVersion)
	}

	return &deployInfo{
		ParseVersion: res.ParseVersion,
		Checksums:    newDeployInfo.Checksums,
		Versions:     newDeployInfo.Versions,
	}, nil
}

func (d *deployCmd) handleError(
	n int,
	err, prevErr error,
	e *env,
) error {
	if err == nil {
		return nil
	}
	if n == d.Retries-1 {
		return err
	}

	var waitTime time.Duration
	if d.wait != nil {
		waitTime = d.wait(n)
	}

	errStr := errorString(e, err)
	if prevErr != nil {
		prevErrStr := errorString(e, prevErr)
		if prevErrStr == errStr {
			fmt.Fprintf(
				e.Err,
				"Sorry, deploy failed again with same error.\nWill retry in %d seconds.\n\n",
				waitTime/time.Second,
			)
			time.Sleep(waitTime)
			return nil
		}
	}

	fmt.Fprintf(
		e.Err,
		"Deploy failed with error:\n%s\nWill retry in %d seconds.\n\n",
		errStr,
		waitTime/time.Second,
	)
	time.Sleep(waitTime)
	return nil
}

func (d *deployCmd) run(e *env, c *context) error {
	var prevErr error
	for i := 0; i < d.Retries; i++ {
		parseVersion := c.Config.getProjectConfig().Parse.JSSDK
		newDeployInfo, err := d.deploy(parseVersion, nil, false, e)
		if err == nil {
			if parseVersion == "" && newDeployInfo != nil && newDeployInfo.ParseVersion != "" {
				c.Config.getProjectConfig().Parse.JSSDK = newDeployInfo.ParseVersion
				return storeProjectConfig(e, c.Config)
			}
			return nil
		}
		if err := d.handleError(i, err, prevErr, e); err != nil {
			return err
		}
		prevErr = err
	}

	return nil
}

func newDeployCmd(e *env) *cobra.Command {
	d := deployCmd{
		Verbose: true,
		Retries: 3,
		wait:    func(n int) time.Duration { return time.Duration(n) * time.Second },
	}

	cmd := &cobra.Command{
		Use:   "deploy [app]",
		Short: "Deploys a Parse App",
		Long:  `Deploys the code to the given app.`,
		Run:   runWithClient(e, d.run),
	}
	cmd.Flags().StringVarP(&d.Description, "description", "d", d.Description,
		"Add an optional description to the deploy")
	cmd.Flags().BoolVarP(&d.Force, "force", "f", d.Force,
		"Force deploy files even if their content is unchanged")
	cmd.Flags().BoolVarP(&d.Verbose, "verbose", "v", d.Verbose,
		"Control verbosity of cmd line logs")
	cmd.Flags().IntVarP(&d.Retries, "retries", "n", d.Retries,
		"Max number of retries to perform until first successful deploy")
	return cmd
}
