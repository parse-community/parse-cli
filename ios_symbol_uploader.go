package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/DHowett/go-plist"
	"github.com/facebookgo/stackerr"
	"github.com/inconshreveable/go-update"
	"github.com/mitchellh/go-homedir"
)

const (
	osxSymbolConverterDownloadURL = "https://www.parse.com/downloads/cloud_code/parse_symbol_converter"
	osxSymbolConverterVersion     = "1.0.2"
)

type iosSymbolUploader struct {
	Path        string
	SkipOsCheck bool
}

func (i *iosSymbolUploader) validate() error {
	if i.SkipOsCheck {
		return nil
	}
	if runtime.GOOS != "darwin" {
		return stackerr.New("Upload of iOS symbol files is only available on OS X.")
	}
	return nil
}

func getDwarfPath(path string) (string, error) {
	// E.g. Yarr.xcarchive/dSYMs/yarr.dSYM/Contents/Resources/DWARF/Yarr

	// Find first dSYM bundle if it is an `xcarchive` bundle
	if filepath.Ext(path) == ".xcarchive" {
		path = filepath.Join(path, "dSYMs")
		filenames, err := readDirNames(path)
		if err != nil {
			return "", err
		}
		for _, filename := range filenames {
			if filepath.Ext(filename) == ".dSYM" {
				path = filename
				break
			}
		}
	}

	// Find first DWARF file inside if it is a `dSYM` bundle
	if filepath.Ext(path) == ".dSYM" {
		dwarfFilesDir := filepath.Join(path, "Contents", "Resources", "DWARF")
		filenames, err := readDirNames(dwarfFilesDir)
		if err != nil {
			return "", stackerr.Wrap(err)
		}
		if len(filenames) > 0 {
			return filenames[0], nil
		}
	}

	// If no extension - just return the file, as it probably is DWARF
	if filepath.Ext(path) == "" {
		return path, nil
	}

	return "", nil
}

func (i *iosSymbolUploader) acceptsPath() bool {
	dwarfPath, err := getDwarfPath(i.Path)
	if err != nil {
		return false
	}
	return dwarfPath != ""
}

func (i *iosSymbolUploader) getBuildVersionFromXcarchive() (string, error) {
	if filepath.Ext(i.Path) != ".xcarchive" {
		return "1", nil
	}

	plistPath := filepath.Join(i.Path, "Info.plist")
	plistFile, err := os.Open(plistPath)
	if err != nil {
		return "", stackerr.Wrap(err)
	}
	type version struct {
		CFBundleVersion string `plist:"CFBundleVersion"`
	}
	var properties struct {
		ApplicationProperties version `plist:"ApplicationProperties"`
	}
	if err := plist.NewDecoder(plistFile).Decode(&properties); err != nil {
		return "", stackerr.Wrap(err)
	}
	return properties.ApplicationProperties.CFBundleVersion, nil
}

func (i *iosSymbolUploader) uploadSymbols(e *env) error {
	appBuildVersion, err := i.getBuildVersionFromXcarchive()
	if err != nil {
		return err
	}
	dwarfFilePath, err := getDwarfPath(i.Path)
	if err != nil {
		return err
	}
	dwarfChecksum, err := base64MD5OfFile(dwarfFilePath)
	if err != nil {
		return err
	}

	symbolFiles, err := i.convertSymbols(e)
	if err != nil {
		return err
	}

	return uploadSymbolFiles(symbolFiles, map[string]string{
		"X-Parse-App-Build-Version": appBuildVersion,
		"X-Parse-Apple-DWARF-MD5":   dwarfChecksum,
	}, true, e)
}

func (i *iosSymbolUploader) prepareSymbolsFolder(folderPath string, e *env) error {
	if err := os.RemoveAll(folderPath); err != nil {
		return stackerr.Wrap(err)
	}
	if err := os.MkdirAll(folderPath, 0755); err != nil {
		return stackerr.Wrap(err)
	}
	return nil
}

type updateTool func(string, string) error

func (i *iosSymbolUploader) symbolConversionTool(baseDir string,
	updater updateTool,
	e *env) (string, error) {

	converter := os.Getenv("PARSE_SYMBOL_CONVERTER")
	if converter != "" {
		return converter, nil
	}

	toUpdate := true

	toolFolder := filepath.Join(baseDir, ".parse")
	toolPath := filepath.Join(toolFolder, "parse_symbol_converter")

	if err := os.MkdirAll(toolFolder, 0755); err != nil {
		return "", stackerr.Wrap(err)
	}

	_, err := os.Lstat(toolPath)
	if err == nil {
		version, err := exec.Command(toolPath, "--version").Output()
		if err == nil && string(bytes.TrimSpace(version)) == osxSymbolConverterVersion {
			toUpdate = false
		}
	}
	if os.IsNotExist(err) {
		file, err := os.Create(toolPath)
		if err != nil {
			return "", stackerr.Wrap(err)
		}
		if err := file.Close(); err != nil {
			return "", stackerr.Wrap(err)
		}
	}

	if updater == nil {
		if toUpdate {
			fmt.Fprintln(e.Out, "Fetching required resources...")
			err, _ = update.New().Target(toolPath).FromUrl(osxSymbolConverterDownloadURL)
		}
	} else {
		err = updater(toolPath, osxSymbolConverterDownloadURL)
	}
	if err != nil {
		return "", stackerr.Wrap(err)
	}
	if err := os.Chmod(toolPath, 0755); err != nil {
		return "", stackerr.Wrap(err)
	}
	if toUpdate {
		fmt.Fprintln(e.Out, "Additional resources installed.")
	}
	return toolPath, nil
}

func (i *iosSymbolUploader) convertSymbols(e *env) ([]string, error) {
	homedir, err := homedir.Dir()
	if err != nil {
		return nil, stackerr.Wrap(err)
	}

	folderPath := filepath.Join(homedir, ".parse", "CrashReportingSymbols")
	if err := i.prepareSymbolsFolder(folderPath, e); err != nil {
		return nil, stackerr.Wrap(err)
	}

	conversionTool, err := i.symbolConversionTool(homedir, nil, e)
	if err != nil {
		return nil, stackerr.Wrap(err)
	}

	cmd := exec.Command(conversionTool, i.Path, folderPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, stackerr.Newf("Symbol conversion failed with:\n%s", string(out))
	}

	filenames, err := readDirNames(folderPath)
	if err != nil {
		return nil, stackerr.Wrap(err)
	}
	return filenames, nil
}
