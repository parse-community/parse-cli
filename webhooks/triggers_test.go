package webhooks

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/ParsePlatform/parse-cli/parsecli"
	"github.com/facebookgo/ensure"
	"github.com/facebookgo/jsonpipe"
	"github.com/facebookgo/parse"
)

func newTriggersHarness(t testing.TB) *parsecli.Harness {
	h := parsecli.NewHarness(t)
	defer h.Stop()

	ht := parsecli.TransportFunc(func(r *http.Request) (*http.Response, error) {
		var body interface{}
		var params map[string]interface{}
		if r.Body != nil {
			err := json.NewDecoder(r.Body).Decode(&params)
			if err != nil {
				return &http.Response{StatusCode: http.StatusInternalServerError}, err
			}
		}

		switch r.Method {
		case "GET":
			if r.URL.Path == "/1/hooks/triggers/foo/beforeSave" {
				body = map[string]interface{}{
					"results": []map[string]interface{}{
						{"className": "foo", "triggerName": "beforeSave"},
						{"className": "foo", "triggerName": "beforeSave", "url": "https://api.example.com/foo/beforeSave"},
					},
				}
			} else if r.URL.Path == defaultTriggersURL {
				body = map[string]interface{}{
					"results": []map[string]interface{}{
						{"className": "foo", "triggerName": "beforeSave"},
						{"className": "foo", "triggerName": "beforeSave", "url": "https://api.example.com/foo/beforeSave"},
						{"className": "bar", "triggerName": "afterSave", "url": "https://api.example.com/bar/afterSave"},
					},
				}
			} else {
				return &http.Response{StatusCode: http.StatusBadRequest},
					errors.New("no such trigger is defined for class")
			}
		case "POST":
			ensure.DeepEqual(t, r.URL.Path, defaultTriggersURL)
			switch fmt.Sprintf("%v:%v", params["className"], params["triggerName"]) {
			case "foo:beforeSave":
				body = map[string]interface{}{
					"className":   "foo",
					"triggerName": "beforeSave",
					"url":         "https://api.example.com/foo/beforeSave",
					"warning":     "beforeSave trigger already exists for class: foo",
				}
			case "bar:afterSave":
				body = map[string]interface{}{
					"className":   "bar",
					"triggerName": "afterSave",
					"url":         "https://api.example.com/bar/afterSave",
				}
			default:
				return &http.Response{StatusCode: http.StatusInternalServerError},
					errors.New("invalid params")
			}
		case "PUT":
			if params["__op"] == "Delete" && strings.HasPrefix(r.URL.Path, "/foo/beforeSave") {
				ensure.DeepEqual(t, r.URL.Path, "/1/hooks/triggers/foo/beforeSave")
				body = map[string]interface{}{"className": "foo", "triggerName": "beforeSave"}
			} else {
				switch strings.Replace(r.URL.Path, defaultTriggersURL, "", 1) {
				case "/foo/beforeSave":
					ensure.DeepEqual(t, r.URL.Path, "/1/hooks/triggers/foo/beforeSave")
					body = map[string]interface{}{
						"className":   "foo",
						"triggerName": "beforeSave",
						"url":         "https://api.example.com/_foo/beforeSave",
						"warning":     "beforeSave trigger already exists for class: foo",
					}
				case "/bar/afterSave":
					ensure.DeepEqual(t, r.URL.Path, "/1/hooks/triggers/bar/afterSave")
					body = map[string]interface{}{
						"className":   "bar",
						"triggerName": "afterSave",
						"url":         "https://api.example.com/_bar/afterSave",
					}
				default:
					return &http.Response{StatusCode: http.StatusInternalServerError},
						errors.New("invalid params")
				}
			}
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(jsonpipe.Encode(body)),
		}, nil
	})
	h.Env.ParseAPIClient = &parsecli.ParseAPIClient{APIClient: &parse.Client{Transport: ht}}
	return h
}

func TestTriggerHookString(t *testing.T) {
	t.Parallel()
	f := triggerHook{ClassName: "foo", TriggerName: "beforeSave"}
	ensure.DeepEqual(t, f.String(), `Class name: "foo", Trigger name: "beforeSave"`)

	f.URL = "https://api.example.com/foo/beforeSave"
	ensure.DeepEqual(t,
		f.String(),
		`Class name: "foo", Trigger name: "beforeSave", URL: "https://api.example.com/foo/beforeSave"`)
}

func TestReadTriggerParams(t *testing.T) {
	t.Parallel()

	h := parsecli.NewHarness(t)
	defer h.Stop()

	h.Env.In = ioutil.NopCloser(strings.NewReader("\n"))
	_, err := readTriggerName(h.Env, nil)
	ensure.Err(t, err, regexp.MustCompile("Class name cannot be empty"))

	h.Env.In = ioutil.NopCloser(strings.NewReader("foo\n"))
	_, err = readTriggerName(h.Env, nil)
	ensure.Err(t, err, regexp.MustCompile("Trigger name cannot be empty"))

	h.Env.In = ioutil.NopCloser(strings.NewReader("foo\nbeforeSave"))
	hook, err := readTriggerName(h.Env, nil)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, *hook, triggerHook{ClassName: "foo", TriggerName: "beforeSave"})

	h.Env.In = ioutil.NopCloser(strings.NewReader("foo\nbeforeSave\napi.example.com/foo/beforeSave\n"))
	hook, err = readTriggerParams(h.Env, nil)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, *hook, triggerHook{
		ClassName:   "foo",
		TriggerName: "beforeSave",
		URL:         "https://api.example.com/foo/beforeSave",
	})
}

