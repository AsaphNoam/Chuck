package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/spf13/cobra"

	"github.com/AsaphNoam/Chuck/internal/state"
)

// newOperatorWorkflowCommands contains the non-agent operator families whose
// request shapes are intentionally passed through unchanged by --request-file.
func newOperatorWorkflowCommands() []*cobra.Command {
	commands := []*cobra.Command{newTaskCmd(), newSignalCmd(), newThinkTankCmd()}
	for _, cmd := range commands {
		documentWorkflowCommands(cmd)
	}
	return commands
}

func documentWorkflowCommands(family *cobra.Command) {
	type help struct{ short, long, example string }
	entries := map[string]help{
		"task list":                         {"List project tasks", "List dependent work for one project. The project is required by the existing tasks API.", "chuck task list --project my-project --json"},
		"task show":                         {"Show one task", "Read the authoritative task detail by task id.", "chuck task show task_123 --json"},
		"task create":                       {"Create dependent work", "Create a project task with an explicit target and instruction. Use --request-file for prerequisites, attachments or other fields in the task request shape.", "chuck task create --project my-project --display-name 'Review cache' --target-kind agent --target-agent agent_123 --text 'Compare eviction options'"},
		"task cancel":                       {"Cancel a task", "Cancel task work through the task owner route.", "chuck task cancel task_123 --json"},
		"task result":                       {"Record a task result", "Record a person-authored outcome and summary for a task that cannot report for itself.", "chuck task result task_123 --outcome failure --text 'The service is unavailable'"},
		"task retry":                        {"Retry a task", "Retry the task through its existing assignee and owner checks.", "chuck task retry task_123 --json"},
		"task rearm":                        {"Replace task prerequisites", "Replace the task's prerequisite arms using the existing rearm request shape.", "chuck task rearm task_123 --request-file rearm.json"},
		"task delete":                       {"Delete a task", "Delete a task and explicitly acknowledge the deletion with --yes.", "chuck task delete task_123 --yes"},
		"signal fire":                       {"Fire a project signal", "Fire one bounded signal name in the named project.", "chuck signal fire --project my-project --name build-ready"},
		"think-tank list":                   {"List Think Tanks", "List rooms using the existing room collection endpoint.", "chuck think-tank list --json"},
		"think-tank show":                   {"Show a Think Tank", "Read one room's authoritative detail by room id.", "chuck think-tank show tt_123 --json"},
		"think-tank create":                 {"Create a Think Tank", "Create a room with its complete participant setup in --request-file; --command-id supports safe identical retry.", "chuck think-tank create --command-id request-1 --project my-project --goal 'Choose a cache' --request-file room.json"},
		"think-tank entries":                {"Read room entries", "Read published entries, optionally starting after an entry sequence with --after.", "chuck think-tank entries tt_123 --after 20 --json"},
		"think-tank activity":               {"Read room activity", "Read room activity, optionally starting after an activity sequence with --after.", "chuck think-tank activity tt_123 --after 20 --json"},
		"think-tank files":                  {"List room files", "List retained source files available from the room.", "chuck think-tank files tt_123 --json"},
		"think-tank commands":               {"List room commands", "Inspect provider commands recorded for room participants; this does not execute them.", "chuck think-tank commands tt_123 --json"},
		"think-tank source-file":            {"Read a retained source file", "Read one retained source file by room and source ids.", "chuck think-tank source-file tt_123 source_123 --json"},
		"think-tank send":                   {"Send a room message", "Send room text and optional repeatable participant mentions as agent_id:start:end UTF-8 byte ranges in the body. Reuse --command-id with identical input after an ambiguous result.", "chuck think-tank send tt_123 --text '@Ari, compare hit rate' --mention agent_123:0:4"},
		"think-tank annotate":               {"Annotate a room", "Submit the existing annotation target and anchor shapes in --request-file. Reuse the same command id and input for an identical retry.", "chuck think-tank annotate tt_123 --request-file annotation.json"},
		"think-tank pause":                  {"Pause a Think Tank", "Pause room progression using its existing control route.", "chuck think-tank pause tt_123 --json"},
		"think-tank resume":                 {"Resume a Think Tank", "Resume room progression using its existing control route.", "chuck think-tank resume tt_123 --json"},
		"think-tank end":                    {"End a Think Tank", "End room progression using its existing control route.", "chuck think-tank end tt_123 --json"},
		"think-tank retry":                  {"Retry a Think Tank operation", "Select setup, turn or judge explicitly with --target. Turn retries can identify an opening attempt with --attempt-id; reuse --command-id for identical retries.", "chuck think-tank retry tt_123 --target turn --attempt-id attempt_123 --command-id request-1"},
		"think-tank participant-turn-limit": {"Change a participant turn limit", "Set an increased turn limit only when --expected-limit matches the current value. A command id makes the request retryable.", "chuck think-tank participant-turn-limit tt_123 agent_123 --expected-limit 2 --limit 5 --command-id request-1"},
		"think-tank delete":                 {"Delete a Think Tank", "Delete room-owned data and explicitly acknowledge the deletion with --yes. Participant agents and their histories remain under their own lifecycle.", "chuck think-tank delete tt_123 --yes"},
	}
	for _, cmd := range family.Commands() {
		key := family.Name() + " " + strings.Fields(cmd.Use)[0]
		h, ok := entries[key]
		if !ok {
			continue
		}
		cmd.Short, cmd.Long, cmd.Example = h.short, h.long, h.example
	}
}

