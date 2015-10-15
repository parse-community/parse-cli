package webhooks

import (
	"encoding/json"
	"errors"
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

func newFunctionsHarness(t testing.TB) *parsecli.Harness {
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
			if r.URL.Path == "/1/hooks/functions/foo" {
				body = map[string]interface{}{
					"results": []map[string]interface{}{
						{"functionName": "foo"},
						{"functionName": "foo", "url": "https://api.example.com/foo"},
					},
				}
			} else if r.URL.Path == defaultFunctionsURL {
				body = map[string]interface{}{
					"results": []map[string]interface{}{
						{"functionName": "foo"},
						{"functionName": "foo", "url": "https://api.example.com/foo"},
						{"functionName": "bar", "url": "https://api.example.com/bar"},
					},
				}
			} else {
				return &http.Response{StatusCode: http.StatusBadRequest},
					errors.New("no such function hook is defined")

			}
		case "POST":
			ensure.DeepEqual(t, r.URL.Path, defaultFunctionsURL)
			switch params["functionName"] {
			case "foo":
				body = map[string]interface{}{
					"functionName": "foo",
					"url":          "https://api.example.com/foo",
					"warning":      "function foo already exists",
				}
			case "bar":
				body = map[string]interface{}{
					"functionName": "bar",
					"url":          "https://api.example.com/bar",
				}
			default:
				return &http.Response{StatusCode: http.StatusInternalServerError},
					errors.New("invalid function name")
			}
		case "PUT":
			if params["__op"] == "Delete" && strings.HasPrefix(r.URL.Path, "/foo") {
				ensure.DeepEqual(t, r.URL.Path, "/1/hooks/functions/foo")
				body = map[string]interface{}{"functionName": "foo"}
			} else {
				switch strings.Replace(r.URL.Path, "/1/hooks/functions/", "", 1) {
				case "foo":
					ensure.DeepEqual(t, r.URL.Path, "/1/hooks/functions/foo")
					body = map[string]interface{}{
						"functionName": "foo",
						"url":          "https://api.example.com/_foo",
						"warning":      "function foo already exists",
					}
				case "bar":
					ensure.DeepEqual(t, r.URL.Path, "/1/hooks/functions/bar")
					body = map[string]interface{}{
						"functionName": "bar",
						"url":          "https://api.example.com/_bar",
					}
				default:
					return &http.Response{StatusCode: http.StatusInternalServerError},
						errors.New("invalid function name")
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

func TestFunctionHookString(t *testing.T) {
	t.Parallel()
	f := functionHook{FunctionName: "foo"}
	ensure.DeepEqual(t, f.String(), `Function name: "foo"`)

	f.URL = "https://api.example.com/foo"
	ensure.DeepEqual(t,
		f.String(),
		`Function name: "foo", URL: "https://api.example.com/foo"`)
}

func TestReadFuncParams(t *testing.T) {
	t.Parallel()

	h := parsecli.NewHarness(t)
	defer h.Stop()

	h.Env.In = strings.NewReader("\n")
	_, err := readFunctionName(h.Env, nil)
	ensure.Err(t, err, regexp.MustCompile("Function name cannot be empty"))

	h.Env.In = strings.NewReader("foo\n")
	hook, err := readFunctionName(h.Env, nil)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, *hook, functionHook{FunctionName: "foo"})

	h.Env.In = strings.NewReader("foo\napi.example.com/foo\n")
	hook, err = readFunctionParams(h.Env, nil)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, *hook, functionHook{
		FunctionName: "foo",
		URL:          "https://api.example.com/foo",
	})
}

func TestFunctionHooksRead(t *testing.T) {
	t.Parallel()

	h := newFunctionsHarness(t)

	f := functionHooksCmd{All: true}
	ensure.Nil(t, f.functionHooksRead(h.Env, nil))
	ensure.DeepEqual(t, h.Out.String(), `The following cloudcode or webhook functions are associated with this app:
Function name: "bar", URL: "https://api.example.com/bar"
Function name: "foo"
Function name: "foo", URL: "https://api.example.com/foo"
`)

	f.All = false
	h.Env.In = strings.NewReader("foo\napi.example.com/foo\n")
	h.Out.Reset()
	ensure.Nil(t, f.functionHooksRead(h.Env, nil))
	ensure.DeepEqual(t, h.Out.String(), `Please enter the function name: The following functions named: "foo" are associated with your app:
Function name: "foo"
Function name: "foo", URL: "https://api.example.com/foo"
`)
}

func TestFunctionHooksCreate(t *testing.T) {
	t.Parallel()

	h := newFunctionsHarness(t)

	var f functionHooksCmd
	h.Env.In = strings.NewReader("foo\napi.example.com/foo\n")
	ensure.Nil(t, f.functionHooksCreate(h.Env, nil))
	ensure.DeepEqual(t,
		h.Out.String(),
		`Please enter the function name: URL: https://Successfully created a webhook function "foo" pointing to "https://api.example.com/foo"
`)
	ensure.DeepEqual(t, h.Err.String(), "WARNING: function foo already exists\n")

	h.Out.Reset()
	h.Err.Reset()

	h.Env.In = strings.NewReader("bar\napi.example.com/bar\n")
	ensure.Nil(t, f.functionHooksCreate(h.Env, nil))
	ensure.DeepEqual(t,
		h.Out.String(),
		`Please enter the function name: URL: https://Successfully created a webhook function "bar" pointing to "https://api.example.com/bar"
`)
	ensure.DeepEqual(t, h.Err.String(), "")
}

func TestFunctionHooksUpdate(t *testing.T) {
	t.Parallel()

	h := newFunctionsHarness(t)

	var f functionHooksCmd
	h.Env.In = strings.NewReader("foo\napi.example.com/_foo\n")
	ensure.Nil(t, f.functionHooksUpdate(h.Env, nil))
	ensure.DeepEqual(t,
		h.Out.String(),
		`Please enter the function name: URL: https://Successfully update the webhook function "foo" to point to "https://api.example.com/_foo"
`)

	ensure.DeepEqual(t, h.Err.String(), "WARNING: function foo already exists\n")

	h.Out.Reset()
	h.Err.Reset()

	h.Env.In = strings.NewReader("bar\napi.example.com/_bar\n")
	ensure.Nil(t, f.functionHooksUpdate(h.Env, nil))
	ensure.DeepEqual(t,
		h.Out.String(),
		`Please enter the function name: URL: https://Successfully update the webhook function "bar" to point to "https://api.example.com/_bar"
`)

	ensure.DeepEqual(t, h.Err.String(), "")
}

func TestFunctionHookDelete(t *testing.T) {
	t.Parallel()

	h := newFunctionsHarness(t)

	f := &functionHooksCmd{interactive: true}
	h.Env.In = strings.NewReader("foo\ny\n")
	ensure.Nil(t, f.functionHooksDelete(h.Env, nil))
	ensure.DeepEqual(t,
		h.Out.String(),
		`Please enter the function name: Are you sure you want to delete webhook function: "foo" (y/n): Successfully deleted webhook function "foo"
Function "foo" defined in Cloud Code will be used henceforth
`)

	h.Out.Reset()
	h.Env.In = strings.NewReader("foo\nn\n")
	ensure.Nil(t, f.functionHooksDelete(h.Env, nil))
	ensure.DeepEqual(t,
		h.Out.String(),
		`Please enter the function name: Are you sure you want to delete webhook function: "foo" (y/n): `)
}
