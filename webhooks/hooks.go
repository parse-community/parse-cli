package webhooks

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/ParsePlatform/parse-cli/parsecli"
	"github.com/facebookgo/stackerr"
)

var (
	errInvalidFormat = errors.New(
		`
invalid format.
valid formats should look like:
{"hooks": [OPERATION]}

OPERATION ->
{"op": "put", "function": {"functionName": "name", "url": "https_url"}}
{"op": "post", "function": {"functionName": "name", "url": "https_url"}}
{"op": "delete", "function": {"functionName": "name"}}

{"op": "put", "trigger": {"className": "cname", "triggerName": "tname", "url":"https_url"}}
{"op": "post", "trigger": {"className": "cname", "triggerName": "tname", "url":"https_url"}}
{"op": "delete", "trigger": {"className": "cname", "triggerName": "tname"}}
`)

	errPostToPut = errors.New(
		`a hook with given name already exists: cannot create a new one.
`)

	errPutToPost = errors.New(
		`a hook with the given name does not exist yet: cannot update the url.
`)

	errNotExist = errors.New(
		`a hook with the given name does not exist. cannot delete it.
	`)
)

type hookOperation struct {
	Method   string        `json:"op,omitempty"`
	Function *functionHook `json:"function,omitempty"`
	Trigger  *triggerHook  `json:"trigger,omitempty"`
}

func validateURL(urlStr string) error {
	netURL, err := url.Parse(urlStr)
	if err != nil {
		return stackerr.Wrap(err)
	}

	if netURL.Scheme != "https" {
		return errors.New("Please enter a valid https url")
	}
	return nil
}

func checkTriggerName(s string) error {
	switch strings.ToLower(s) {
	case "beforesave", "beforedelete", "aftersave", "afterdelete":
		return nil
	}
	return stackerr.Newf(
		`invalid trigger name: %v.
	This is the list of valid trigger names:
		beforeSave
		afterSave
		beforeDelete
		afterDelete
`,
		s,
	)
}

type Hooks struct {
	HooksStrict    bool
	BaseURL        string
	baseWebhookURL *url.URL
}

func (h *Hooks) checkTriggerName(s string) error {
	switch strings.ToLower(s) {
	case "beforesave", "beforedelete", "aftersave", "afterdelete":
		return nil
	}
	return stackerr.Newf(
		`invalid trigger name: %v.
	This is the list of valid trigger names:
		beforeSave
		afterSave
		beforeDelete
		afterDelete
`,
		s,
	)
}

func (h *Hooks) appendHookOperation(
	e *parsecli.Env,
	hookOp *hookOperation,
	hooksOps []*hookOperation,
) (bool, []*hookOperation, error) {
	if hookOp == nil || (hookOp.Function == nil && hookOp.Trigger == nil) ||
		(hookOp.Function != nil && hookOp.Trigger != nil) {
		return false, hooksOps, nil
	}

	method := strings.ToUpper(hookOp.Method)
	if method != "POST" && method != "PUT" && method != "DELETE" {
		return false, nil, stackerr.Wrap(errInvalidFormat)
	}

	hookOp.Method = method
	if hookOp.Trigger != nil {
		if err := h.checkTriggerName(hookOp.Trigger.TriggerName); err != nil {
			return false, nil, err
		}
	}

	hooksOps = append(hooksOps, hookOp)
	return true, hooksOps, nil
}

func (h *Hooks) createHooksOperations(
	e *parsecli.Env,
	reader io.Reader,
) ([]*hookOperation, error) {
	var input struct {
		HooksOps []*hookOperation `json:"hooks,omitempty"`
	}
	err := json.NewDecoder(ioutil.NopCloser(reader)).Decode(&input)
	if err != nil {
		return nil, stackerr.Wrap(err)
	}

	var (
		hooksOps []*hookOperation
		added    bool
	)
	for _, hookOp := range input.HooksOps {
		added, hooksOps, err = h.appendHookOperation(e, hookOp, hooksOps)
		if err != nil {
			return nil, err
		}
		if !added {
			op, err := json.MarshalIndent(hookOp, "", " ")
			if err == nil {
				fmt.Fprintf(e.Out, "Ignoring hook operation: \n%s\n", op)
			}
		}
	}

	return hooksOps, nil
}

func (h *Hooks) checkStrictMode(restOp string, exists bool) (string, bool, error) {
	restOp = strings.ToUpper(restOp)
	if !exists {
		if restOp == "PUT" {
			if h.HooksStrict {
				return "", false, stackerr.Wrap(errPutToPost)
			}
			return "POST", true, nil
		}
		if restOp == "DELETE" {
			if h.HooksStrict {
				return "", false, stackerr.Wrap(errNotExist)
			}
			return "DELETE", true, nil
		}
	} else if restOp == "POST" {
		if h.HooksStrict {
			return "", false, stackerr.Wrap(errPostToPut)
		}
		return "PUT", true, nil
	}
	return restOp, false, nil
}