func operatorAction(use, method, path string, body func(*cobra.Command, []string) (any, error), destructive bool) *cobra.Command {
	cmd := &cobra.Command{Use: use, Args: cobra.ArbitraryArgs}
	addOperatorFlags(cmd)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if destructive {
			yes, _ := cmd.Flags().GetBool("yes")
			if !yes {
				return &operatorError{Code: "confirmation_required", Message: "this operation deletes data; pass --yes", ExitCode: 2}
			}
		}
		payload, err := body(cmd, args)
		if err != nil {
			return err
		}
		return operatorRun(cmd, method, path, payload, nil)
	}
	if destructive {
		cmd.Flags().Bool("yes", false, "acknowledge deletion")
	}
	return cmd
}

func operatorRequestBody(cmd *cobra.Command, positional map[string]any) (map[string]any, error) {
	raw, err := operatorInput(cmd)
	if err != nil {
		return nil, err
	}
	body := make(map[string]any, len(raw)+len(positional))
	for k, v := range raw {
		body[k] = v
	}
	for k, v := range positional {
		if v == nil {
			continue
		}
		if _, exists := raw[k]; exists {
			return nil, fmt.Errorf("duplicate request field %q", k)
		}
		body[k] = v
	}
	return body, nil
}

func operatorCommandBody(cmd *cobra.Command, extra map[string]any) (map[string]any, error) {
	if extra == nil {
		extra = map[string]any{}
	} else {
		copy := make(map[string]any, len(extra)+1)
		for key, value := range extra {
			copy[key] = value
		}
		extra = copy
	}
	raw, err := operatorInput(cmd)
	if err != nil {
		return nil, err
	}
	if _, ok := raw["command_id"]; !ok {
		if _, ok := extra["command_id"]; !ok {
			extra["command_id"] = newPipelineRequestID()
		}
	}
	body := make(map[string]any, len(raw)+len(extra))
	for k, v := range raw {
		body[k] = v
	}
	for k, v := range extra {
		if _, exists := raw[k]; exists {
			return nil, fmt.Errorf("duplicate request field %q", k)
		}
		body[k] = v
	}
	return body, nil
}

func idPath(base string, args []string) (string, error) {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		return "", errors.New("an id is required")
	}
	return base + "/" + url.PathEscape(args[0]), nil
}

func newTaskCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "task", Short: "Inspect and control dependent work"}
	var project string
	list := operatorAction("list", http.MethodGet, "/api/tasks", func(c *cobra.Command, _ []string) (any, error) {
		project, _ := c.Flags().GetString("project")
		path := "/api/tasks"
		if project != "" {
			path += "?project=" + url.QueryEscape(project)
		}
		c.Annotations = map[string]string{"operator-path": path}
		return nil, nil
	}, false)
	// GET paths with query parameters are carried by the command annotation so
	// the generic runner remains shared with mutation commands.
	list.RunE = func(c *cobra.Command, _ []string) error {
		if !c.Flags().Changed("project") || strings.TrimSpace(project) == "" {
			return operatorInputError(errors.New("task list requires --project"))
		}
		path := "/api/tasks"
		if project != "" {
			path += "?project=" + url.QueryEscape(project)
		}
		return operatorRun(c, http.MethodGet, path, nil, nil)
	}
	list.Flags().StringVar(&project, "project", "", "project id")
	list.Args = cobra.NoArgs
	cmd.AddCommand(list)
	show := operatorAction("show <task_id>", http.MethodGet, "/api/tasks", func(_ *cobra.Command, args []string) (any, error) { return nil, nil }, false)
	show.RunE = func(c *cobra.Command, args []string) error {
		p, err := idPath("/api/tasks", args)
		if err != nil {
			return err
		}
		return operatorRun(c, http.MethodGet, p, nil, nil)
	}
	show.Args = cobra.ExactArgs(1)
	cmd.AddCommand(show)

	create := workflowBodyCommand("create", http.MethodPost, "/api/tasks", map[string]string{"project": "project", "display_name": "display-name", "instruction": "instruction", "target_kind": "target-kind", "target_agent_id": "target-agent", "role": "role", "backend": "backend", "model": "model", "effort": "effort"})
	create.Flags().Bool("fast", false, "use fast mode")
	addOperatorTextFlags(create)
	create.Args = cobra.NoArgs
	cmd.AddCommand(create)
	for _, op := range []string{"cancel", "result", "retry", "rearm"} {
		op := op
		a := operatorAction(op+" <task_id>", http.MethodPost, "/api/tasks", func(c *cobra.Command, args []string) (any, error) {
			p, err := idPath("/api/tasks", args)
			if err != nil {
				return nil, err
			}
			c.Annotations = map[string]string{"operator-path": p + "/" + op}
			return operatorRequestBody(c, nil)
		}, false)
		a.RunE = func(c *cobra.Command, args []string) error {
			p, err := idPath("/api/tasks", args)
			if err != nil {
				return err
			}
			b, err := operatorRequestBody(c, nil)
			if err != nil {
				return err
			}
			return operatorRun(c, http.MethodPost, p+"/"+op, b, nil)
		}
		a.Args = cobra.ExactArgs(1)
		if op == "result" {
			addOperatorTextFlags(a)
			a.Flags().String("outcome", "", "recorded outcome: success, failure, blocked or cancelled")
			a.Flags().String("details", "", "additional result details")
		}
		if op == "result" {
			a.RunE = func(c *cobra.Command, args []string) error {
				text, ok, e := operatorText(c)
				if e != nil {
					return e
				}
				p, e := idPath("/api/tasks", args)
				if e != nil {
					return e
				}
				extra := map[string]any{}
				if ok {
					extra["summary"] = text
				}
				for _, field := range []string{"outcome", "details"} {
					if c.Flags().Changed(field) {
						value, _ := c.Flags().GetString(field)
						extra[field] = value
					}
				}
				b, e := operatorRequestBody(c, extra)
				if e != nil {
					return e
				}
				if !operatorNonEmptyString(b["summary"]) {
					return operatorInputError(errors.New("task result requires a summary via --text, --text-file or --request-file"))
				}
				if value, exists := b["outcome"]; !exists || !operatorNonEmptyString(value) {
					return operatorInputError(errors.New("task result requires --outcome or outcome in --request-file"))
				}
				return operatorRun(c, http.MethodPost, p+"/result", b, nil)
			}
		}
		cmd.AddCommand(a)
	}
	del := operatorAction("delete <task_id>", http.MethodDelete, "/api/tasks", func(c *cobra.Command, args []string) (any, error) { return nil, nil }, true)
	del.RunE = func(c *cobra.Command, args []string) error {
		p, err := idPath("/api/tasks", args)
		if err != nil {
			return err
		}
		yes, _ := c.Flags().GetBool("yes")
		if !yes {
			return &operatorError{Code: "confirmation_required", Message: "this operation deletes data; pass --yes", ExitCode: 2}
		}
		return operatorRun(c, http.MethodDelete, p, nil, nil)
	}
	del.Args = cobra.ExactArgs(1)
	cmd.AddCommand(del)
	return cmd
}

