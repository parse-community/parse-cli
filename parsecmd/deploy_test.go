package parsecmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/ParsePlatform/parse-cli/parsecli"
	"github.com/facebookgo/ensure"
	"github.com/facebookgo/parse"
)

func jsonStr(t testing.TB, v interface{}) string {
	b, err := json.Marshal(v)
	ensure.Nil(t, err)
	return string(b)
}

func createParseProject(t testing.TB) *parsecli.Harness {
	h := parsecli.NewHarness(t)
	h.MakeEmptyRoot()

	ensure.Nil(t, os.Mkdir(filepath.Join(h.Env.Root, "parse"), 0755))
	h.Env.Root = filepath.Join(h.Env.Root, "parse")

	ensure.Nil(t, os.Mkdir(filepath.Join(h.Env.Root, parsecli.CloudDir), 0755))
	ensure.Nil(t, os.Mkdir(filepath.Join(h.Env.Root, parsecli.HostingDir), 0755))
	ensure.Nil(t, os.Mkdir(filepath.Join(h.Env.Root, parsecli.ConfigDir), 0755))

	ensure.Nil(t,
		ioutil.WriteFile(
			filepath.Join(h.Env.Root, parsecli.HostingDir, "index.html"),
			[]byte(`<html>
<head> <title> Parse Project </title></head>
</html>`),
			0600),
	)

	ensure.Nil(t,
		ioutil.WriteFile(
			filepath.Join(h.Env.Root, parsecli.CloudDir, "main.js"),
			[]byte(`echo {"success": "ok"}`),
			0600),
	)
	ensure.Nil(t,
		ioutil.WriteFile(
			filepath.Join(h.Env.Root, parseIgnore),
			[]byte(`
*.swp
*~
`),
			0600),
	)

	ensure.Nil(t,
		ioutil.WriteFile(
			filepath.Join(h.Env.Root, parsecli.LegacyConfigFile),
			[]byte(`{
 "global": {"parseVersion": "1.2.9"}
}
`),
			0600,
		),
	)

	ignoredFiles := []string{
		filepath.Join(h.Env.Root, "public", ".ignore"),
		filepath.Join(h.Env.Root, "public", "#ignore"),
		filepath.Join(h.Env.Root, "cloud", "sample.txt"),
		filepath.Join(h.Env.Root, "cloud", "test~"),
	}

	for _, ignoredFile := range ignoredFiles {
		file, err := os.Create(ignoredFile)
		ensure.Nil(t, err)
		ensure.Nil(t, file.Close())
	}

	return h
}

func setupForDeploy(t testing.TB, info *deployInfo) *parsecli.Harness {
	h := createParseProject(t)

	ht := parsecli.TransportFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/1/deploy":
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(strings.NewReader(jsonStr(t, info))),
			}, nil
		case "/1/scripts", "/1/hosted_files":
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(strings.NewReader(`{"version":"f2"}`)),
			}, nil
		case "/1/jsVersions":
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(strings.NewReader(`{"js":["1.0","2.0"]}`)),
			}, nil
		default:
			return &http.Response{
				StatusCode: http.StatusExpectationFailed,
				Body:       ioutil.NopCloser(strings.NewReader(`{"error": "something is wrong"}`)),
			}, nil
		}
	})
	h.Env.ParseAPIClient = &parsecli.ParseAPIClient{APIClient: &parse.Client{Transport: ht}}
	return h
}

func TestGenericGetSourceFiles(t *testing.T) {
	t.Parallel()

	h := parsecli.NewHarness(t)
	h.MakeEmptyRoot()
	defer h.Stop()

	ensure.Nil(t, os.Mkdir(filepath.Join(h.Env.Root, "src"), 0755))
	for _, name := range []string{"index.html"} {
		file, err := os.Create(filepath.Join(h.Env.Root, "src", name))
		ensure.Nil(t, err)
		ensure.Nil(t, file.Close())
	}

	ensure.Nil(t, os.Symlink(filepath.Join(h.Env.Root, "src"), filepath.Join(h.Env.Root, "public")))

	var d deployCmd

	files, ignored, err := d.getSourceFiles(
		filepath.Join(h.Env.Root, "public"),
		map[string]struct{}{".html": {}},
		h.Env,
	)

	ensure.Nil(t, err)
	ensure.DeepEqual(t, files, []string{filepath.Join(h.Env.Root, "public", "index.html")})
	ensure.DeepEqual(t, len(ignored), 0)
}

