package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/facebookgo/stackerr"
	"github.com/spf13/cobra"
)

const (
	expressEjs  = "express-ejs"
	expressJade = "express-jade"
	comma       = ", "
)

var validTypes = map[string]bool{
	expressEjs:  true,
	expressJade: true,
}

type generateCmd struct {
	generateType string
}

func (g *generateCmd) validateArgs() error {
	if _, ok := validTypes[g.generateType]; !ok {
		var buf bytes.Buffer
		for key := range validTypes {
			buf.WriteString(key)
			buf.WriteString(comma)
		}
		return stackerr.Newf("type can only be one of {%s}", buf.String())
	}
	return nil
}

func (g *generateCmd) run(e *env) error {
	if err := g.validateArgs(); err != nil {
		return err
	}

	destDir := filepath.Join(e.Root, cloudDir)
	existingFiles := false

	appJs := filepath.Join(destDir, "app.js")
	if _, err := os.Stat(appJs); err == nil {
		fmt.Fprintf(e.Err, "%s already exists.\n", appJs)
		existingFiles = true
	} else {
		if !os.IsNotExist(err) {
			return stackerr.Wrap(err)
		}
	}

	viewsDir := filepath.Join(destDir, "views")
	if _, err := os.Stat(viewsDir); err != nil {
		if !os.IsNotExist(err) {
			return stackerr.Wrap(err)
		}
		fmt.Fprintf(e.Out, "Creating directory %s.\n", viewsDir)
		if err := os.MkdirAll(viewsDir, 0755); err != nil {
			return stackerr.Wrap(err)
		}
	}

	var helloHTMLTemplate string
	switch g.generateType {
	case expressJade:
		helloHTMLTemplate = filepath.Join(viewsDir, "hello.jade")
	case expressEjs:
		helloHTMLTemplate = filepath.Join(viewsDir, "hello.ejs")
	}

	if _, err := os.Stat(helloHTMLTemplate); err == nil {
		fmt.Fprintf(e.Err, "%s already exists.\n", helloHTMLTemplate)
		existingFiles = true
	} else {
		if !os.IsNotExist(err) {
			return stackerr.Wrap(err)
		}
	}

	if existingFiles {
		return stackerr.New("Please remove the above existing files and try again.")
	}

	fmt.Fprintf(e.Out, "Writing out sample file %s\n", appJs)
	file, err := os.Create(appJs)
	if err != nil && !os.IsExist(err) {
		return stackerr.Wrap(err)
	}

	defer file.Close()

	switch g.generateType {
	case expressJade:
		if _, err := file.WriteString(strings.Replace(sampleAppJS, "ejs", "jade", -1)); err != nil {
			return stackerr.Wrap(err)
		}
	case expressEjs:
		if _, err := file.WriteString(sampleAppJS); err != nil {
			return stackerr.Wrap(err)
		}
	}
	if err := file.Close(); err != nil {
		return stackerr.Wrap(err)
	}

	fmt.Fprintf(e.Out, "Writing out sample file %s\n", helloHTMLTemplate)
	file, err = os.Create(helloHTMLTemplate)
	if err != nil && !os.IsExist(err) {
		return stackerr.Wrap(err)
	}

	defer file.Close()

	switch g.generateType {
	case expressJade:
		if _, err := file.WriteString(helloJade); err != nil {
			return stackerr.Wrap(err)
		}
	case expressEjs:
		if _, err := file.WriteString(helloEJS); err != nil {
			return stackerr.Wrap(err)
		}
	}
	if err := file.Close(); err != nil {
		return stackerr.Wrap(err)
	}

	fmt.Fprintf(e.Out, "\nAlmost done! Please add this line to the top of your main.js:\n\n")
	fmt.Fprintf(e.Out, "\trequire('cloud/app.js');\n\n")
	return nil
}

func newGenerateCmd(e *env) *cobra.Command {
	c := generateCmd{}
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generates a sample express app in the current project directory",
		Long:  "Generates a sample express app in the current project directory.",
		Run:   runNoArgs(e, c.run),
	}
	cmd.Flags().StringVarP(&c.generateType, "type", "t", expressEjs, "Type of templates to use for generation.")
	return cmd
}
