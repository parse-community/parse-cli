package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/facebookgo/stackerr"
)

func (n *newCmd) cloneSampleCloudCode(e *env, dumpTemplate bool) error {
	err := os.MkdirAll(e.Root, 0755)
	if err != nil {
		return stackerr.Wrap(err)
	}

	err = n.createConfigWithContent(
		filepath.Join(e.Root, parseProject),
		fmt.Sprintf(
			`{
  "project_type" : %d,
  "parse": {"jssdk":""}
}`,
			parseFormat,
		),
	)
	if err != nil {
		return err
	}
	err = n.createConfigWithContent(
		filepath.Join(e.Root, parseLocal),
		"{}",
	)
	if err != nil {
		return err
	}

	// no need to set up the template code
	if !dumpTemplate {
		return nil
	}

	for _, info := range newProjectFiles {
		sampleDir := filepath.Join(e.Root, info.dirname)
		if _, err := os.Stat(sampleDir); err != nil {
			if !os.IsNotExist(err) {
				return stackerr.Wrap(err)
			}
			if err := os.Mkdir(sampleDir, 0755); err != nil {
				return stackerr.Wrap(err)
			}
		}

		sampleFile := filepath.Join(sampleDir, info.filename)
		if _, err := os.Stat(sampleFile); err != nil {
			if os.IsNotExist(err) {
				file, err := os.Create(sampleFile)
				if err != nil && !os.IsExist(err) {
					return stackerr.Wrap(err)
				}

				defer file.Close()

				if _, err := file.WriteString(info.content); err != nil {
					return stackerr.Wrap(err)
				}
				if err := file.Close(); err != nil {
					return stackerr.Wrap(err)
				}
			}
		}
	}
	return nil
}