func TestComputeChecksums(t *testing.T) {
	t.Parallel()
	h := createParseProject(t)
	defer h.Stop()

	files := []string{
		filepath.Join(h.Env.Root, "cloud", "main.js"),
		filepath.Join(h.Env.Root, "public", "index.html"),
	}

	var d deployCmd

	prefixLen := len(filepath.Join(h.Env.Root, "1")) - 1
	res, err := d.computeChecksums(files, func(name string) string {
		name = filepath.ToSlash(filepath.Clean(name))
		return name[prefixLen:]
	})
	ensure.Nil(t, err)
	ensure.DeepEqual(t, res, map[string]string{
		"cloud/main.js":     "4ece160cc8e5e828ee718e7367cf5d37",
		"public/index.html": "9e2354a0ebac5852bc674026137c8612"})
}

func TestUploadFileNoFile(t *testing.T) {
	t.Parallel()
	h := parsecli.NewHarness(t)
	defer h.Stop()

	var d deployCmd
	_, err := d.uploadFile("cloud/master.js", "", h.Env, nil)
	switch runtime.GOOS {
	case "windows":
		ensure.Err(t, err, regexp.MustCompile(`The system cannot find the path specified.`))
	default:
		ensure.Err(t, err, regexp.MustCompile(`no such file or directory`))
	}
}

func TestUploadFileHttpError(t *testing.T) {
	t.Parallel()
	h := createParseProject(t)
	defer h.Stop()

	var d deployCmd
	ht := parsecli.TransportFunc(func(r *http.Request) (*http.Response, error) {
		ensure.DeepEqual(t, r.URL.Path, "/1/uploads")
		return &http.Response{
			StatusCode: http.StatusExpectationFailed,
			Body:       ioutil.NopCloser(strings.NewReader(`{"error": "something is wrong"}`)),
		}, nil
	})
	h.Env.ParseAPIClient = &parsecli.ParseAPIClient{APIClient: &parse.Client{Transport: ht}}
	dirRoot := filepath.Join(h.Env.Root, "cloud")
	_, err := d.uploadFile(filepath.Join(dirRoot, "main.js"), "uploads",
		h.Env, func(name string) string { return "main.js" })
	ensure.Err(t, err, regexp.MustCompile("something is wrong"))
}

func TestUploadFileMalformed(t *testing.T) {
	t.Parallel()
	h := createParseProject(t)
	defer h.Stop()

	var d deployCmd
	ht := parsecli.TransportFunc(func(r *http.Request) (*http.Response, error) {
		ensure.DeepEqual(t, r.URL.Path, "/1/uploads")
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(strings.NewReader(`{"version": ""}`)),
		}, nil
	})

	h.Env.ParseAPIClient = &parsecli.ParseAPIClient{APIClient: &parse.Client{Transport: ht}}
	dirRoot := filepath.Join(h.Env.Root, "cloud")
	_, err := d.uploadFile(filepath.Join(dirRoot, "main.js"), "uploads", h.Env,
		func(name string) string { return "main.js" })
	ensure.Err(t, err, regexp.MustCompile(`Malformed response when trying to upload `))
}

func TestMakeNewRelease(t *testing.T) {
	t.Parallel()
	h := parsecli.NewHarness(t)
	defer h.Stop()

	var d deployCmd
	info := deployInfo{
		ReleaseName:  "v2",
		ParseVersion: "latest",
		Checksums: deployFileData{
			Cloud:  map[string]string{"main.js": "4ece160cc8e5e828ee718e7367cf5d37"},
			Public: map[string]string{"index.html": "9e2354a0ebac5852bc674026137c8612"},
		},
		Versions: deployFileData{
			Cloud:  map[string]string{"main.js": "f2"},
			Public: map[string]string{"index.html": "f2"},
		},
	}
	ht := parsecli.TransportFunc(func(r *http.Request) (*http.Response, error) {
		ensure.DeepEqual(t, r.URL.Path, "/1/deploy")
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(strings.NewReader(jsonStr(t, &info))),
		}, nil
	})
	h.Env.ParseAPIClient = &parsecli.ParseAPIClient{APIClient: &parse.Client{Transport: ht}}
	res, err := d.makeNewRelease(&deployInfo{}, h.Env)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, info, res)
}

