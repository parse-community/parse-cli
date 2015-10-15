package webhooks

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/ParsePlatform/parse-cli/parsecli"
	"github.com/facebookgo/ensure"
)

func TestCheckTriggerName(t *testing.T) {
	t.Parallel()

	c := &Hooks{}

	ensure.Nil(t, c.checkTriggerName("beforeSave"))
	ensure.Nil(t, c.checkTriggerName("afterSave"))
	ensure.Nil(t, c.checkTriggerName("beforeDelete"))
	ensure.Nil(t, c.checkTriggerName("afterDelete"))

	ensure.Nil(t, c.checkTriggerName("BeforeSAVE"))
	ensure.Nil(t, c.checkTriggerName("AFTERsave"))
	ensure.Nil(t, c.checkTriggerName("BeforeDELETE"))
	ensure.Nil(t, c.checkTriggerName("AFTERdelete"))

	ensure.Err(t, c.checkTriggerName("invalid"), regexp.MustCompile("list of valid trigger names"))
}

func TestAppendHookOperation(t *testing.T) {
	t.Parallel()

	h := parsecli.NewHarness(t)
	defer h.Stop()

	c := &Hooks{}

	var hooksOps []*hookOperation
	_, ops, err := c.appendHookOperation(h.Env, nil, hooksOps)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, hooksOps, ops)

	_, ops, err = c.appendHookOperation(h.Env, &hookOperation{}, nil)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, hooksOps, ops)

	_, ops, err = c.appendHookOperation(h.Env,
		&hookOperation{
			Method: "post",
			Function: &functionHook{
				FunctionName: "call",
				URL:          "https://twilio.com/call",
			},
		},
		hooksOps,
	)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, ops[len(ops)-1].Method, "POST")
	ensure.DeepEqual(
		t,
		*ops[len(ops)-1].Function,
		functionHook{FunctionName: "call", URL: "https://twilio.com/call"},
	)

	_, ops, err = c.appendHookOperation(h.Env,
		&hookOperation{
			Method: "put",
			Function: &functionHook{
				FunctionName: "call",
				URL:          "https://twilio.com/call_1",
			},
		},
		hooksOps,
	)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, ops[len(ops)-1].Method, "PUT")
	ensure.DeepEqual(
		t,
		*ops[len(ops)-1].Function,
		functionHook{FunctionName: "call", URL: "https://twilio.com/call_1"},
	)

	//random stuff
	_, ops, err = c.appendHookOperation(h.Env,
		&hookOperation{
			Method: "posT",
			Trigger: &triggerHook{
				ClassName:   "_User",
				TriggerName: "afterDelete",
				URL:         "https://twilio.com/message",
			},
		},
		hooksOps,
	)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, ops[len(ops)-1].Method, "POST")
	ensure.DeepEqual(
		t,
		*ops[len(ops)-1].Trigger,
		triggerHook{ClassName: "_User", TriggerName: "afterDelete", URL: "https://twilio.com/message"},
	)

	_, ops, err = c.appendHookOperation(h.Env,
		&hookOperation{
			Method: "pUt",
			Trigger: &triggerHook{
				ClassName:   "_User",
				TriggerName: "afterDelete",
				URL:         "https://twilio.com/message_1",
			},
		},
		hooksOps,
	)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, ops[len(ops)-1].Method, "PUT")
	ensure.DeepEqual(
		t,
		*ops[len(ops)-1].Trigger,
		triggerHook{ClassName: "_User", TriggerName: "afterDelete", URL: "https://twilio.com/message_1"},
	)

	// random stuff
	_, ops, err = c.appendHookOperation(h.Env,
		&hookOperation{
			Method: "pUt",
			Trigger: &triggerHook{
				ClassName:   "_User",
				TriggerName: "afterDelete",
				URL:         "https://twilio.com/message_1,message_2",
			},
		},
		hooksOps,
	)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, ops[len(ops)-1].Method, "PUT")
	ensure.DeepEqual(
		t,
		*ops[len(ops)-1].Trigger,
		triggerHook{ClassName: "_User", TriggerName: "afterDelete", URL: "https://twilio.com/message_1,message_2"},
	)
}

func TestCheckStrictMode(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		restOp     string
		exists     bool
		strictMode bool

		op      string
		updated bool
		err     *regexp.Regexp
	}{
		{exists: false, restOp: "put", strictMode: false, op: "POST", updated: true, err: nil},
		{exists: false, restOp: "post", strictMode: false, op: "POST", updated: false, err: nil},
		{exists: false, restOp: "delete", strictMode: false, op: "DELETE", updated: true, err: nil},
		{exists: true, restOp: "put", strictMode: false, op: "PUT", updated: false, err: nil},
		{exists: true, restOp: "post", strictMode: false, op: "PUT", updated: true, err: nil},
		{exists: true, restOp: "delete", strictMode: false, op: "DELETE", updated: false, err: nil},

		{exists: false, restOp: "put", strictMode: true, err: regexp.MustCompile("does not exist")},
		{exists: false, restOp: "post", strictMode: true, op: "POST", updated: false, err: nil},
		{exists: false, restOp: "delete", strictMode: true, err: regexp.MustCompile("cannot delete")},
		{exists: true, restOp: "put", strictMode: true, op: "PUT", updated: false, err: nil},
		{exists: true, restOp: "post", strictMode: true, err: regexp.MustCompile("already exists")},
		{exists: true, restOp: "delete", strictMode: true, op: "DELETE", updated: false, err: nil},
	}

	c := &Hooks{}
	for _, testCase := range testCases {
		c.HooksStrict = testCase.strictMode
		op, updated, err := c.checkStrictMode(testCase.restOp, testCase.exists)
		if testCase.err == nil {
			ensure.Nil(t, err)
			ensure.DeepEqual(t, op, testCase.op)
			ensure.DeepEqual(t, updated, testCase.updated)
		} else {
			ensure.Err(t, err, testCase.err)
		}
	}
}

