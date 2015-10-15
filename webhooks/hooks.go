package webhooks

import (
	"bufio"
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
[put|post],functionName,https_url
delete,functionName

[put|post],className:triggerName,https_url
delete,className:triggerName
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
	method   string
	function *functionHook
	trigger  *triggerHook
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

func (h *Hooks) postOrPutHook(
	e *parsecli.Env,
	hooksOps []*hookOperation,
	fields ...string,
) (bool, []*hookOperation, error) {
	restOp := strings.ToUpper(fields[0])
	if restOp != "POST" && restOp != "PUT" {
		return false, nil, stackerr.Wrap(errInvalidFormat)
	}

	switch len(fields) {
	case 3:
		hooksOps = append(hooksOps, &hookOperation{
			method:   restOp,
			function: &functionHook{FunctionName: fields[1], URL: fields[2]},
		})
		return true, hooksOps, nil

	case 4:
		if err := h.checkTriggerName(fields[2]); err != nil {
			return false, nil, err
		}
		hooksOps = append(hooksOps, &hookOperation{
			method:  restOp,
			trigger: &triggerHook{ClassName: fields[1], TriggerName: fields[2], URL: fields[3]},
		})
		return true, hooksOps, nil
	}
	return false, nil, stackerr.Wrap(errInvalidFormat)

}

func (h *Hooks) deleteHook(
	e *parsecli.Env,
	hooksOps []*hookOperation,
	fields ...string,
) (bool, []*hookOperation, error) {
	restOp := strings.ToUpper(fields[0])
	if restOp != "DELETE" {
		return false, nil, stackerr.Wrap(errInvalidFormat)
	}

	switch len(fields) {
	case 2:
		hooksOps = append(hooksOps, &hookOperation{
			method:   "DELETE",
			function: &functionHook{FunctionName: fields[1]},
		})
		return true, hooksOps, nil
	case 3:
		if err := h.checkTriggerName(fields[2]); err != nil {
			return false, nil, err
		}
		hooksOps = append(hooksOps, &hookOperation{
			method:  "DELETE",
			trigger: &triggerHook{ClassName: fields[1], TriggerName: fields[2]},
		})
		return true, hooksOps, nil
	}

	return false, nil, stackerr.Wrap(errInvalidFormat)
}

func (h *Hooks) appendHookOperation(
	e *parsecli.Env,
	fields []string,
	hooks []*hookOperation,
) (bool, []*hookOperation, error) {
	if len(fields) == 0 {
		return false, hooks, nil
	}

	switch strings.ToLower(fields[0]) {
	case "post", "put":
		return h.postOrPutHook(e, hooks, fields...)

	case "delete":
		return h.deleteHook(e, hooks, fields...)
	}
	return false, nil, stackerr.Wrap(errInvalidFormat)
}

func (h *Hooks) processHooksOperation(e *parsecli.Env, op string) ([]string, error) {
	op = strings.TrimSpace(op)
	if op == "" {
		return nil, nil
	}

	fields := strings.SplitN(op, ",", 3)
	switch restOp := strings.ToLower(fields[0]); restOp {
	case "post", "put":
		if len(fields) < 3 {
			return nil, stackerr.Wrap(errInvalidFormat)
		}
	case "delete":
		if len(fields) != 2 {
			return nil, stackerr.Wrap(errInvalidFormat)
		}
		subFields := strings.SplitN(fields[1], ":", 2)
		if len(subFields) == 2 {
			fields = append(fields, subFields[1])
			fields[1] = subFields[0]
		}
		return fields, nil
	default:
		return nil, stackerr.Wrap(errInvalidFormat)
	}

	if h.baseWebhookURL != nil {
		u, err := h.baseWebhookURL.Parse(fields[2])
		if err != nil {
			return nil, stackerr.Wrap(err)
		}
		fields[2] = u.String()
	}

	switch subFields := strings.SplitN(fields[1], ":", 2); len(subFields) {
	case 1:
		return fields, nil
	case 2:
		fields = append(fields, fields[2])
		fields[3] = fields[2]
		fields[2] = subFields[1]
		fields[1] = subFields[0]
		return fields, nil
	}
	return nil, stackerr.Wrap(errInvalidFormat)
}

func (h *Hooks) createHooksOperations(
	e *parsecli.Env,
	reader io.Reader,
) ([]*hookOperation, error) {
	scanner := bufio.NewScanner(reader)
	var (
		hooksOps []*hookOperation
		added    bool
	)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		fields, err := h.processHooksOperation(e, scanner.Text())
		if err != nil {
			return nil, err
		}
		added, hooksOps, err = h.appendHookOperation(e, fields, hooksOps)
		if err != nil {
			return nil, err
		}
		if !added {
			fmt.Fprintf(e.Out, "Ignoring line: %d\n", lineNum)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, stackerr.Wrap(err)
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
	if op.function == nil {
		return stackerr.New("cannot deploy nil function hook")
	}
	exists, err := h.functionHookExists(e, op.function.FunctionName)
	if err != nil {
		return err
	}

	restOp, suppressed, err := h.checkStrictMode(op.method, exists)
	if err != nil {
		return err
	}

	function := &functionHooksCmd{Function: op.function}
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
	if op.trigger == nil {
		return stackerr.New("cannot deploy nil trigger hook")
	}

	exists, err := h.triggerHookExists(e, op.trigger.ClassName, op.trigger.TriggerName)
	if err != nil {
		return err
	}
	restOp, suppressed, err := h.checkStrictMode(op.method, exists)
	if err != nil {
		return err
	}

	trigger := &triggerHooksCmd{Trigger: op.trigger, All: false}
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
		if op.function == nil && op.trigger == nil {
			return stackerr.New("hook operation is neither a function, not a trigger.")
		}
		if op.function != nil && op.trigger != nil {
			return stackerr.New("a hook cannot be both a function and a trigger.")
		}
		if op.function != nil {
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
		reader = ioutil.NopCloser(file)
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