func TestMakeNewReleaseError(t *testing.T) {
	t.Parallel()
	h := parsecli.NewHarness(t)
	defer h.Stop()

	var d deployCmd
	ht := parsecli.TransportFunc(func(r *http.Request) (*http.Response, error) {
		ensure.DeepEqual(t, r.URL.Path, "/1/deploy")
		return &http.Response{
			StatusCode: http.StatusExpectationFailed,
			Body:       ioutil.NopCloser(strings.NewReader(`{"error": "something is wrong"}`)),
		}, nil
	})
	h.Env.ParseAPIClient = &parsecli.ParseAPIClient{APIClient: &parse.Client{Transport: ht}}
	_, err := d.makeNewRelease(&deployInfo{}, h.Env)
	ensure.Err(t, err, regexp.MustCompile("something is wrong"))
}

func TestGetPrevDeplInfo(t *testing.T) {
	t.Parallel()
	h := parsecli.NewHarness(t)
	defer h.Stop()

	var d deployCmd
	info := &deployInfo{
		ReleaseName:  "v1",
		ParseVersion: "latest",
		Checksums: deployFileData{
			Cloud:  map[string]string{"main.js": "d41d8cd98f00b204e9800998ecf8427e"},
			Public: map[string]string{"index.html": "d41d8cd98f00b204e9800998ecf8427e"},
		},
		Versions: deployFileData{
			Cloud:  map[string]string{"main.js": "f1"},
			Public: map[string]string{"index.html": "f1"},
		},
	}
	ht := parsecli.TransportFunc(func(r *http.Request) (*http.Response, error) {
		ensure.DeepEqual(t, r.URL.Path, "/1/deploy")
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(strings.NewReader(jsonStr(t, info))),
		}, nil
	})
	h.Env.ParseAPIClient = &parsecli.ParseAPIClient{APIClient: &parse.Client{Transport: ht}}
	res, err := d.getPrevDeplInfo(h.Env)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, res, info)
}

func TestGetPrevDeplInfoLegacy(t *testing.T) {
	t.Parallel()
	h := parsecli.NewHarness(t)
	defer h.Stop()

	var d deployCmd

	info := &struct {
		ReleaseName  string            `json:"releaseName,omitempty"`
		Description  string            `json:"description,omitempty"`
		ParseVersion string            `json:"parseVersion,omitempty"`
		Checksums    map[string]string `json:"checksums,omitempty"`
		Versions     map[string]string `json:"userFiles,omitempty"`
	}{
		ReleaseName:  "v1",
		ParseVersion: "latest",
		Checksums:    map[string]string{"main.js": "d41d8cd98f00b204e9800998ecf8427e"},
		Versions:     map[string]string{"main.js": "f1"},
	}
	ht := parsecli.TransportFunc(func(r *http.Request) (*http.Response, error) {
		ensure.DeepEqual(t, r.URL.Path, "/1/deploy")
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(strings.NewReader(jsonStr(t, info))),
		}, nil
	})
	h.Env.ParseAPIClient = &parsecli.ParseAPIClient{APIClient: &parse.Client{Transport: ht}}
	res, err := d.getPrevDeplInfo(h.Env)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, res, &deployInfo{
		ReleaseName:  "v1",
		ParseVersion: "latest",
		Checksums: deployFileData{
			Cloud: map[string]string{"main.js": "d41d8cd98f00b204e9800998ecf8427e"},
		},
		Versions: deployFileData{
			Cloud: map[string]string{"main.js": "f1"},
		},
	})
}

func TestGetPrevDeplInfoError(t *testing.T) {
	t.Parallel()
	h := parsecli.NewHarness(t)
	defer h.Stop()

	var d deployCmd
	ht := parsecli.TransportFunc(func(r *http.Request) (*http.Response, error) {
		ensure.DeepEqual(t, r.URL.Path, "/1/deploy")
		return &http.Response{
			StatusCode: http.StatusExpectationFailed,
			Body:       ioutil.NopCloser(strings.NewReader(`{"error": "something is wrong"}`)),
		}, nil
	})
	h.Env.ParseAPIClient = &parsecli.ParseAPIClient{APIClient: &parse.Client{Transport: ht}}
	_, err := d.getPrevDeplInfo(h.Env)
	ensure.Err(t, err, regexp.MustCompile("something is wrong"))
}