func workflowBodyCommand(use, method, path string, fields map[string]string) *cobra.Command {
	cmd := operatorAction(use, method, path, func(c *cobra.Command, _ []string) (any, error) {
		vals, err := workflowRequestValues(c, fields)
		if err != nil {
			return nil, err
		}
		var body map[string]any
		if _, ok := fields["command_id"]; ok {
			body, err = operatorCommandBody(c, vals)
		} else {
			body, err = operatorRequestBody(c, vals)
		}
		if err != nil {
			return nil, err
		}
		if _, ok := fields["instruction"]; ok {
			text, exists := operatorStringValue(body["instruction"])
			if !exists || strings.TrimSpace(text) == "" {
				return nil, operatorInputError(errors.New("task create requires instruction via --instruction, --text, --text-file or --request-file"))
			}
			if utf8.RuneCountInString(text) > 16000 {
				return nil, operatorInputError(errors.New("task instruction exceeds 16000 characters"))
			}
			name, exists := operatorStringValue(body["display_name"])
			if !exists || strings.TrimSpace(name) == "" || utf8.RuneCountInString(name) > 200 {
				return nil, operatorInputError(errors.New("task display name is required and limited to 200 characters"))
			}
			target, _ := operatorStringValue(body["target_kind"])
			switch target {
			case "agent":
				if id, ok := operatorStringValue(body["target_agent_id"]); !ok || strings.TrimSpace(id) == "" {
					return nil, operatorInputError(errors.New("agent task target requires --target-agent or target_agent_id in --request-file"))
				}
			case "launch":
				if role, ok := operatorStringValue(body["role"]); !ok || strings.TrimSpace(role) == "" {
					return nil, operatorInputError(errors.New("launch task target requires --role or role in --request-file"))
				}
			}
		}
		if _, ok := fields["name"]; ok {
			name, exists := operatorStringValue(body["name"])
			if !exists || strings.TrimSpace(name) == "" || utf8.RuneCountInString(name) > 200 {
				return nil, operatorInputError(errors.New("signal fire requires a name of at most 200 characters"))
			}
		}
		for _, key := range []string{"project", "display_name", "target_kind", "origin_project", "goal"} {
			if _, required := fields[key]; !required {
				continue
			}
			text, exists := operatorStringValue(body[key])
			if !exists || strings.TrimSpace(text) == "" {
				return nil, operatorInputError(fmt.Errorf("%s is required via its flag or --request-file", key))
			}
		}
		if _, ok := fields["title"]; ok {
			if raw, exists := body["participants"]; !exists || !operatorNonEmptyArray(raw) {
				return nil, operatorInputError(errors.New("think-tank create requires participants in --request-file"))
			}
		}
		return body, nil
	}, false)
	for _, flag := range fields {
		cmd.Flags().String(flag, "", flag)
	}
	return cmd
}

