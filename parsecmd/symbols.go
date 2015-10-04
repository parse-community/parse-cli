package parsecmd

import (
	"fmt"

	"github.com/ParsePlatform/parse-cli/parsecli"
	"github.com/facebookgo/stackerr"
	"github.com/spf13/cobra"
)

type symbolsCmd struct {
	path        string
	apk         string
	manifest    string
	aapt        string
	skipOsCheck bool
}

func (s *symbolsCmd) run(e *parsecli.Env, c *parsecli.Context) error {
	android := &androidSymbolUploader{
		Path:     s.path,
		Apk:      s.apk,
		Manifest: s.manifest,
		AAPT:     s.aapt}
	ios := &iosSymbolUploader{
		Path:        s.path,
		SkipOsCheck: s.skipOsCheck}
	switch {
	case android.acceptsPath():
		if err := android.validate(); err != nil {
			return err
		}
		fmt.Fprintln(e.Out, "Uploading Android symbol files...")
		return android.uploadSymbols(e)
	case ios.acceptsPath():
		if err := ios.validate(); err != nil {
			return err
		}
		fmt.Fprintln(e.Out, "Uploading iOS symbol files...")
		return ios.uploadSymbols(e)
	default:
		if s.path == "" {
			return stackerr.New("Please specify path to symbol files")
		}
		return stackerr.Newf("Do not understand symbol files at : %s", s.path)
	}
}

func NewSymbolsCmd(e *parsecli.Env) *cobra.Command {
	var s symbolsCmd
	cmd := &cobra.Command{
		Use:   "symbols [app]",
		Short: "Uploads symbol files",
		Long: `Uploads the symbol files for the application to symbolicate crash reports with.
Path specifies the path to xcarchive/dSYM/DWARF for iOS or mapping.txt for Android.`,
		Run: parsecli.RunWithClient(e, s.run),
	}
	cmd.Flags().StringVarP(&s.path, "path", "p", s.path,
		"Path to symbols files")
	cmd.Flags().StringVarP(&s.apk, "apk", "a", s.apk,
		"Path to apk file")
	cmd.Flags().StringVarP(&s.manifest, "manifest", "m", s.manifest,
		"Path to AndroidManifest.xml file")
	cmd.Flags().StringVarP(&s.aapt, "aapt", "t", s.aapt,
		"Path to aapt for android")
	return cmd
}
