package main

import (
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/facebookgo/ensure"
)

func TestConfigureAccountKey(t *testing.T) {
	t.Parallel()

	h := newTokenHarness(t)
	defer h.Stop()

	c := configureCmd{login: login{tokenReader: strings.NewReader("")}}

	h.env.In = ioutil.NopCloser(strings.NewReader("token\n"))
	ensure.Nil(t, c.accountKey(h.env))
	ensure.StringContains(
		t,
		h.Out.String(),
		`
Input your account key or press enter to generate a new one.
`)

	h.env.In = ioutil.NopCloser(strings.NewReader("invalid\n"))
	ensure.Err(t, c.accountKey(h.env), regexp.MustCompile("is not valid"))
	ensure.DeepEqual(t,
		h.Err.String(),
		"Could not store credentials. Please try again.\n\n",
	)

	h.env.Server = "http://api.parse.com/1/"
	c.tokenReader = strings.NewReader(
		`machine api.parse.com#email
			login default
			password token2
		`,
	)
	h.Err.Reset()
	h.env.In = ioutil.NopCloser(strings.NewReader("token\n"))
	ensure.Nil(t, c.accountKey(h.env))
	ensure.DeepEqual(t, h.Err.String(),
		`Note: this operation will overwrite the account key:
 "*oken"
for email: "email"
`)

	h.env.Server = "http://api.parse.com/1/"
	c.tokenReader = strings.NewReader(
		`machine api.parse.com#email
			login default
			password token2
		`,
	)
	c.isDefault = true
	h.Err.Reset()
	h.env.In = ioutil.NopCloser(strings.NewReader("token\n"))
	ensure.Nil(t, c.accountKey(h.env))
	ensure.DeepEqual(t, h.Err.String(), "")

	h.env.Server = "http://api.parse.com/1/"
	c.tokenReader = strings.NewReader(
		`machine api.parse.com
			login default
			password token2
		`,
	)
	c.isDefault = true
	h.Err.Reset()
	h.env.In = ioutil.NopCloser(strings.NewReader("token\n"))
	ensure.Nil(t, c.accountKey(h.env))
	ensure.DeepEqual(t, h.Err.String(), "Note: this operation will overwrite the default account key\n")

	h.env.Server = "http://api.parse.com/1/"
	c.tokenReader = strings.NewReader(
		`machine api.parse.com
			login default
			password token2
		`,
	)
	h.Err.Reset()
	c.isDefault = false
	h.env.In = ioutil.NopCloser(strings.NewReader("token\n"))
	ensure.Nil(t, c.accountKey(h.env))
	ensure.DeepEqual(t, h.Err.String(), "")
}

func TestParserEmail(t *testing.T) {
	t.Parallel()

	h := newTokenHarness(t)
	h.makeEmptyRoot()
	defer h.Stop()

	var n newCmd
	ensure.Nil(t, n.createConfigWithContent(filepath.Join(h.env.Root, parseLocal), "{}"))
	ensure.Nil(t,
		n.createConfigWithContent(
			filepath.Join(h.env.Root, parseProject),
			`{"project_type": 1}`,
		),
	)

	var c configureCmd
	ensure.Nil(t, c.parserEmail(h.env, []string{"email2"}))
	ensure.DeepEqual(
		t,
		h.Out.String(),
		`Successfully configured email for current project to: "email2"
`,
	)

	ensure.Err(t, c.parserEmail(h.env, nil), regexp.MustCompile("Invalid args:"))
	ensure.Err(t, c.parserEmail(h.env, []string{"a", "b"}), regexp.MustCompile("Invalid args:"))
}

func TestProjectType(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	defer h.Stop()

	h.makeEmptyRoot()
	ensure.Nil(t, (&newCmd{}).cloneSampleCloudCode(h.env, false))

	c := &configureCmd{}
	err := c.projectType(h.env, []string{"1", "2"})
	ensure.Err(t, err, regexp.MustCompile("only an optional project type argument is expected"))

	h.env.In = ioutil.NopCloser(strings.NewReader("invalid\n"))
	err = c.projectType(h.env, nil)
	ensure.StringContains(t, h.Err.String(), "Invalid selection. Please enter a number")
	ensure.Err(t, err, regexp.MustCompile("Could not make a selection. Please try again."))
	h.Err.Reset()
	h.Out.Reset()

	h.env.In = ioutil.NopCloser(strings.NewReader("0\n"))
	err = c.projectType(h.env, nil)
	ensure.StringContains(t, h.Err.String(), "Please enter a number between 1 and")
	ensure.Err(t, err, regexp.MustCompile("Could not make a selection. Please try again."))
	h.Err.Reset()
	h.Out.Reset()

	h.env.In = ioutil.NopCloser(strings.NewReader("1\n"))
	err = c.projectType(h.env, nil)
	ensure.StringContains(t, h.Out.String(), "Successfully set project type to: parse")
	ensure.Nil(t, err)
}

