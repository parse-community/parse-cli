package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/facebookgo/stackerr"
	"github.com/spf13/cobra"
)

type migrateCmd struct {
	retainMaster bool
}

func (m *migrateCmd) upgradeLegacy(e *env, c config) (*parseConfig, error) {
	p := c.getProjectConfig()
	if p.Type != legacyParseFormat {
		return nil, stackerr.New("Already using the preferred config format.")
	}
	p.Type = parseFormat
	config, ok := (c).(*parseConfig)
	if !ok {
		return nil, stackerr.Newf("Unexpected config format: %d", p.Type)
	}
	if !m.retainMaster {
		for _, app := range config.Applications {
			app.MasterKey = ""
		}
	}
	return config, nil
}

func (m *migrateCmd) run(e *env) error {
	c, err := configFromDir(e.Root)
	if err != nil {
		return err
	}

	c, err = m.upgradeLegacy(e, c)
	if err != nil {
		return err
	}
	localErr := storeConfig(e, c)
	projectErr := storeProjectConfig(e, c)
	if localErr == nil && projectErr == nil {
		legacy := filepath.Join(e.Root, legacyConfigFile)
		err := os.Remove(legacy)
		if err != nil {
			fmt.Fprintf(e.Err, "Could not delete: %q. Please remove this file manually.\n", legacy)
		}
	} else {
		local := filepath.Join(e.Root, parseLocal)
		err := os.Remove(local)
		if err != nil {
			fmt.Fprintf(e.Err, "Failed to clean up: %q. Please remove this file manually.\n", local)
		}
		project := filepath.Join(e.Root, parseProject)
		err = os.Remove(project)
		if err != nil {
			fmt.Fprintf(e.Err, "Failed to clean up: %q. Please remove this file manually.\n", project)
		}
	}
	return nil
}

func newMigrateCmd(e *env) *cobra.Command {
	var m migrateCmd
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate project config format to preferred format",
		Long: `Use this on projects with legacy config format to migrate
to the preferred format.
`,
		Run: runNoArgs(e, m.run),
	}
	cmd.Flags().BoolVarP(&m.retainMaster, "retain", "r", m.retainMaster,
		"Retain any master keys present in config during migration")
	return cmd
}
