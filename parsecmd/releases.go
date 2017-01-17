package parsecmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/ParsePlatform/parse-cli/parsecli"
	"github.com/facebookgo/stackerr"
	"github.com/spf13/cobra"
)

type userFiles struct {
	Cloud  map[string]interface{} `json:"cloud"`
	Public map[string]interface{} `json:"public"`
}

type releasesResponse struct {
	Version     string `json:"version"`
	Description string `json:"description"`
	Timestamp   string `json:"timestamp"`
	UserFiles   string `json:"userFiles"`
}

type releasesCmd struct {
	version string
}

func (r *releasesCmd) printFileNames(
	fileVersions map[string]interface{},
	e *parsecli.Env) {
	var files []string
	for name := range fileVersions {
		files = append(files, name)
	}
	sort.Strings(files)
	fmt.Fprintln(e.Out, strings.Join(files, "\n"))
}

func (r *releasesCmd) printFiles(version string,
	releases []releasesResponse,
	e *parsecli.Env) error {
	var files string
	for _, release := range releases {
		if release.Version == version {
			files = release.UserFiles
			break
		}
	}
	if files == "" {
		return stackerr.Newf(`Unable to fetch files for release version: %s
Note that you can list files for all releases shown in "back4app releases"`,
			version)
	}
	var versionFileNames userFiles
	if err := json.NewDecoder(
		strings.NewReader(files),
	).Decode(&versionFileNames); err != nil {
		return stackerr.Wrap(err)
	}
	if len(versionFileNames.Cloud) != 0 {
		fmt.Fprintf(e.Out, "Deployed Cloud Code files:\n")
		r.printFileNames(versionFileNames.Cloud, e)
	}
	if len(versionFileNames.Cloud) != 0 && len(versionFileNames.Public) != 0 {
		fmt.Fprintln(e.Out)
	}
	if len(versionFileNames.Public) != 0 {
		fmt.Fprintf(e.Out, "Deployed public hosting files:\n")
		r.printFileNames(versionFileNames.Public, e)
	}
	return nil
}

func (r *releasesCmd) run(e *parsecli.Env, c *parsecli.Context) error {
	u := &url.URL{
		Path: "releases",
	}
	var releasesList []releasesResponse
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
		fmt.Fprintf(w, "%s\t%s\t%s\n", release.Version, description, release.Timestamp)
	}
	w.Flush()
	return nil
}

func NewReleasesCmd(e *parsecli.Env) *cobra.Command {
	r := &releasesCmd{}
	cmd := &cobra.Command{
		Use:   "releases [app]",
		Short: "Gets the releases for a Parse App",
		Long:  "Prints the releases the server knows about.",
		Run:   parsecli.RunWithClient(e, r.run),
	}
	cmd.Flags().StringVarP(&r.version, "version", "v", r.version,
		"List files names of the deployed version.")
	return cmd
}
