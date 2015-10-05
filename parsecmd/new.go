package parsecmd

import (
	"fmt"

	"github.com/ParsePlatform/parse-cli/parsecli"
	"github.com/facebookgo/parse"
)

func CloneSampleCloudCode(
	e *parsecli.Env,
	isNew, configOnly bool,
	appConfig parsecli.AppConfig) (bool, error) {
	dumpTemplate := false
	if !isNew && !configOnly {
		// if parse app was already created try to fetch Cloud Code and populate dir
		masterKey, err := appConfig.GetMasterKey(e)
		if err != nil {
			return false, err
		}
		e.ParseAPIClient = e.ParseAPIClient.WithCredentials(
			parse.MasterKey{
				ApplicationID: appConfig.GetApplicationID(),
				MasterKey:     masterKey,
			},
		)

		d := &downloadCmd{destination: e.Root}
		err = d.run(e, nil)
		if err != nil {
			if err == errNoFiles {
				dumpTemplate = true
			} else {
				fmt.Fprintln(
					e.Out,
					`
NOTE: If you like to fetch the latest deployed Cloud Code from Parse, 
you can use the "parse download" command after finishing the set up.
This will download Cloud Code to a temporary location.
`,
				)
			}
		}
	}
	dumpTemplate = (isNew || dumpTemplate) && !configOnly
	return dumpTemplate, parsecli.CloneSampleCloudCode(e, dumpTemplate)
}
