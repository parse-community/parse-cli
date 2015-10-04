package herokucmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"sort"
	"sync"

	"github.com/ParsePlatform/parse-cli/parsecli"
	"github.com/facebookgo/errgroup"
	"github.com/facebookgo/jsonpipe"
	"github.com/facebookgo/stackerr"
)

const maxRequests = 5

type herokuLink struct {
	parseAppID string
}

var (
	herokuNameTakenRegex        = regexp.MustCompile("Name is already taken")
	herokuAccountNotLinkedRegex = regexp.MustCompile("Parse account is not linked to Heroku")
)

func herokuAppNameTaken(err error) bool {
	return herokuNameTakenRegex.MatchString(err.Error())
}

func herokuAccountNotLinked(err error) bool {
	return herokuAccountNotLinkedRegex.MatchString(err.Error())
}

func (h *herokuLink) getHerokuAppName(e *parsecli.Env) string {
	fmt.Fprint(e.Out,
		`Let's create a new Heroku app in which server code will be run.
You can either provide a name for the Heroku app or
just hit ENTER to use a randomly generated name.
Note that this can be changed later using Heroku API or Dashboard.

Heroku App Name: `)
	var name string
	fmt.Fscanf(e.In, "%s\n", &name)
	return name
}

func (h *herokuLink) createNewLink(e *parsecli.Env, herokuAppName string) (string, error) {
	var l parsecli.Login
	_, err := l.AuthUserWithToken(e, true)
	if err != nil {
		return "", stackerr.Wrap(err)
	}

	req, err := http.NewRequest(
		"POST",
		"herokuLink",
		ioutil.NopCloser(
			jsonpipe.Encode(
				map[string]string{"herokuAppName": herokuAppName},
			),
		),
	)
	if err != nil {
		return "", stackerr.Wrap(err)
	}
	req.Header = make(http.Header)
	req.Header.Set("X-Parse-Application-Id", h.parseAppID)
	req.Header.Set("X-Parse-Account-Key", l.Credentials.Token)

	resp, err := e.ParseAPIClient.RoundTrip(req)
	if err != nil {
		if herokuAppNameTaken(err) {
			fmt.Fprintln(e.Err, `
Please provide a unique name that might not already be taken on Heroku.
`)
		}
		return "", stackerr.Wrap(err)
	}
	result := &struct {
		Name string `json:"name"`
		ID   string `json:"id"`
	}{}
	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return "", stackerr.Wrap(err)
	}
	if result.Name != herokuAppName || result.ID == "" {
		return "", stackerr.New("could not create heroku app link")
	}
	return result.ID, nil
}

type nameID struct {
	name string
	id   string
}

type nameIDs []nameID

func (n nameIDs) Len() int {
	return len(n)
}

func (n nameIDs) Swap(i, j int) {
	n[i], n[j] = n[j], n[i]
}

func (n nameIDs) Less(i, j int) bool {
	if n[i].name > n[j].name {
		return false
	}
	return n[i].name < n[j].name || n[i].id < n[j].id
}

func (h *herokuLink) herokuAppNames(ids []string, e *parsecli.Env) (nameIDs, []string, error) {
	var wg errgroup.Group
	wg.Add(len(ids))
	maxParallel := make(chan struct{}, maxRequests)

	var (
		ret               nameIDs
		deletedLinks      []string
		retMutex          sync.Mutex
		deletedLinksMutex sync.Mutex
	)

	getAppName := func(id string) {
		defer func() {
			wg.Done()
			<-maxParallel
		}()
		appName, err := parsecli.FetchHerokuAppName(id, e)
		if err != nil {
			if stackerr.HasUnderlying(err, stackerr.MatcherFunc(parsecli.HerokuAppNotFound)) {
				deletedLinksMutex.Lock()
				defer deletedLinksMutex.Unlock()
				deletedLinks = append(deletedLinks, id)
				return
			}
			wg.Error(err) // ignore error if corresponding heroku app was deleted
			return
		}

		retMutex.Lock()
		defer retMutex.Unlock()
		ret = append(ret, nameID{id: id, name: appName})
	}

	for _, id := range ids {
		go getAppName(id)
	}
	err := wg.Wait()
	sort.Sort(ret)
	return ret, deletedLinks, stackerr.Wrap(err)
}

func (h *herokuLink) getAppLinks(e *parsecli.Env) (nameIDs, error) {
	var l parsecli.Login
	_, err := l.AuthUserWithToken(e, true)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(
		"GET",
		"herokuLink",
		nil,
	)
	if err != nil {
		return nil, stackerr.Wrap(err)
	}
	req.Header = make(http.Header)
	req.Header.Set("X-Parse-Application-Id", h.parseAppID)
	req.Header.Set("X-Parse-Account-Key", l.Credentials.Token)

	resp, err := e.ParseAPIClient.RoundTrip(req)
	if err != nil {
		return nil, stackerr.Wrap(err)
	}
	type herokuAppID struct {
		ID string `json:"herokuAppId,omitempty"`
	}
	result := &struct {
		Apps []*herokuAppID `json:"herokuApps"`
	}{}
	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return nil, stackerr.Wrap(err)
	}
	var ret []string
	for _, herokuApp := range result.Apps {
		if herokuApp.ID != "" {
			ret = append(ret, herokuApp.ID)
		}
	}
	names, deletedIDs, err := h.herokuAppNames(ret, e)
	h.removeStaleLinks(e, l.Credentials.Token, deletedIDs)
	return names, err
}

// removeStaleLinks tries to best effort delete stale links
// where corresponding heroku app was deleted
func (h *herokuLink) removeStaleLinks(e *parsecli.Env, token string, removeLinks []string) {
	for _, removeLink := range removeLinks {
		req, err := http.NewRequest(
			"DELETE",
			"herokuLink",
			jsonpipe.Encode(
				map[string]string{"herokuAppId": removeLink},
			),
		)
		if err != nil {
			continue
		}
		req.Header = make(http.Header)
		req.Header.Set("X-Parse-Application-Id", h.parseAppID)
		req.Header.Set("X-Parse-Account-Key", token)

		_, err = e.ParseAPIClient.RoundTrip(req)
		if err != nil {
			continue
		}
	}
}