func TestUploadSourceFilesChanged(t *testing.T) {
	t.Parallel()
	h := createParseProject(t)
	defer h.Stop()

	u := &uploader{
		DirName:  "cloud",
		Suffixes: map[string]struct{}{".js": {}},
		EndPoint: "uploads",
		Env:      h.Env,
		PrevChecksums: map[string]string{
			"main.js": "d41d8cd98f00b204e9800998ecf8427e",
		},
		PrevVersions: map[string]string{
			"main.js": "f1",
		},
	}

	ht := parsecli.TransportFunc(func(r *http.Request) (*http.Response, error) {
		ensure.DeepEqual(t, r.URL.Path, "/1/uploads")
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(strings.NewReader(`{"version": "f2"}`)),
		}, nil
	})

	h.Env.ParseAPIClient = &parsecli.ParseAPIClient{APIClient: &parse.Client{Transport: ht}}

	var d deployCmd
	checksums, versions, err := d.uploadSourceFiles(u)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, checksums, map[string]string{"main.js": "4ece160cc8e5e828ee718e7367cf5d37"})
	ensure.DeepEqual(t, versions, map[string]string{"main.js": "f2"})
}

func TestUploadSourceFilesUnChanged(t *testing.T) {
	t.Parallel()

	info := &deployInfo{
		ReleaseName:  "v1",
		ParseVersion: "latest",
		Checksums: deployFileData{
			Cloud: map[string]string{"main.js": "d41d8cd98f00b204e9800998ecf8427e"},
		},
		Versions: deployFileData{
			Cloud: map[string]string{"main.js": "f1"},
		},
	}

	h := setupForDeploy(t, info)
	defer h.Stop()

	u := &uploader{
		DirName:  "cloud",
		Suffixes: map[string]struct{}{".js": {}},
		EndPoint: "scripts",
		Env:      h.Env,
		PrevChecksums: map[string]string{
			"main.js": "4ece160cc8e5e828ee718e7367cf5d37",
		},
		PrevVersions: map[string]string{
			"main.js": "f1",
		},
	}

	var d deployCmd
	checksums, versions, err := d.uploadSourceFiles(u)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, checksums, map[string]string{"main.js": "4ece160cc8e5e828ee718e7367cf5d37"})
	ensure.DeepEqual(t, versions, map[string]string{"main.js": "f1"})

	d.Force = true // test force upload of unchanged files
	checksums, versions, err = d.uploadSourceFiles(u)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, checksums, map[string]string{"main.js": "4ece160cc8e5e828ee718e7367cf5d37"})
	ensure.DeepEqual(t, versions, map[string]string{"main.js": "f2"})
}

func TestDeployFilesChanged(t *testing.T) {
	t.Parallel()
	info := &deployInfo{
		ReleaseName:  "v1",
		ParseVersion: "latest",
		Checksums: deployFileData{
			Cloud:  map[string]string{"main.js": "d41d8cd98f00b204e9800998ecf8427e"},
			Public: map[string]string{"index.html": "d41d8cd98f00b204e9800998ecf8427e"},
		},
		Versions: deployFileData{
			Cloud:  map[string]string{"main.js": "f1"},
			Public: map[string]string{"index.html": "f1"},
		},
	}

	h := setupForDeploy(t, info)
	defer h.Stop()

	d := deployCmd{Verbose: true}
	res, err := d.deploy("old", nil, false, h.Env)
	ensure.Nil(t, err)

	expected := &deployInfo{
		ParseVersion: "latest",
		Checksums: deployFileData{
			Cloud:  map[string]string{"main.js": "4ece160cc8e5e828ee718e7367cf5d37"},
			Public: map[string]string{"index.html": "9e2354a0ebac5852bc674026137c8612"},
		},
		Versions: deployFileData{
			Cloud:  map[string]string{"main.js": "f2"},
			Public: map[string]string{"index.html": "f2"},
		},
	}
	ensure.DeepEqual(t, res, expected)
	ensure.DeepEqual(t,
		h.Out.String(),
		fmt.Sprintf(`Uploading source files
Uploading recent changes to scripts...
The following files will be uploaded:
%s
The following files will be ignored:
%s
Uploading recent changes to hosting...
The following files will be uploaded:
%s
The following files will be ignored:
%s
Finished uploading files
New release is named v1 (using Parse JavaScript SDK vlatest)
`,
			filepath.Join(h.Env.Root, parsecli.CloudDir, "main.js"),
			strings.Join([]string{
				filepath.Join(h.Env.Root, parsecli.CloudDir, "sample.txt"),
				filepath.Join(h.Env.Root, parsecli.CloudDir, "test~")},
				"\n"),
			filepath.Join(h.Env.Root, parsecli.HostingDir, "index.html"),
			strings.Join([]string{
				filepath.Join(h.Env.Root, parsecli.HostingDir, "#ignore"),
				filepath.Join(h.Env.Root, parsecli.HostingDir, ".ignore")},
				"\n"),
		),
	)
}