func workflowRequestValues(c *cobra.Command, fields map[string]string) (map[string]any, error) {
	vals := map[string]any{}
	for key, flag := range fields {
		if c.Flags().Changed(flag) {
			v, _ := c.Flags().GetString(flag)
			vals[key] = v
		}
	}
	if key := textField(fields); key != "" && (c.Flags().Changed("text") || c.Flags().Changed("text-file")) {
		v, supplied, err := operatorText(c)
		if err != nil {
			return nil, err
		}
		if supplied {
			if c.Flags().Changed(fields[key]) {
				return nil, operatorInputError(fmt.Errorf("use either --%s or --text/--text-file", fields[key]))
			}
			vals[key] = v
		}
	}
	if c.Flags().Changed("fast") {
		v, _ := c.Flags().GetBool("fast")
		vals["fast"] = v
	}
	if c.Flags().Changed("openings") {
		v, _ := c.Flags().GetBool("openings")
		vals["openings"] = v
	}
	return vals, nil
}

func textField(fields map[string]string) string {
	if _, ok := fields["instruction"]; ok {
		return "instruction"
	}
	if _, ok := fields["body"]; ok {
		return "body"
	}
	return ""
}

func newSignalCmd() *cobra.Command {
	fire := workflowBodyCommand("fire", http.MethodPost, "/api/signals", map[string]string{"project": "project", "name": "name"})
	fire.Short = "Fire a project signal"
	fire.Args = cobra.NoArgs
	cmd := &cobra.Command{Use: "signal", Short: "Fire project signals"}
	cmd.AddCommand(fire)
	return cmd
}

func newThinkTankCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "think-tank", Short: "Inspect and control Think Tanks"}
	list := operatorAction("list", http.MethodGet, "/api/think-tanks", func(_ *cobra.Command, _ []string) (any, error) { return nil, nil }, false)
	cmd.AddCommand(list)
	show := operatorAction("show <room_id>", http.MethodGet, "/api/think-tanks", nil, false)
	show.RunE = func(c *cobra.Command, args []string) error {
		p, e := idPath("/api/think-tanks", args)
		if e != nil {
			return e
		}
		return operatorRun(c, http.MethodGet, p, nil, nil)
	}
	show.Args = cobra.ExactArgs(1)
	cmd.AddCommand(show)
	create := workflowBodyCommand("create", http.MethodPost, "/api/think-tanks", map[string]string{"command_id": "command-id", "title": "title", "goal": "goal", "origin_project": "project"})
	create.Flags().Bool("openings", false, "start openings")
	create.Args = cobra.NoArgs
	cmd.AddCommand(create)
	for _, op := range []string{"entries", "activity", "files", "commands"} {
		op := op
		a := operatorAction(op+" <room_id>", http.MethodGet, "/api/think-tanks", nil, false)
		a.RunE = func(c *cobra.Command, args []string) error {
			p, e := idPath("/api/think-tanks", args)
			if e != nil {
				return e
			}
			return operatorRun(c, http.MethodGet, p+"/"+op, nil, nil)
		}
		a.Args = cobra.ExactArgs(1)
		if op == "entries" || op == "activity" {
			a.Flags().Int64("after", 0, "return records after sequence")
			a.RunE = func(c *cobra.Command, args []string) error {
				after, _ := c.Flags().GetInt64("after")
				p, e := idPath("/api/think-tanks", args)
				if e != nil {
					return e
				}
				path := p + "/" + op
				if c.Flags().Changed("after") {
					path += "?after=" + strconv.FormatInt(after, 10)
				}
				return operatorRun(c, http.MethodGet, path, nil, nil)
			}
		}
		cmd.AddCommand(a)
	}
	source := operatorAction("source-file <room_id> <source_id>", http.MethodGet, "/api/think-tanks", nil, false)
	source.RunE = func(c *cobra.Command, args []string) error {
		if len(args) != 2 {
			return errors.New("room_id and source_id are required")
		}
		return operatorRun(c, http.MethodGet, "/api/think-tanks/"+url.PathEscape(args[0])+"/sources/"+url.PathEscape(args[1])+"/file", nil, nil)
	}
	source.Args = cobra.ExactArgs(2)
	cmd.AddCommand(source)
	for _, op := range []string{"pause", "resume", "end"} {
		op := op
		a := operatorAction(op+" <room_id>", http.MethodPost, "/api/think-tanks", func(c *cobra.Command, _ []string) (any, error) { return operatorRequestBody(c, nil) }, false)
		a.RunE = func(c *cobra.Command, args []string) error {
			p, e := idPath("/api/think-tanks", args)
			if e != nil {
				return e
			}
			b, e := operatorRequestBody(c, nil)
			if e != nil {
				return e
			}
			return operatorRun(c, http.MethodPost, p+"/"+op, b, nil)
		}
		a.Args = cobra.ExactArgs(1)
		cmd.AddCommand(a)
	}
	msg := workflowBodyCommand("send <room_id>", http.MethodPost, "/api/think-tanks", map[string]string{"command_id": "command-id", "body": "body"})
	msg.Flags().StringSlice("mention", nil, "participant mention")
	addOperatorTextFlags(msg)
	msg.RunE = func(c *cobra.Command, args []string) error {
		p, e := idPath("/api/think-tanks", args)
		if e != nil {
			return e
		}
		extra, e := workflowRequestValues(c, map[string]string{"command_id": "command-id", "body": "body"})
		if e != nil {
			return e
		}
		if c.Flags().Changed("mention") {
			mentions, _ := c.Flags().GetStringSlice("mention")
			items := make([]map[string]any, 0, len(mentions))
			for _, mention := range mentions {
				parts := strings.Split(mention, ":")
				if len(parts) != 3 || strings.TrimSpace(parts[0]) == "" {
					return operatorInputError(errors.New("--mention must be agent_id:start:end using UTF-8 byte offsets"))
				}
				start, startErr := strconv.Atoi(parts[1])
				end, endErr := strconv.Atoi(parts[2])
				if startErr != nil || endErr != nil || start < 0 || end <= start {
					return operatorInputError(errors.New("--mention requires a nonnegative start and a greater end byte offset"))
				}
				items = append(items, map[string]any{"agent_id": parts[0], "start": start, "end": end})
			}
			extra["mentions"] = items
		}
		b, e := operatorCommandBody(c, extra)
		if e != nil {
			return e
		}
		if !operatorNonEmptyString(b["body"]) {
			return operatorInputError(errors.New("think-tank send requires a body via --body, --text, --text-file or --request-file"))
		}
		body, _ := operatorStringValue(b["body"])
		if len([]byte(body)) > state.ThinkTankMaxTextBytes {
			return operatorInputError(fmt.Errorf("think-tank message exceeds %d bytes", state.ThinkTankMaxTextBytes))
		}
		return operatorRun(c, http.MethodPost, p+"/messages", b, nil)
	}
	msg.Args = cobra.ExactArgs(1)
	cmd.AddCommand(msg)
	annotate := operatorAction("annotate <room_id>", http.MethodPost, "/api/think-tanks", func(c *cobra.Command, _ []string) (any, error) {
		values, err := workflowRequestValues(c, map[string]string{"command_id": "command-id"})
		if err != nil {
			return nil, err
		}
		return operatorCommandBody(c, values)
	}, false)
	annotate.Flags().String("command-id", "", "stable annotation request id; reuse with identical input after an ambiguous result")
	annotate.RunE = func(c *cobra.Command, args []string) error {
		p, e := idPath("/api/think-tanks", args)
		if e != nil {
			return e
		}
		values, e := workflowRequestValues(c, map[string]string{"command_id": "command-id"})
		if e != nil {
			return e
		}
		b, e := operatorCommandBody(c, values)
		if e != nil {
			return e
		}
		return operatorRun(c, http.MethodPost, p+"/annotations", b, nil)
	}
	annotate.Args = cobra.ExactArgs(1)
	cmd.AddCommand(annotate)
	retry := workflowBodyCommand("retry <room_id>", http.MethodPost, "/api/think-tanks", map[string]string{"target": "target", "attempt_id": "attempt-id", "command_id": "command-id"})
	retry.RunE = func(c *cobra.Command, args []string) error {
		p, e := idPath("/api/think-tanks", args)
		if e != nil {
			return e
		}
		vals, e := workflowRequestValues(c, map[string]string{"target": "target", "attempt_id": "attempt-id", "command_id": "command-id"})
		if e != nil {
			return e
		}
		b, e := operatorCommandBody(c, vals)
		if e != nil {
			return e
		}
		if !operatorNonEmptyString(b["target"]) {
			return operatorInputError(errors.New("think-tank retry requires --target or a target in --request-file"))
		}
		return operatorRun(c, http.MethodPost, p+"/retry", b, nil)
	}
	retry.Args = cobra.ExactArgs(1)
	cmd.AddCommand(retry)
	limit := workflowBodyCommand("participant-turn-limit <room_id> <agent_id>", http.MethodPost, "/api/think-tanks", map[string]string{"command_id": "command-id"})
	limit.Flags().Int("expected-limit", 0, "expected current limit")
	limit.Flags().Int("limit", 0, "new limit")
	limit.RunE = func(c *cobra.Command, args []string) error {
		if len(args) != 2 {
			return errors.New("room_id and agent_id are required")
		}
		extra, e := workflowRequestValues(c, map[string]string{"command_id": "command-id"})
		if e != nil {
			return e
		}
		if c.Flags().Changed("expected-limit") {
			v, _ := c.Flags().GetInt("expected-limit")
			extra["expected_limit"] = v
		}
		if c.Flags().Changed("limit") {
			v, _ := c.Flags().GetInt("limit")
			extra["limit"] = v
		}
		b, e := operatorCommandBody(c, extra)
		if e != nil {
			return e
		}
		if _, expected := b["expected_limit"]; !expected {
			return operatorInputError(errors.New("participant turn-limit requires --expected-limit and --limit in --request-file"))
		}
		if _, limit := b["limit"]; !limit {
			return operatorInputError(errors.New("participant turn-limit requires --expected-limit and --limit in --request-file"))
		}
		return operatorRun(c, http.MethodPost, "/api/think-tanks/"+url.PathEscape(args[0])+"/participants/"+url.PathEscape(args[1])+"/turn-limit", b, nil)
	}
	limit.Args = cobra.ExactArgs(2)
	cmd.AddCommand(limit)
	del := operatorAction("delete <room_id>", http.MethodDelete, "/api/think-tanks", nil, true)
	del.RunE = func(c *cobra.Command, args []string) error {
		p, e := idPath("/api/think-tanks", args)
		if e != nil {
			return e
		}
		yes, _ := c.Flags().GetBool("yes")
		if !yes {
			return &operatorError{Code: "confirmation_required", Message: "this operation deletes data; pass --yes", ExitCode: 2}
		}
		return operatorRun(c, http.MethodDelete, p, nil, nil)
	}
	del.Args = cobra.ExactArgs(1)
	cmd.AddCommand(del)
	return cmd
}

func operatorNonEmptyString(value any) bool {
	switch v := value.(type) {
	case string:
		return v != ""
	case []byte:
		var text string
		return json.Unmarshal(v, &text) == nil && text != ""
	case json.RawMessage:
		var text string
		return json.Unmarshal(v, &text) == nil && text != ""
	default:
		return false
	}
}

func operatorStringValue(value any) (string, bool) {
	switch v := value.(type) {
	case string:
		return v, true
	case []byte:
		var text string
		if json.Unmarshal(v, &text) == nil {
			return text, true
		}
	case json.RawMessage:
		var text string
		if json.Unmarshal(v, &text) == nil {
			return text, true
		}
	}
	return "", false
}

func operatorNonEmptyArray(value any) bool {
	var raw []byte
	switch v := value.(type) {
	case []byte:
		raw = v
	case json.RawMessage:
		raw = v
	default:
		return false
	}
	var values []json.RawMessage
	return json.Unmarshal(raw, &values) == nil && len(values) > 0
}