func TestCheckTriggerName(t *testing.T) {
	t.Parallel()

	c := &configureCmd{}

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

func TestPostOrPutHook(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	defer h.Stop()

	c := &configureCmd{}

	_, _, err := c.postOrPutHook(h.env, nil, "other")
	ensure.Err(t, err, regexp.MustCompile("invalid format"))

	_, _, err = c.postOrPutHook(h.env, nil, "post", "1")
	ensure.Err(t, err, regexp.MustCompile("invalid format"))

	_, _, err = c.postOrPutHook(h.env, nil, "put", "1", "2", "3", "4")
	ensure.Err(t, err, regexp.MustCompile("invalid format"))

	var hooksOps []*hookOperation
	_, ops, err := c.postOrPutHook(h.env, hooksOps,
		"post", "call", "https://api.twilio.com/call")
	ensure.Nil(t, err)
	ensure.DeepEqual(t, ops[len(ops)-1].method, "POST")
	ensure.DeepEqual(
		t,
		*ops[len(ops)-1].function,
		functionHook{
			FunctionName: "call",
			URL:          "https://api.twilio.com/call",
		},
	)

	_, ops, err = c.postOrPutHook(h.env, hooksOps,
		"put", "call", "https://api.twilio.com/call_1")
	ensure.Nil(t, err)
	ensure.DeepEqual(t, ops[len(ops)-1].method, "PUT")
	ensure.DeepEqual(
		t,
		*ops[len(ops)-1].function,
		functionHook{
			FunctionName: "call",
			URL:          "https://api.twilio.com/call_1",
		},
	)

	_, ops, err = c.postOrPutHook(h.env, hooksOps, "post",
		"_User", "beforeSave", "https://api.twilio.com/message")
	ensure.Nil(t, err)
	ensure.DeepEqual(t, ops[len(ops)-1].method, "POST")
	ensure.DeepEqual(
		t,
		*ops[len(ops)-1].trigger,
		triggerHook{
			ClassName:   "_User",
			TriggerName: "beforeSave",
			URL:         "https://api.twilio.com/message",
		},
	)

	_, ops, err = c.postOrPutHook(h.env, hooksOps, "put",
		"_User", "beforeSave", "https://api.twilio.com/message_1")
	ensure.Nil(t, err)
	ensure.DeepEqual(t, ops[len(ops)-1].method, "PUT")
	ensure.DeepEqual(
		t,
		*ops[len(ops)-1].trigger,
		triggerHook{
			ClassName:   "_User",
			TriggerName: "beforeSave",
			URL:         "https://api.twilio.com/message_1",
		},
	)

	_, ops, err = c.postOrPutHook(h.env, hooksOps, "post",
		"_User", "invalid", "https://other")
	ensure.Err(t, err, regexp.MustCompile("invalid trigger"))
}

func TestDeleteHook(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	defer h.Stop()

	c := &configureCmd{}
	_, _, err := c.deleteHook(h.env, nil, "delete")
	ensure.Err(t, err, regexp.MustCompile("invalid format"))

	_, _, err = c.deleteHook(h.env, nil, "invalid", "1")
	ensure.Err(t, err, regexp.MustCompile("invalid format"))

	_, _, err = c.deleteHook(h.env, nil, "delete", "1", "2")
	ensure.Err(t, err, regexp.MustCompile("invalid trigger name"))

	var hooksOps []*hookOperation
	_, ops, err := c.deleteHook(h.env, hooksOps, "delete", "call")
	ensure.Nil(t, err)
	ensure.DeepEqual(t, ops[len(ops)-1].method, "DELETE")
	ensure.DeepEqual(
		t,
		*ops[len(ops)-1].function,
		functionHook{FunctionName: "call"},
	)

	_, ops, err = c.deleteHook(h.env, hooksOps, "delete", "_User", "beforeSave")
	ensure.Nil(t, err)
	ensure.DeepEqual(t, ops[len(ops)-1].method, "DELETE")
	ensure.DeepEqual(
		t,
		*ops[len(ops)-1].trigger,
		triggerHook{ClassName: "_User", TriggerName: "beforeSave"},
	)
}

func TestProcessHookOperation(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	defer h.Stop()

	c := &configureCmd{}

	ops, err := c.processHooksOperation(h.env, "\n\t ")
	ensure.Nil(t, err)
	ensure.DeepEqual(t, len(ops), 0)

	_, err = c.processHooksOperation(h.env, "invalid")
	ensure.Err(t, err, regexp.MustCompile("invalid format"))

	_, err = c.processHooksOperation(h.env, "delete,call,caller")
	ensure.Err(t, err, regexp.MustCompile("invalid format"))

	ops, err = c.processHooksOperation(h.env, "delete,call")
	ensure.Nil(t, err)
	ensure.DeepEqual(t, ops, []string{"delete", "call"})

	ops, err = c.processHooksOperation(h.env, "delete,_User:beforeSave")
	ensure.Nil(t, err)
	ensure.DeepEqual(t, ops, []string{"delete", "_User", "beforeSave"})

	ops, err = c.processHooksOperation(h.env, "post,call,https://twilio.com/call")
	ensure.Nil(t, err)
	ensure.DeepEqual(t, ops, []string{"post", "call", "https://twilio.com/call"})

	ops, err = c.processHooksOperation(h.env, "put,call,https://twilio.com/call_1,call_2")
	ensure.Nil(t, err)
	ensure.DeepEqual(t, ops, []string{"put", "call", "https://twilio.com/call_1,call_2"})

	ops, err = c.processHooksOperation(h.env, "put,call,https://twilio.com/call_1,call_2,call_3")
	ensure.Nil(t, err)
	ensure.DeepEqual(t, ops, []string{"put", "call", "https://twilio.com/call_1,call_2,call_3"})

	ops, err = c.processHooksOperation(h.env,
		"pUt,_User:afterDelete,https://twilio.com/message_1,message_2")
	ensure.Nil(t, err)
	ensure.DeepEqual(t, ops, []string{"pUt", "_User", "afterDelete",
		"https://twilio.com/message_1,message_2"})

	u, err := url.Parse("https://parse.com")
	ensure.Nil(t, err)
	c.baseWebhookURL = u

	ops, err = c.processHooksOperation(h.env, "post,call,https://parse.com/call")
	ensure.Nil(t, err)
	ensure.DeepEqual(t, ops, []string{"post", "call", "https://parse.com/call"})

	ops, err = c.processHooksOperation(h.env, "post,call,https://twilio.com/call")
	ensure.Nil(t, err)
	ensure.DeepEqual(t, ops, []string{"post", "call", "https://twilio.com/call"})

	ops, err = c.processHooksOperation(h.env, "put,call,/call_1,call_2,call_3")
	ensure.Nil(t, err)
	ensure.DeepEqual(t, ops, []string{"put", "call", "https://parse.com/call_1,call_2,call_3"})

	ops, err = c.processHooksOperation(h.env, "put,call,https://twilio.com/call_1,call_2,call_3")
	ensure.Nil(t, err)
	ensure.DeepEqual(t, ops, []string{"put", "call", "https://twilio.com/call_1,call_2,call_3"})
}

func TestAppendHookOperation(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	defer h.Stop()

	c := &configureCmd{}

	var hooksOps []*hookOperation
	_, ops, err := c.appendHookOperation(h.env, nil, hooksOps)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, hooksOps, ops)

	_, _, err = c.appendHookOperation(h.env, []string{"invalid"}, nil)
	ensure.Err(t, err, regexp.MustCompile("invalid format"))

	_, ops, err = c.appendHookOperation(h.env,
		[]string{"post", "call", "https://twilio.com/call"}, hooksOps)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, ops[len(ops)-1].method, "POST")
	ensure.DeepEqual(
		t,
		*ops[len(ops)-1].function,
		functionHook{FunctionName: "call", URL: "https://twilio.com/call"},
	)

	_, ops, err = c.appendHookOperation(h.env,
		[]string{"put", "call", "https://twilio.com/call_1"}, hooksOps)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, ops[len(ops)-1].method, "PUT")
	ensure.DeepEqual(
		t,
		*ops[len(ops)-1].function,
		functionHook{FunctionName: "call", URL: "https://twilio.com/call_1"},
	)

	//random stuff
	_, ops, err = c.appendHookOperation(h.env,
		[]string{"posT", "_User", "afterDelete", "https://twilio.com/message"}, hooksOps)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, ops[len(ops)-1].method, "POST")
	ensure.DeepEqual(
		t,
		*ops[len(ops)-1].trigger,
		triggerHook{ClassName: "_User", TriggerName: "afterDelete", URL: "https://twilio.com/message"},
	)

	_, ops, err = c.appendHookOperation(h.env,
		[]string{"pUt", "_User", "afterDelete", "https://twilio.com/message_1"}, hooksOps)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, ops[len(ops)-1].method, "PUT")
	ensure.DeepEqual(
		t,
		*ops[len(ops)-1].trigger,
		triggerHook{ClassName: "_User", TriggerName: "afterDelete", URL: "https://twilio.com/message_1"},
	)

	// random stuff
	_, ops, err = c.appendHookOperation(h.env,
		[]string{"pUt", "_User", "afterDelete", "https://twilio.com/message_1,message_2"}, hooksOps)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, ops[len(ops)-1].method, "PUT")
	ensure.DeepEqual(
		t,
		*ops[len(ops)-1].trigger,
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

	c := &configureCmd{}
	for _, testCase := range testCases {
		c.hooksStrict = testCase.strictMode
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
	h := newHarness(t)
	h.makeEmptyRoot()
	defer h.Stop()

	hooksConfigFile := filepath.Join(h.env.Root, "webhooks.csv")
	err := ioutil.WriteFile(
		hooksConfigFile,
		[]byte{},
		0666,
	)
	ensure.Nil(t, err)

	c := &configureCmd{}

	f, err := os.Open(hooksConfigFile)
	ensure.Nil(t, err)
	ops, err := c.createHooksOperations(h.env, f)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, len(ops), 0)
	ensure.Nil(t, f.Close())

	err = ioutil.WriteFile(
		hooksConfigFile,
		[]byte("\n"),
		0666,
	)

	f, err = os.Open(hooksConfigFile)
	ensure.Nil(t, err)
	ops, err = c.createHooksOperations(h.env, f)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, len(ops), 0)
	ensure.DeepEqual(t, h.Out.String(), "Ignoring line: 1\n")
	ensure.Nil(t, f.Close())
	h.Out.Reset()
}

