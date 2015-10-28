package herokucmd

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/ParsePlatform/parse-cli/parsecli"
	"github.com/facebookgo/stackerr"
	"github.com/skratchdot/open-golang/open"
)

const linkHerokuURL = "https://parse.com/account/edit"

func selectHerokuApp(apps nameIDs, e *parsecli.Env) (*nameID, error) {
	fmt.Fprintf(e.Out, "Please select from the following Heroku apps: (Enter a number between 1 and %d)\n", len(apps))
	for i, app := range apps {
		w := new(tabwriter.Writer)
		w.Init(e.Out, 0, 8, 0, '\t', 0)
		fmt.Fprintf(w, "%d: %s\t\t(%s)\n", i+1, app.name, app.id)
		if err := w.Flush(); err != nil {
			return nil, stackerr.Wrap(err)
		}
	}
	fmt.Fprintf(e.Out, "Selection: ")
	var selection string
	fmt.Fscanf(e.In, "%s\n", &selection)

	n, err := strconv.Atoi(selection)
	if err != nil {
		return nil, err
	}

	lapps := len(apps)
	if n <= 0 || n > lapps {
		return nil, stackerr.Newf("Invalid selection: can only be in range 1..%d", lapps)
	}
	return &apps[n-1], nil
}

func getRandomAppName(app *parsecli.App) string {
	rand.Seed(time.Now().Unix())

	namePrefix := strings.Join(strings.Fields(strings.ToLower(app.Name)), "")
	if namePrefix == "" {
		namePrefix = "parsoku"
	}

	construct := fmt.Sprintf(
		"%s-%s-%d",
		namePrefix,
		strings.ToLower(app.ApplicationID[:6]),
		rand.Intn(10000),
	)
	l := len(construct)
	if l > 25 {
		l = 25
	}
	if construct[l-1] == '-' { // ensure no dash at the end
		l -= -1
	}
	return construct[:l]
}

func GetLinkedHerokuAppConfig(app *parsecli.App, e *parsecli.Env) (bool, *parsecli.HerokuAppConfig, error) {
	h := &herokuLink{parseAppID: app.ApplicationID}
	apps, err := h.getAppLinks(e)
	if err != nil {
		return false, nil, err
	}
	if len(apps) == 0 {
		randomName := getRandomAppName(app)
		appName := randomName
		for i := 0; i < 3; i++ {
			if appName == randomName {
				fmt.Fprintf(e.Out,
					`Let's create a new Heroku app in which server code will be run.
The Heroku app will be named: %q
Note that this can be changed later using Heroku API or Dashboard.

`,
					randomName,
				)
			} else {
				appName = h.getHerokuAppName(e)
			}

			id, err := h.createNewLink(e, appName)
			if err == nil {
				return true, &parsecli.HerokuAppConfig{
					ParseAppID:  app.ApplicationID,
					HerokuAppID: id,
				}, nil
			}
			if i == 2 {
				return false, nil, err
			}

			switch {
			case stackerr.HasUnderlying(err, stackerr.MatcherFunc(herokuAppNameTaken)):
				fmt.Fprintf(e.Err, "App name %s already taken.\nPlease try again...\n\n", appName)
				appName = ""

			case stackerr.HasUnderlying(err, stackerr.MatcherFunc(herokuAccountNotLinked)):
				fmt.Fprintf(e.Err, `Looks like you have not yet linked your Heroku Account to your Parse account.
Trying to open %q in the browser.
Please click "Link Heroku" button at the bottom.
`,
					linkHerokuURL,
				)
				appName = randomName
				err := open.Run(linkHerokuURL)
				if err != nil {
					fmt.Fprintf(e.Err,
						`Sorry, we couldn’t open the browser for you. Go to
	%q
and click the "Link Heroku" button to link your Heroku account to Parse.
`,
						linkHerokuURL,
					)
				}

				fmt.Fprintf(e.Out, "Press ENTER when you are done linking your Heroku account to Parse: ")
				var discard string
				fmt.Fscanf(e.In, "%s\n", &discard)

			default:
				return false, nil, err
			}
		}
	}

	if len(apps) == 1 {
		return false, &parsecli.HerokuAppConfig{
			ParseAppID:  app.ApplicationID,
			HerokuAppID: apps[0].id,
		}, nil
	}

	// NOTE: this part of code will not be used for now
	for r := 0; r < 3; r++ {
		selected, err := selectHerokuApp(apps, e)
		if err != nil {
			fmt.Fprintf(e.Err, "error: %s.\nPlease try again...\n", parsecli.ErrorString(e, err))
			continue
		}
		if selected.id != "" {
			return false, &parsecli.HerokuAppConfig{
				ParseAppID:  app.ApplicationID,
				HerokuAppID: selected.id,
			}, nil
		}
		id, err := h.createNewLink(e, selected.name)
		if err != nil {
			return false, nil, err
		}
		return false, &parsecli.HerokuAppConfig{
			ParseAppID:  app.ApplicationID,
			HerokuAppID: id,
		}, nil
	}
	return false, nil, stackerr.New("failed to selected an heroku app in 3 attempts")
}

func AddSelectedHerokuApp(
	appName string,
	appConfig *parsecli.HerokuAppConfig,
	args []string,
	makeDefault, verbose bool,
	e *parsecli.Env,
) error {
	config, err := parsecli.ConfigFromDir(e.Root)
	if err != nil {
		return err
	}
	herokuConfig, ok := config.(*parsecli.HerokuConfig)
	if !ok {
		return stackerr.New("Invalid Heroku config.")
	}

	// add app to config
	if _, ok := herokuConfig.Applications[appName]; ok {
		return stackerr.Newf("App %s has already been added", appName)
	}

	herokuConfig.Applications[appName] = appConfig

	if len(args) > 0 && args[0] != "" {
		alias := args[0]
		aliasConfig, ok := herokuConfig.Applications[alias]
		if !ok {
			herokuConfig.Applications[alias] = &parsecli.HerokuAppConfig{Link: appName}
		}
		if ok && aliasConfig.GetLink() != "" {
			fmt.Fprintf(e.Out, "Overwriting alias: %q to point to %q\n", alias, appName)
			herokuConfig.Applications[alias] = &parsecli.HerokuAppConfig{Link: appName}
		}
	}

	if makeDefault {
		if _, ok := herokuConfig.Applications[parsecli.DefaultKey]; ok {
			return stackerr.New(`Default key already set. To override default, use command "parse default"`)
		}
		herokuConfig.Applications[parsecli.DefaultKey] = &parsecli.HerokuAppConfig{Link: appName}
	}

	if err := parsecli.StoreConfig(e, herokuConfig); err != nil {
		return err
	}
	if verbose {
		fmt.Fprintf(e.Out, "Written config for %q\n", appName)
		if makeDefault {
			fmt.Fprintf(e.Out, "Set %q as default\n", appName)
		}
	}

	return nil
}