func TestDeployFilesUnChanged(t *testing.T) {
	t.Parallel()
	info := &deployInfo{
		ParseVersion: "latest",
		Checksums: deployFileData{
			Cloud:  map[string]string{"main.js": "4ece160cc8e5e828ee718e7367cf5d37"},
			Public: map[string]string{"index.html": "9e2354a0ebac5852bc674026137c8612"},
		},
		Versions: deployFileData{
			Cloud:  map[string]string{"main.js": "f2"},
			Public: map[string]string{"index.html": "f2"},
		},
	}

	h := setupForDeploy(t, info)
	defer h.Stop()

	d := deployCmd{Verbose: true}
	res, err := d.deploy("latest", nil, false, h.Env)
	ensure.Nil(t, err)

	expected := &deployInfo{
		ParseVersion: "latest",
		Checksums: deployFileData{
			Cloud:  map[string]string{"main.js": "4ece160cc8e5e828ee718e7367cf5d37"},
			Public: map[string]string{"index.html": "9e2354a0ebac5852bc674026137c8612"},
		},
		Versions: deployFileData{
			Cloud:  map[string]string{"main.js": "f2"},
			Public: map[string]string{"index.html": "f2"},
		},
	}
	ensure.DeepEqual(t, res, expected)
	ensure.DeepEqual(t, h.Out.String(), `Uploading source files
Finished uploading files
Not creating a release because no files have changed
`)
}

func TestDeployFilesNoVersion(t *testing.T) {
	t.Parallel()
	info := &deployInfo{
		ReleaseName:  "v1",
		ParseVersion: "latest",
		Checksums: deployFileData{
			Cloud:  map[string]string{"main.js": "d41d8cd98f00b204e9800998ecf8427e"},
			Public: map[string]string{"index.html": "d41d8cd98f00b204e9800998ecf8427e"},
		},
		Versions: deployFileData{
			Cloud:  map[string]string{"main.js": "f1"},
			Public: map[string]string{"index.html": "f1"},
		},
	}

	h := setupForDeploy(t, info)
	defer h.Stop()

	d := deployCmd{Verbose: true}
	res, err := d.deploy("", nil, false, h.Env)
	ensure.Nil(t, err)

	expected := &deployInfo{
		ParseVersion: "latest",
		Checksums: deployFileData{
			Cloud:  map[string]string{"main.js": "4ece160cc8e5e828ee718e7367cf5d37"},
			Public: map[string]string{"index.html": "9e2354a0ebac5852bc674026137c8612"},
		},
		Versions: deployFileData{
			Cloud:  map[string]string{"main.js": "f2"},
			Public: map[string]string{"index.html": "f2"},
		},
	}

	ensure.DeepEqual(t, res, expected)
	ensure.DeepEqual(t,
		h.Out.String(),
		fmt.Sprintf(`Uploading source files
Uploading recent changes to scripts...
The following files will be uploaded:
%s
The following files will be ignored:
%s
Uploading recent changes to hosting...
The following files will be uploaded:
%s
The following files will be ignored:
%s
Finished uploading files
New release is named v1 (using Parse JavaScript SDK vlatest)
`,
			filepath.Join(h.Env.Root, parsecli.CloudDir, "main.js"),
			strings.Join([]string{
				filepath.Join(h.Env.Root, parsecli.CloudDir, "sample.txt"),
				filepath.Join(h.Env.Root, parsecli.CloudDir, "test~")},
				"\n"),
			filepath.Join(h.Env.Root, parsecli.HostingDir, "index.html"),
			strings.Join([]string{
				filepath.Join(h.Env.Root, parsecli.HostingDir, "#ignore"),
				filepath.Join(h.Env.Root, parsecli.HostingDir, ".ignore")},
				"\n"),
		),
	)

	c, err := parsecli.ConfigFromDir(h.Env.Root)
	ensure.Nil(t, err)
	config, ok := (c).(*parsecli.ParseConfig)
	ensure.True(t, ok)
	ensure.DeepEqual(t, config.ProjectConfig.Parse.JSSDK, "2.0")
	ensure.True(t, strings.Contains(h.Err.String(), `JS SDK version not set, setting it to latest available JS SDK version
`),
	)
}