func TestCreateHookOperations(t *testing.T) {
	t.Parallel()
	h := parsecli.NewHarness(t)
	h.MakeEmptyRoot()
	defer h.Stop()

	c := &Hooks{}

	hooksConfigFile := filepath.Join(h.Env.Root, "webhooks.json")
	err := ioutil.WriteFile(
		hooksConfigFile,
		[]byte("{}"),
		0666,
	)
	ensure.Nil(t, err)

	f, err := os.Open(hooksConfigFile)
	ensure.Nil(t, err)
	ops, err := c.createHooksOperations(h.Env, f)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, len(ops), 0)
	ensure.Nil(t, f.Close())

	err = ioutil.WriteFile(
		hooksConfigFile,
		[]byte("{1}"),
		0666,
	)
	ensure.Nil(t, err)

	f, err = os.Open(hooksConfigFile)
	ensure.Nil(t, err)
	ops, err = c.createHooksOperations(h.Env, f)
	ensure.Err(t, err, regexp.MustCompile("invalid"))
	ensure.DeepEqual(t, len(ops), 0)
	ensure.Nil(t, f.Close())

	err = ioutil.WriteFile(
		hooksConfigFile,
		[]byte(`{
			"hooks": [
			{
				"op": "put",
				"function": {
					"functionName": "message",
					"url": "https://api.twilio.com/message"
				}
			}
		]}`),
		0666,
	)

	f, err = os.Open(hooksConfigFile)
	ensure.Nil(t, err)
	ops, err = c.createHooksOperations(h.Env, f)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, len(ops), 1)
	ensure.DeepEqual(t,
		*ops[0].Function,
		functionHook{FunctionName: "message", URL: "https://api.twilio.com/message"},
	)
	ensure.DeepEqual(t, ops[0].Method, "PUT")
	ensure.Nil(t, f.Close())
	h.Out.Reset()

	err = ioutil.WriteFile(
		hooksConfigFile,
		[]byte(`{
			"hooks": [
			{
				"op": "post",
				"function": {
					"functionName": "message",
					"url": "https://api.twilio.com/message"
				}
			},
			{
				"op": "put",
				"function": {
					"functionName": "message",
					"url": "https://api.twilio.com/message_1"
				}
			},
			{
				"op": "post"
			}
		]}`),
		0666,
	)

	f, err = os.Open(hooksConfigFile)
	ensure.Nil(t, err)
	ops, err = c.createHooksOperations(h.Env, f)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, len(ops), 2)
	ensure.DeepEqual(t,
		*ops[0].Function,
		functionHook{FunctionName: "message", URL: "https://api.twilio.com/message"},
	)
	ensure.DeepEqual(t, ops[0].Method, "POST")

	ensure.DeepEqual(t,
		*ops[1].Function,
		functionHook{FunctionName: "message", URL: "https://api.twilio.com/message_1"},
	)
	ensure.DeepEqual(t,
		h.Out.String(),
		`Ignoring hook operation: 
{
 "op": "post"
}
`,
	)
	ensure.DeepEqual(t, ops[1].Method, "PUT")
	ensure.Nil(t, f.Close())
	h.Out.Reset()

}

func TestFunctionHookExists(t *testing.T) {
	t.Parallel()
	h := newFunctionsHarness(t)
	c := &Hooks{}
	exists, err := c.functionHookExists(h.Env, "foo")
	ensure.Nil(t, err)
	ensure.True(t, exists)

	exists, err = c.functionHookExists(h.Env, "bar")
	ensure.Nil(t, err)
	ensure.False(t, exists)
}

func TestDeployFunctionHooks(t *testing.T) {
	t.Parallel()
	h := newFunctionsHarness(t)
	c := &Hooks{}

	err := c.deployFunctionHook(
		h.Env,
		&hookOperation{
			Method: "post",
			Function: &functionHook{
				FunctionName: "foo",
				URL:          "https://api.example.com/foo",
			},
		},
	)
	// not an error -> will be converted to put
	ensure.Nil(t, err)
}

func TestTriggerHookExists(t *testing.T) {
	t.Parallel()
	h := newTriggersHarness(t)
	c := &Hooks{}
	exists, err := c.triggerHookExists(h.Env, "foo", "beforeSave")
	ensure.Nil(t, err)
	ensure.True(t, exists)

	exists, err = c.triggerHookExists(h.Env, "bar", "other")
	ensure.Nil(t, err)
	ensure.False(t, exists)
}

func TestDeployTriggerHooks(t *testing.T) {
	t.Parallel()
	h := newTriggersHarness(t)
	c := &Hooks{}

	err := c.deployTriggerHook(
		h.Env,
		&hookOperation{
			Method: "post",
			Trigger: &triggerHook{
				ClassName:   "foo",
				TriggerName: "beforeSave",
				URL:         "https://api.example.com/foo",
			},
		},
	)
	// not an error -> will be converted to put
	ensure.Nil(t, err)
}

func TestParseBaseURL(t *testing.T) {
	t.Parallel()

	h := parsecli.NewHarness(t)
	defer h.Stop()

	c := &Hooks{}
	c.BaseURL = "http://hello"
	ensure.Err(t, c.parseBaseURL(h.Env), regexp.MustCompile("valid https url"))

	c.BaseURL = "https://hello"
	ensure.Nil(t, c.parseBaseURL(h.Env))
	ensure.DeepEqual(t, c.baseWebhookURL.String(), c.BaseURL)
}
