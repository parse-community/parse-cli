package parsecmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ParsePlatform/parse-cli/parsecli"
	"github.com/facebookgo/stackerr"
	"github.com/spf13/cobra"
)

type migrateCmd struct {
	retainMaster bool
}

func (m *migrateCmd) upgradeLegacy(e *parsecli.Env, c parsecli.Config) (*parsecli.ParseConfig, error) {
	p := c.GetProjectConfig()
	if p.Type != parsecli.LegacyParseFormat {
		return nil, stackerr.New("Already using the preferred config format.")
	}
	p.Type = parsecli.ParseFormat
	config, ok := (c).(*parsecli.ParseConfig)
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

func (m *migrateCmd) run(e *parsecli.Env) error {
	c, err := parsecli.ConfigFromDir(e.Root)
	if err != nil {
		return err
	}

	c, err = m.upgradeLegacy(e, c)
	if err != nil {
		return err
	}
	localErr := parsecli.StoreConfig(e, c)
	projectErr := parsecli.StoreProjectConfig(e, c)
	if localErr == nil && projectErr == nil {
		legacy := filepath.Join(e.Root, parsecli.LegacyConfigFile)
		err := os.Remove(legacy)
		if err != nil {
			fmt.Fprintf(e.Err, "Could not delete: %q. Please remove this file manually.\n", legacy)
		}
	} else {
		local := filepath.Join(e.Root, parsecli.ParseLocal)
		err := os.Remove(local)
		if err != nil {
			fmt.Fprintf(e.Err, "Failed to clean up: %q. Please remove this file manually.\n", local)
		}
		project := filepath.Join(e.Root, parsecli.ParseProject)
		err = os.Remove(project)
		if err != nil {
			fmt.Fprintf(e.Err, "Failed to clean up: %q. Please remove this file manually.\n", project)
		}
	}
	fmt.Fprintln(e.Out, "Successfully migrated to the preferred config format.")
	return nil
}

func NewMigrateCmd(e *parsecli.Env) *cobra.Command {
	var m migrateCmd
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate project config format to preferred format",
		Long: `Use this on projects with legacy config format to migrate
to the preferred format.
`,
		Run: parsecli.RunNoArgs(e, m.run),
	}
	cmd.Flags().BoolVarP(&m.retainMaster, "retain", "r", m.retainMaster,
		"Retain any master keys present in config during migration")
	return cmd
}
