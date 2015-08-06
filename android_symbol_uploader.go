package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/facebookgo/stackerr"
)

var androidCodeversion = regexp.MustCompile(`versionCode='(\d+)'`)

type androidSymbolUploader struct {
	Path     string
	Apk      string
	Manifest string
	AAPT     string
}

func findAAPT(root string) []string {
	var aapts []string
	// we ignore the error returned by filepath.Walk because we are only making
	// a best eeffort search for aapt, if there is an error for any reason,
	// we just ignore it & forgo the search
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info != nil && filepath.Base(path) == "aapt" {
			aapts = append(aapts, path)
		}
		return nil
	})
	return aapts
}

func (a *androidSymbolUploader) validate() error {
	if (a.Apk == "") == (a.Manifest == "") {
		return stackerr.New("You need to supply one of either the apk or manifest file.")
	}
	return nil
}

func (a *androidSymbolUploader) acceptsPath() bool {
	return filepath.Base(a.Path) == "mapping.txt"
}

func (a *androidSymbolUploader) getAAPT(androidHome string) (string, error) {
	options := []string{a.AAPT}
	// either in $ANDROID_HOME/platform-tools/aapt
	options = append(options, filepath.Join(androidHome, "platform_tools", "aapt"))

	// or $ANDROID_HOME/build-tools/<VERSION>/aapt
	buildTools := filepath.Join(androidHome, "build_tools")
	options = append(options, findAAPT(buildTools)...)

	for _, o := range options {
		if out, err := exec.Command(o).Output(); err == nil {
			if bytes.HasPrefix(out, []byte("Android Asset Packaging Tool")) {
				return o, nil
			}
		}
	}
	return "", stackerr.New("Cannot find aapt, you might need to set ANDROID_HOME.")
}

func (a *androidSymbolUploader) getBuildVersionFromManifest() (int, error) {
	if a.Manifest == "" {
		return 0, stackerr.New("Please provide manifest file path.")
	}
	var version struct {
		XMLName     xml.Name `xml:"manifest"`
		VersionCode int      `xml:"http://schemas.android.com/apk/res/android versionCode,attr"`
	}
	manifest, err := os.Open(a.Manifest)
	if err != nil {
		return 0, stackerr.Wrap(err)
	}
	defer manifest.Close()
	if err := xml.NewDecoder(manifest).Decode(&version); err != nil {
		return 0, stackerr.Wrap(err)
	}
	if version.VersionCode == 0 {
		return 0, stackerr.New("Manifest does not contain build version.")
	}
	return version.VersionCode, nil
}

func (a *androidSymbolUploader) getBuildVersionFromAPK() (int, error) {
	if a.Apk == "" {
		return 0, stackerr.New("Please provide apk file path.")
	}
	if a.AAPT == "" {
		androidHome := os.Getenv("ANDROID_HOME")
		if androidHome == "" {
			androidHome = os.Getenv("ANDROID_SDK")
		}
		if androidHome == "" {
			return 0, stackerr.New("Cannot find aapt, you might need to set ANDROID_HOME.")
		}

		aapt, err := a.getAAPT(androidHome)
		if err != nil {
			return 0, err
		}
		a.AAPT = aapt
	}

	bytes, err := exec.Command(a.AAPT, "dump", "badging", a.Apk).Output()
	if err != nil {
		return 0, stackerr.Wrap(err)
	}
	return parseVersionFromBytes(bytes)
}

func parseVersionFromBytes(bytes []byte) (int, error) {
	v := androidCodeversion.FindAllSubmatch(bytes, 1)
	if len(v) == 0 {
		return 0, stackerr.New("Cannot determine build version.")
	}
	version, err := strconv.Atoi(string(v[0][1]))
	if err != nil {
		return 0, stackerr.Wrap(err)
	}
	return int(version), nil
}

func (a *androidSymbolUploader) getBuildVersion(e *env) (int, error) {
	var f = func(versionCode int, err error) (int, error) {
		if err != nil {
			return 0, err
		}
		if versionCode == 1 {
			fmt.Fprintln(e.Out, "Warning: build number is '1'.")
		}
		return versionCode, nil
	}

	var (
		versionCode int
		err         error
	)
	if a.Manifest != "" {
		versionCode, err = a.getBuildVersionFromManifest()
	} else {
		versionCode, err = a.getBuildVersionFromAPK()
	}

	return f(versionCode, err)
}

func (a *androidSymbolUploader) uploadSymbols(e *env) error {
	appBuildVersion, err := a.getBuildVersion(e)
	if err != nil {
		return err
	}
	commonHeaders := map[string]string{
		"X-Parse-App-Build-Version": fmt.Sprintf("%d", appBuildVersion),
	}
	return uploadSymbolFiles([]string{a.Path}, commonHeaders, false, e)
}