func TestDeployRetries(t *testing.T) {
	t.Parallel()

	h := parsecli.NewHarness(t)
	defer h.Stop()

	info := &struct {
		ReleaseName  string            `json:"releaseName,omitempty"`
		Description  string            `json:"description,omitempty"`
		ParseVersion string            `json:"parseVersion,omitempty"`
		Checksums    map[string]string `json:"checksums,omitempty"`
		Versions     map[string]string `json:"userFiles,omitempty"`
	}{
		ReleaseName:  "v1",
		ParseVersion: "latest",
		Checksums:    map[string]string{"main.js": "d41d8cd98f00b204e9800998ecf8427e"},
		Versions:     map[string]string{"main.js": "f1"},
	}
	ht := parsecli.TransportFunc(func(r *http.Request) (*http.Response, error) {
		ensure.DeepEqual(t, r.URL.Path, "/1/deploy")
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(strings.NewReader(jsonStr(t, info))),
		}, nil
	})
	h.Env.ParseAPIClient = &parsecli.ParseAPIClient{APIClient: &parse.Client{Transport: ht}}

	d := &deployCmd{Retries: 1}
	ctx := parsecli.Context{Config: defaultParseConfig}
	ctx.Config.GetProjectConfig().Parse.JSSDK = "latest"

	ensure.Err(t, d.run(h.Env, &ctx), regexp.MustCompile("no such file or directory"))
	ensure.DeepEqual(t, h.Err.String(), "")

	h.Err.Reset()
	d.Retries = 2
	ensure.Err(t, d.run(h.Env, &ctx), regexp.MustCompile("no such file or directory"))
	ensure.DeepEqual(
		t,
		h.Err.String(),
		"Deploy failed with error:\nlstat cloud: no such file or directory\nWill retry in 0 seconds.\n\n",
	)

	h.Err.Reset()
	d.Retries = 5
	ensure.Err(t, d.run(h.Env, &ctx), regexp.MustCompile("no such file or directory"))
	errStr := "Deploy failed with error:\nlstat cloud: no such file or directory\nWill retry in 0 seconds.\n\n"
	errStr += strings.Repeat("Sorry, deploy failed again with same error.\nWill retry in 0 seconds.\n\n", 3)
	ensure.DeepEqual(t, h.Err.String(), errStr)
}

func TestIgnoredFiles(t *testing.T) {
	t.Parallel()
	h := createParseProject(t)
	defer h.Stop()

	d := deployCmd{Verbose: true}
	_, ignored, err := d.getSourceFiles(filepath.Join(h.Env.Root, parsecli.CloudDir), map[string]struct{}{}, h.Env)
	ensure.Nil(t, err)
	ensure.DeepEqual(t,
		ignored,
		[]string{
			filepath.Join(h.Env.Root, parsecli.CloudDir, "test~")})
	_, ignored, err = d.getSourceFiles(filepath.Join(h.Env.Root, parsecli.HostingDir), map[string]struct{}{}, h.Env)
	ensure.DeepEqual(t,
		ignored,
		[]string{
			filepath.Join(h.Env.Root, parsecli.HostingDir, "#ignore"),
			filepath.Join(h.Env.Root, parsecli.HostingDir, ".ignore"),
		})
}

func TestIgnoredFilesUnderDotDir(t *testing.T) {
	t.Parallel()

	h := createParseProject(t)
	defer h.Stop()

	d := deployCmd{Verbose: true}

	_, ignored, err := d.getSourceFiles(filepath.Join(h.Env.Root, parsecli.CloudDir), map[string]struct{}{}, h.Env)
	ensure.Nil(t, err)
	ensure.DeepEqual(t,
		ignored,
		[]string{
			filepath.Join(h.Env.Root, parsecli.CloudDir, "test~")})
	_, ignored, err = d.getSourceFiles(filepath.Join(h.Env.Root, parsecli.HostingDir), map[string]struct{}{}, h.Env)
	ensure.DeepEqual(t,
		ignored,
		[]string{
			filepath.Join(h.Env.Root, parsecli.HostingDir, "#ignore"),
			filepath.Join(h.Env.Root, parsecli.HostingDir, ".ignore"),
		})
}
