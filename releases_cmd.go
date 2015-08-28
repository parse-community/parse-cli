package main

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/facebookgo/stackerr"
	"github.com/spf13/cobra"
)

type releasesCmd struct {
	version string
}

func (r *releasesCmd) printFileNames(
	fileVersions map[string]string,
	e *env) {
	var files []string
	for name := range fileVersions {
		files = append(files, name)
	}
	sort.Strings(files)
	fmt.Fprintln(e.Out, strings.Join(files, "\n"))
}

func (r *releasesCmd) printFiles(
	version string,
	releases []deployInfo,
	e *env) error {
	var files deployFileData
	for _, release := range releases {
		if release.ParseVersion == version {
			files = release.Versions
			break
		}
	}
	if len(files.Cloud) == 0 && len(files.Public) == 0 {
		return stackerr.Newf(`Unable to fetch files for release version: %s
Note that you can list files for all releases shown in "parse releases"`,
			version)
	}
	if len(files.Cloud) != 0 {
		fmt.Fprintf(e.Out, "Deployed cloud code files:\n")
		r.printFileNames(files.Cloud, e)
	}
	if len(files.Cloud) != 0 && len(files.Public) != 0 {
		fmt.Fprintln(e.Out)
	}
	if len(files.Public) != 0 {
		fmt.Fprintf(e.Out, "Deployed public hosting files:\n")
		r.printFileNames(files.Public, e)
	}
	return nil
}

func (r *releasesCmd) run(e *env, c *context) error {
	u := &url.URL{
		Path: "releases",
	}
	var releasesList []deployInfo
	if _, err := e.ParseAPIClient.Get(u, &releasesList); err != nil {
		return stackerr.Wrap(err)
	}

	if r.version != "" {
		return r.printFiles(r.version, releasesList, e)
	}

	w := new(tabwriter.Writer)
	w.Init(e.Out, 32, 8, 0, ' ', 0)
	fmt.Fprintln(w, "Name\tDescription\tDate")
	for _, release := range releasesList {
		description := "No release notes given"
		if release.Description != "" {
			description = release.Description
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", release.ParseVersion, description, release.Timestamp)
	}
	w.Flush()
	return nil
}

func newReleasesCmd(e *env) *cobra.Command {
	r := &releasesCmd{}
	cmd := &cobra.Command{
		Use:   "releases [app]",
		Short: "Gets the releases for a Parse App",
		Long:  "Prints the releases the server knows about.",
		Run:   runWithClient(e, r.run),
	}
	cmd.Flags().StringVarP(&r.version, "version", "v", r.version,
		"List files names of the deployed version.")
	return cmd
}