func (h *Hooks) functionHookExists(e *parsecli.Env, name string) (bool, error) {
	functionsURL, err := url.Parse(path.Join(defaultFunctionsURL, name))
	if err != nil {
		return false, stackerr.Wrap(err)
	}
	var results struct {
		Results []*functionHook `json:"results,omitempty"`
	}
	_, err = e.ParseAPIClient.Get(functionsURL, &results)
	if err != nil {
		if strings.Contains(err.Error(), "is defined") {
			return false, nil
		}
		return false, stackerr.Wrap(err)
	}
	for _, result := range results.Results {
		if result.URL != "" && result.FunctionName == name {
			return true, nil
		}
	}
	return false, nil
}

func (h *Hooks) deployFunctionHook(e *parsecli.Env, op *hookOperation) error {
	if op.Function == nil {
		return stackerr.New("cannot deploy nil function hook")
	}
	exists, err := h.functionHookExists(e, op.Function.FunctionName)
	if err != nil {
		return err
	}

	restOp, suppressed, err := h.checkStrictMode(op.Method, exists)
	if err != nil {
		return err
	}

	function := &functionHooksCmd{Function: op.Function}
	switch restOp {
	case "POST":
		return function.functionHooksCreate(e, nil)
	case "PUT":
		return function.functionHooksUpdate(e, nil)
	case "DELETE":
		if suppressed {
			return nil
		}
		return function.functionHooksDelete(e, nil)
	}
	return stackerr.Wrap(errInvalidFormat)
}

func (h *Hooks) triggerHookExists(e *parsecli.Env, className, triggerName string) (bool, error) {
	triggersURL, err := url.Parse(path.Join(defaultTriggersURL, className, triggerName))
	if err != nil {
		return false, stackerr.Wrap(err)
	}
	var results struct {
		Results []*triggerHook `json:"results,omitempty"`
	}
	_, err = e.ParseAPIClient.Get(triggersURL, &results)
	if err != nil {
		if strings.Contains(err.Error(), "is defined") {
			return false, nil
		}
		return false, stackerr.Wrap(err)
	}
	for _, result := range results.Results {
		if result.URL != "" && result.ClassName == className && result.TriggerName == triggerName {
			return true, nil
		}
	}
	return false, nil
}

func (h *Hooks) deployTriggerHook(e *parsecli.Env, op *hookOperation) error {
	if op.Trigger == nil {
		return stackerr.New("cannot deploy nil trigger hook")
	}

	exists, err := h.triggerHookExists(e, op.Trigger.ClassName, op.Trigger.TriggerName)
	if err != nil {
		return err
	}
	restOp, suppressed, err := h.checkStrictMode(op.Method, exists)
	if err != nil {
		return err
	}

	trigger := &triggerHooksCmd{Trigger: op.Trigger, All: false}
	switch restOp {
	case "POST":
		return trigger.triggerHooksCreate(e, nil)
	case "PUT":
		return trigger.triggerHooksUpdate(e, nil)
	case "DELETE":
		if suppressed {
			return nil
		}
		return trigger.triggerHooksDelete(e, nil)
	}
	return stackerr.Wrap(errInvalidFormat)

}

func (h *Hooks) deployWebhooksConfig(e *parsecli.Env, hooksOps []*hookOperation) error {
	for _, op := range hooksOps {
		if op.Function == nil && op.Trigger == nil {
			return stackerr.New("hook operation is neither a function, not a trigger.")
		}
		if op.Function != nil && op.Trigger != nil {
			return stackerr.New("a hook cannot be both a function and a trigger.")
		}
		if op.Function != nil {
			if err := h.deployFunctionHook(e, op); err != nil {
				return err
			}
		} else {
			if err := h.deployTriggerHook(e, op); err != nil {
				return err
			}
		}
		fmt.Fprintln(e.Out)
	}
	return nil
}

func (h *Hooks) parseBaseURL(e *parsecli.Env) error {
	if h.BaseURL != "" {
		u, err := url.Parse(h.BaseURL)
		if err != nil {
			fmt.Fprintln(e.Err, "Invalid base webhook url provided")
			return stackerr.Wrap(err)
		}
		if u.Scheme != "https" {
			return stackerr.New("Please provide a valid https url")
		}
		h.baseWebhookURL = u
	}
	return nil
}

func (h *Hooks) HooksCmd(e *parsecli.Env, ctx *parsecli.Context, args []string) error {
	if len(args) > 1 {
		return fmt.Errorf("Invalid args: %v, only an optional hooks config file is expected.", args)
	}
	if err := h.parseBaseURL(e); err != nil {
		return err
	}

	reader := e.In
	if len(args) == 1 {
		file, err := os.Open(args[0])
		if err != nil {
			return stackerr.Wrap(err)
		}
		reader = file
	} else {
		fmt.Fprintln(e.Out, "Since a webhooks config file was not provided reading from stdin.")
	}
	hooksOps, err := h.createHooksOperations(e, reader)
	if err != nil {
		return err
	}
	err = h.deployWebhooksConfig(e, hooksOps)
	if err != nil {
		fmt.Fprintln(
			e.Out,
			"Failed to deploy the webhooks config. Please try again...",
		)
		return err
	}
	fmt.Fprintln(e.Out, "Successfully configured the given webhooks for the app.")
	return nil
}