func TestTriggerHooksRead(t *testing.T) {
	t.Parallel()

	h := newTriggersHarness(t)

	tr := triggerHooksCmd{All: true}
	ensure.Nil(t, tr.triggerHooksRead(h.Env, nil))
	ensure.DeepEqual(t, h.Out.String(), `The following cloudcode or webhook triggers are associated with this app:
Class name: "bar", Trigger name: "afterSave", URL: "https://api.example.com/bar/afterSave"
Class name: "foo", Trigger name: "beforeSave"
Class name: "foo", Trigger name: "beforeSave", URL: "https://api.example.com/foo/beforeSave"
`)

	tr.All = false
	h.Env.In = ioutil.NopCloser(strings.NewReader("foo\nbeforeSave\n"))
	h.Out.Reset()
	ensure.Nil(t, tr.triggerHooksRead(h.Env, nil))
	ensure.DeepEqual(t, h.Out.String(), `Please enter following details about the trigger webhook
Class name: Trigger name: The following triggers named: "beforeSave" are associated with the class: "foo"
Class name: "foo", Trigger name: "beforeSave"
Class name: "foo", Trigger name: "beforeSave", URL: "https://api.example.com/foo/beforeSave"
`)
}

func TestTriggerHooksCreate(t *testing.T) {
	t.Parallel()

	h := newTriggersHarness(t)

	var tr triggerHooksCmd
	h.Env.In = ioutil.NopCloser(strings.NewReader("foo\nbeforeSave\napi.example.com/foo/beforeSave\n"))
	ensure.Nil(t, tr.triggerHooksCreate(h.Env, nil))
	ensure.DeepEqual(t,
		h.Out.String(),
		`Please enter following details about the trigger webhook
Class name: Trigger name: URL: https://Successfully created a "beforeSave" trigger for class "foo" pointing to "https://api.example.com/foo/beforeSave"
`)

	ensure.DeepEqual(t, h.Err.String(), "WARNING: beforeSave trigger already exists for class: foo\n")

	h.Out.Reset()
	h.Err.Reset()

	h.Env.In = ioutil.NopCloser(strings.NewReader("bar\nafterSave\napi.example.com/bar/afterSave\n"))
	ensure.Nil(t, tr.triggerHooksCreate(h.Env, nil))
	ensure.DeepEqual(t,
		h.Out.String(),
		`Please enter following details about the trigger webhook
Class name: Trigger name: URL: https://Successfully created a "afterSave" trigger for class "bar" pointing to "https://api.example.com/bar/afterSave"
`)
	ensure.DeepEqual(t, h.Err.String(), "")
}

func TestTriggerHooksUpdate(t *testing.T) {
	t.Parallel()

	h := newTriggersHarness(t)

	var tr triggerHooksCmd
	h.Env.In =
		ioutil.NopCloser(strings.NewReader("foo\nbeforeSave\napi.example.com/_foo/beforeSave\n"))
	ensure.Nil(t, tr.triggerHooksUpdate(h.Env, nil))
	ensure.DeepEqual(t,
		h.Out.String(),
		`Please enter following details about the trigger webhook
Class name: Trigger name: URL: https://Successfully update the "beforeSave" trigger for class "foo" to point to "https://api.example.com/_foo/beforeSave"
`)

	ensure.DeepEqual(t, h.Err.String(), "WARNING: beforeSave trigger already exists for class: foo\n")

	h.Out.Reset()
	h.Err.Reset()

	h.Env.In =
		ioutil.NopCloser(strings.NewReader("bar\nafterSave\napi.example.com/_bar/afterSave\n"))
	ensure.Nil(t, tr.triggerHooksUpdate(h.Env, nil))
	ensure.DeepEqual(t,
		h.Out.String(),
		`Please enter following details about the trigger webhook
Class name: Trigger name: URL: https://Successfully update the "afterSave" trigger for class "bar" to point to "https://api.example.com/_bar/afterSave"
`)

	ensure.DeepEqual(t, h.Err.String(), "")
}

func TestTriggerHookDelete(t *testing.T) {
	t.Parallel()

	h := newTriggersHarness(t)

	tr := triggerHooksCmd{interactive: true}
	h.Env.In = ioutil.NopCloser(strings.NewReader("foo\nbeforeSave\ny\n"))
	ensure.Nil(t, tr.triggerHooksDelete(h.Env, nil))
	ensure.DeepEqual(t,
		h.Out.String(),
		`Please enter following details about the trigger webhook
Class name: Trigger name: Are you sure you want to delete "beforeSave" webhook trigger for class: "foo" (y/n): Successfully deleted "beforeSave" webhook trigger for class "foo"
"beforeSave" trigger defined in cloudcode for class "foo" will be used henceforth
`)

	h.Out.Reset()
	h.Env.In = ioutil.NopCloser(strings.NewReader("foo\nn\n"))
	ensure.Nil(t, tr.triggerHooksDelete(h.Env, nil))
	ensure.DeepEqual(t,
		h.Out.String(),
		`Please enter following details about the trigger webhook
Class name: Trigger name: Are you sure you want to delete "n" webhook trigger for class: "foo" (y/n): `)
}
