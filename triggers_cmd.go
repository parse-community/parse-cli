package main

import (
	"errors"
	"fmt"
	"net/url"
	"path"
	"sort"
	"strings"

	"github.com/facebookgo/stackerr"
	"github.com/spf13/cobra"
)

type triggerHook struct {
	ClassName   string `json:"className,omitempty"`
	TriggerName string `json:"triggerName,omitempty"`
	URL         string `json:"url,omitempty"`
	Warning     string `json:"warning,omitempty"`
}

func (t *triggerHook) String() string {
	if t.URL != "" {
		return fmt.Sprintf("Class name: %q, Trigger name: %q, URL: %q", t.ClassName, t.TriggerName, t.URL)
	}
	return fmt.Sprintf("Class name: %q, Trigger name: %q", t.ClassName, t.TriggerName)
}

type triggerHooksCmd struct {
	All         bool
	Trigger     *triggerHook
	interactive bool
}

func readTriggerName(e *env, params *triggerHook) (*triggerHook, error) {
	if params != nil && params.ClassName != "" && params.TriggerName != "" {
		return params, nil
	}

	var t triggerHook
	fmt.Fprintln(e.Out, "Please enter following details about the trigger webhook")
	fmt.Fprint(e.Out, "Class name: ")
	fmt.Fscanf(e.In, "%s\n", &t.ClassName)
	if t.ClassName == "" {
		return nil, errors.New("Class name cannot be empty")
	}
	fmt.Fprint(e.Out, "Trigger name: ")
	fmt.Fscanf(e.In, "%s\n", &t.TriggerName)
	if t.TriggerName == "" {
		return nil, errors.New("Trigger name cannot be empty")
	}
	return &t, nil
}

func readTriggerParams(e *env, params *triggerHook) (*triggerHook, error) {
	if params != nil && params.ClassName != "" && params.TriggerName != "" && params.URL != "" {
		return params, nil
	}

	t, err := readTriggerName(e, params)
	if err != nil {
		return nil, err
	}
	fmt.Fprint(e.Out, "URL: https://")
	fmt.Fscanf(e.In, "%s\n", &t.URL)
	t.URL = "https://" + t.URL
	if err := validateURL(t.URL); err != nil {
		return nil, err
	}
	return t, nil
}

const defaultTriggersURL = "/1/hooks/triggers"

func (h *triggerHooksCmd) triggerHooksCreate(e *env, ctx *context) error {
	params, err := readTriggerParams(e, h.Trigger)
	if err != nil {
		return err
	}
	var res triggerHook
	triggersURL, err := url.Parse(defaultTriggersURL)
	if err != nil {
		return stackerr.Wrap(err)
	}
	_, err = e.ParseAPIClient.Post(triggersURL, params, &res)
	if err != nil {
		return stackerr.Wrap(err)
	}
	if res.Warning != "" {
		fmt.Fprintf(e.Err, "WARNING: %s\n", res.Warning)
	}
	fmt.Fprintf(e.Out,
		"Successfully created a %q trigger for class %q pointing to %q\n",
		res.TriggerName,
		res.ClassName,
		res.URL,
	)
	return nil
}

func (h *triggerHooksCmd) triggerHooksRead(e *env, ctx *context) error {
	u := defaultTriggersURL
	var trigger *triggerHook
	if !h.All {
		trig, err := readTriggerName(e, h.Trigger)
		if err != nil {
			return err
		}
		trigger = trig
		u = path.Join(u, trigger.ClassName, trigger.TriggerName)
	}
	triggersURL, err := url.Parse(u)
	if err != nil {
		return stackerr.Wrap(err)
	}
	var res struct {
		Results []*triggerHook `json:"results,omitempty"`
	}
	_, err = e.ParseAPIClient.Get(triggersURL, &res)
	if err != nil {
		return stackerr.Wrap(err)
	}
	var output []string
	for _, trigger := range res.Results {
		output = append(output, trigger.String())
	}
	sort.Strings(output)

	if h.All {
		fmt.Fprintln(e.Out, "The following cloudcode or webhook triggers are associated with this app:")
	} else {
		if len(output) == 1 {
			fmt.Fprintf(e.Out, "You have one trigger named: %q for class: %q\n", trigger.TriggerName, trigger.ClassName)
		} else {
			fmt.Fprintf(e.Out, "The following triggers named: %q are associated with the class: %q\n", trigger.TriggerName, trigger.ClassName)
		}
	}
	fmt.Fprintln(e.Out, strings.Join(output, "\n"))

	return nil
}

