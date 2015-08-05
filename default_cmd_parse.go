package main

import (
	"fmt"

	"github.com/facebookgo/stackerr"
)

func (d *defaultCmd) setParseDefault(e *env, newDefault, defaultApp string, parseConfig *parseConfig) error {
	apps := parseConfig.Applications
	if _, ok := apps[newDefault]; !ok {
		parseConfig.pprintApps(e)
		return stackerr.Newf("Invalid application name \"%s\". Please select from the valid applications printed above.", newDefault)
	}

	if defaultApp == "" {
		apps[defaultKey] = &parseAppConfig{Link: newDefault}
	} else {
		apps[defaultKey].Link = newDefault
	}
	if err := storeParseConfig(e, parseConfig); err != nil {
		return err
	}
	fmt.Fprintf(e.Out, "Default app set to %s.\n", newDefault)
	return nil
}