func TestFunctionHookExists(t *testing.T) {
	t.Parallel()
	h := newFunctionsHarness(t)
	c := &configureCmd{}
	exists, err := c.functionHookExists(h.env, "foo")
	ensure.Nil(t, err)
	ensure.True(t, exists)

	exists, err = c.functionHookExists(h.env, "bar")
	ensure.Nil(t, err)
	ensure.False(t, exists)
}

func TestDeployFunctionHooks(t *testing.T) {
	t.Parallel()
	h := newFunctionsHarness(t)
	c := &configureCmd{}

	err := c.deployFunctionHook(
		h.env,
		&hookOperation{
			method: "post",
			function: &functionHook{
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
	c := &configureCmd{}
	exists, err := c.triggerHookExists(h.env, "foo", "beforeSave")
	ensure.Nil(t, err)
	ensure.True(t, exists)

	exists, err = c.triggerHookExists(h.env, "bar", "other")
	ensure.Nil(t, err)
	ensure.False(t, exists)
}

func TestDeployTriggerHooks(t *testing.T) {
	t.Parallel()
	h := newTriggersHarness(t)
	c := &configureCmd{}

	err := c.deployTriggerHook(
		h.env,
		&hookOperation{
			method: "post",
			trigger: &triggerHook{
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

	h := newHarness(t)
	defer h.Stop()

	c := &configureCmd{}
	c.baseURL = "http://hello"
	ensure.Err(t, c.parseBaseURL(h.env), regexp.MustCompile("valid https url"))

	c.baseURL = "https://hello"
	ensure.Nil(t, c.parseBaseURL(h.env))
	ensure.DeepEqual(t, c.baseWebhookURL.String(), c.baseURL)
}