func (h *triggerHooksCmd) triggerHooksUpdate(e *env, ctx *context) error {
	params, err := readTriggerParams(e, h.Trigger)
	if err != nil {
		return err
	}
	var res triggerHook
	triggersURL, err := url.Parse(path.Join(defaultTriggersURL, params.ClassName, params.TriggerName))
	if err != nil {
		return stackerr.Wrap(err)
	}
	_, err = e.ParseAPIClient.Put(triggersURL, &triggerHook{URL: params.URL}, &res)
	if err != nil {
		return stackerr.Wrap(err)
	}
	if res.Warning != "" {
		fmt.Fprintf(e.Err, "WARNING: %s\n", res.Warning)
	}
	fmt.Fprintf(e.Out,
		"Successfully update the %q trigger for class %q to point to %q\n",
		res.TriggerName,
		res.ClassName,
		res.URL,
	)

	return nil
}

func (h *triggerHooksCmd) triggerHooksDelete(e *env, ctx *context) error {
	params, err := readTriggerName(e, h.Trigger)
	if err != nil {
		return err
	}
	triggersURL, err := url.Parse(path.Join(defaultTriggersURL, params.ClassName, params.TriggerName))
	if err != nil {
		return stackerr.Wrap(err)
	}

	confirmMessage := fmt.Sprintf("Are you sure you want to delete %q webhook trigger for class: %q (y/n): ",
		params.TriggerName,
		params.ClassName,
	)

	var res triggerHook
	if !h.interactive || getConfirmation(confirmMessage, e) {
		_, err = e.ParseAPIClient.Put(triggersURL, map[string]interface{}{"__op": "Delete"}, &res)
		if err != nil {
			return stackerr.Wrap(err)
		}
		fmt.Fprintf(e.Out, "Successfully deleted %q webhook trigger for class %q\n",
			params.TriggerName,
			params.ClassName,
		)
		if res.TriggerName != "" && res.ClassName != "" {
			fmt.Fprintf(e.Out, "%q trigger defined in cloudcode for class %q will be used henceforth\n",
				res.TriggerName,
				res.ClassName,
			)
		}
	}
	return nil
}

func (h *triggerHooksCmd) triggerHooks(e *env, c *context) error {
	hp := *h
	hp.All = true
	return hp.triggerHooksRead(e, c)
}

func newTriggerHooksCmd(e *env) *cobra.Command {
	h := &triggerHooksCmd{interactive: true}

	c := &cobra.Command{
		Use:   "triggers",
		Short: "List cloud code triggers and trigger webhooks",
		Long:  "List cloud code triggers and trigger webhooks",
		Run:   runWithClient(e, h.triggerHooks),
	}

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a trigger webhook",
		Long:  "Create a trigger webhook",
		Run:   runWithClient(e, h.triggerHooksCreate),
	}
	c.AddCommand(createCmd)

	changeCmd := &cobra.Command{
		Use:   "edit",
		Short: "Edit the URL of a trigger webhook",
		Long:  "Edit the URL of a trigger webhook",
		Run:   runWithClient(e, h.triggerHooksUpdate),
	}
	c.AddCommand(changeCmd)

	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a trigger webhook",
		Long:  "Delete a trigger webhook",
		Run:   runWithClient(e, h.triggerHooksDelete),
	}
	c.AddCommand(deleteCmd)

	return c
}
