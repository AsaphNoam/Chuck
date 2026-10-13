package cli

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
)

func operatorInvalid(message string) error {
	return &operatorError{Code: "invalid_input", Message: message, ExitCode: 2}
}

// These named commands describe the existing session API; request-file keeps
// nested operation shapes available without a second set of server semantics.
func newOperatorAgentCmd() *cobra.Command {
	parent := &cobra.Command{Use: "agent", Short: "Inspect and control a named conversation"}
	list := agentOperation("list", "GET", "", 0, false, false, nil)
	parent.AddCommand(list)
	for _, suffix := range []string{"show", "held", "files", "commands", "available-commands", "messages", "think-tank-results"} {
		route := suffix
		if suffix == "show" {
			route = ""
		}
		if suffix == "held" {
			route = "prompt"
		}
		cmd := agentOperation(suffix+" <id>", "GET", route, 1, false, false, nil)
		if suffix == "messages" {
			cmd.Flags().Int("limit", 50, "newest messages, at most 200")
			cmd.Flags().Bool("unread-only", false, "only unread messages")
		}
		if suffix == "think-tank-results" {
			cmd.Flags().Int64("after", 0, "result cursor from the prior page")
		}
		parent.AddCommand(cmd)
	}
	launch := agentOperation("launch", "POST", "", 0, false, false, []string{"role", "project"})
	for _, field := range []string{"role", "project", "backend", "model", "effort", "interface", "driver", "name", "group"} {
		launch.Flags().String(field, "", "launch "+field)
	}
	launch.Flags().Bool("fast", false, "request fast mode")
	launch.Long = "Always create a new agent. Request JSON: {role,project,backend?,model?,effort?,fast?,interface?,driver?,name?,group?}. No dashboard auto-start."
	launch.Example = "chuck agent launch --role default --project my-project --json"
	parent.AddCommand(launch)
	for _, action := range []string{"resume", "stop", "clone", "archive", "restore"} {
		parent.AddCommand(agentOperation(action+" <id>", "POST", action, 1, false, false, nil))
	}
	rename := agentOperation("rename <id>", "POST", "rename", 1, false, false, []string{"name"})
	rename.Flags().String("name", "", "new display name")
	parent.AddCommand(rename)
	identity := agentOperation("identity <id>", "POST", "identity", 1, false, false, nil)
	identity.Flags().String("name", "", "display name")
	identity.Flags().String("group", "", "project-scoped group; empty clears membership")
	parent.AddCommand(identity)
	for _, action := range []string{"switch-runtime", "session-config"} {
		cmd := agentOperation(action+" <id>", "POST", action, 1, false, false, nil)
		for _, field := range []string{"effort", "interface", "backend", "model", "driver"} {
			if action == "switch-runtime" || field == "effort" {
				cmd.Flags().String(field, "", field)
			}
		}
		if action == "session-config" {
			cmd.Flags().Bool("fast", false, "enable or disable fast mode")
		}
		parent.AddCommand(cmd)
	}
	cancel := agentOperation("cancel <id>", "POST", "cancel", 1, false, false, nil)
	cancel.Flags().String("target", "turn", "turn or quota_continuation")
	cancel.Flags().String("episode-id", "", "exact quota episode id")
	cancel.Flags().Int64("revision", 0, "exact quota revision")
	parent.AddCommand(cancel)
	for _, action := range []string{"approve", "deny"} {
		cmd := agentOperation(action+" <id>", "POST", "permission", 1, false, false, []string{"tool_call_id"})
		cmd.Flags().String("tool-call-id", "", "exact pending tool call id")
		parent.AddCommand(cmd)
	}
	background := agentOperation("background-task-stop <id>", "POST", "background-task-stop", 1, false, false, []string{"task_id"})
	background.Flags().String("task-id", "", "native background task id")
	background.Flags().String("activity-id", "", "native child activity id")
	parent.AddCommand(background)
	for _, action := range []string{"send", "steer"} {
		route := "prompt"
		if action == "steer" {
			route = "steer"
		}
		cmd := agentOperation(action+" <id>", "POST", route, 1, true, false, nil)
		cmd.Flags().StringArray("attach", nil, "local file path (repeatable); 10 files, 10 MiB each, 25 MiB total")
		cmd.Flags().String("request-id", "", "receipt id; identical explicit retries reuse this id and input")
		cmd.Long = "Submit text or attachments. Request JSON: {text,request_id?,attachment_ids?}. Send reports sent/held; Steer reports steered/new_turn. Acceptance is not model completion. After an unknown result, inspect held/history; only receipt-bearing requests support identical retry."
		cmd.Example = "chuck agent " + action + " AGENT_ID --text-file message.txt --json"
		parent.AddCommand(cmd)
	}
	withdraw := agentOperation("withdraw <id>", "DELETE", "prompt", 1, false, false, nil)
	withdraw.Flags().String("recovery-request-id", "", "dismiss only this owned recovered message")
	parent.AddCommand(withdraw)
	parent.AddCommand(agentOperation("annotate <id>", "POST", "annotations", 1, false, false, []string{"annotations", "target"}))
	search := agentOperation("file-search <id>", "GET", "file-search", 1, false, false, nil)
	search.Flags().String("query", "", "workspace filename query")
	parent.AddCommand(search)
	file := agentOperation("file <id>", "GET", "file", 1, false, false, nil)
	file.Flags().String("path", "", "explicit source path")
	parent.AddCommand(file)
	parent.AddCommand(newAgentTranscriptCmd(), newAgentDownloadCmd(false), newAgentDownloadCmd(true), newOperatorAgentStatusCmd())
	return parent
}

func agentOperation(use, method, suffix string, nargs int, text, yes bool, required []string) *cobra.Command {
	cmd := &cobra.Command{Use: use, Short: strings.Split(use, " ")[0] + " conversation state", Args: cobra.ExactArgs(nargs)}
	addOperatorFlags(cmd)
	if text {
		addOperatorTextFlags(cmd)
	}
	if yes {
		cmd.Flags().Bool("yes", false, "acknowledge deletion")
	}
	cmd.Long = "Use stable agent ids. JSON input uses the existing operation shape through --request-file <path|->. Failures retain server ownership and capability guards."
	shapes := map[string]string{
		"launch": "{role,project,backend?,model?,effort?,fast?,interface?,driver?,name?,group?}; always creates a new agent",
		"rename": "{name}", "identity": "{name?,group?}; an explicit empty group removes membership",
		"switch-runtime": "{interface,backend?,model?,effort?,driver?}", "session-config": "{effort?,fast?}; explicit false disables fast mode",
		"cancel":  "{target:\"turn\"} or {target:\"quota_continuation\",episode_id,revision}; quota cancellation requires the exact positive revision",
		"approve": "{tool_call_id}; approves only the exact pending tool call", "deny": "{tool_call_id}; denies only the exact pending tool call",
		"background-task-stop": "{task_id,activity_id?}; reports acceptance, then inspect native task state for its terminal update",
		"annotate":             "{annotations:[{anchor,seq?,path?,side?,start_line?,end_line?,excerpt,instruction}],overall_instruction?,target:{kind:\"self\"|\"agent\",agent_id?}}; launch and annotate are separate operations",
		"held":                 "no input; inspect the current held or recovered message", "withdraw": "no body; --recovery-request-id dismisses only that recovered receipt",
		"file": "--path; reads text without running commands", "file-search": "--query; bounded workspace filename search",
		"available-commands": "no input; discovery never executes a provider command", "messages": "--limit 1–200 and --unread-only", "think-tank-results": "--after durable result cursor",
	}
	if shape := shapes[strings.Split(use, " ")[0]]; shape != "" {
		cmd.Long += " Input: " + shape + "."
	} else {
		cmd.Long += " No request body; returns the existing server state including unknown and unavailable values."
	}
	cmd.Example = "chuck agent " + strings.Split(use, " ")[0] + " AGENT_ID --json"
	if nargs == 0 {
		cmd.Example = "chuck agent " + use + " --json"
	}
	cmd.RunE = func(c *cobra.Command, args []string) error {
		body, err := operatorInput(c)
		if err != nil {
			return err
		}
		for _, field := range []string{"role", "project", "backend", "model", "effort", "interface", "driver", "name", "group", "fast", "target", "episode-id", "revision", "tool-call-id", "task-id", "activity-id", "request-id"} {
			if c.Flags().Lookup(field) == nil || !c.Flags().Changed(field) {
				continue
			}
			var value any
			if field == "fast" {
				value, _ = c.Flags().GetBool(field)
			} else if field == "revision" {
				value, _ = c.Flags().GetInt64(field)
			} else {
				value, _ = c.Flags().GetString(field)
			}
			if err := operatorMerge(body, strings.ReplaceAll(field, "-", "_"), value); err != nil {
				return operatorInvalid(err.Error())
			}
		}
		if c.Name() == "cancel" {
			if _, ok := body["target"]; !ok {
				body["target"] = json.RawMessage(`"turn"`)
			}
			var target string
			_ = json.Unmarshal(body["target"], &target)
			if target != "turn" && target != "quota_continuation" {
				return operatorInvalid("--target must be turn or quota_continuation")
			}
			if target == "quota_continuation" {
				var episode string
				var revision int64
				_ = json.Unmarshal(body["episode_id"], &episode)
				_ = json.Unmarshal(body["revision"], &revision)
				if episode == "" || revision < 1 {
					return operatorInvalid("quota cancellation requires --episode-id and positive --revision")
				}
			}
		}
		if c.Name() == "approve" || c.Name() == "deny" {
			if err := operatorMerge(body, "decision", c.Name()); err != nil {
				return operatorInvalid(err.Error())
			}
		}
		for _, key := range required {
			if raw := body[key]; len(raw) == 0 || string(raw) == `""` || string(raw) == "null" {
				return operatorInvalid(fmt.Sprintf("%s is required via a flag or --request-file", key))
			}
		}
		path := "/api/sessions"
		if len(args) > 0 {
			path += "/" + url.PathEscape(args[0])
		}
		if suffix != "" {
			path += "/" + suffix
		}
		query := url.Values{}
		for _, pair := range [][2]string{{"query", "q"}, {"path", "path"}, {"recovery-request-id", "recovery_request_id"}} {
			if c.Flags().Lookup(pair[0]) != nil && c.Flags().Changed(pair[0]) {
				v, _ := c.Flags().GetString(pair[0])
				query.Set(pair[1], v)
			}
		}
		if c.Name() == "messages" {
			limit, _ := c.Flags().GetInt("limit")
			if limit < 1 || limit > 200 {
				return operatorInvalid("--limit must be 1–200")
			}
			query.Set("limit", fmt.Sprint(limit))
			unread, _ := c.Flags().GetBool("unread-only")
			query.Set("unread_only", fmt.Sprint(unread))
		}
		if c.Flags().Lookup("after") != nil {
			after, _ := c.Flags().GetInt64("after")
			if after < 0 {
				return operatorInvalid("--after must be nonnegative")
			}
			query.Set("after", fmt.Sprint(after))
		}
		if c.Name() == "file" {
			p, _ := c.Flags().GetString("path")
			if p == "" {
				return operatorInvalid("--path is required")
			}
		}
		if len(query) > 0 {
			path += "?" + query.Encode()
		}
		if text {
			return operatorMessage(c, method, path, body)
		}
		if method == "GET" || method == "DELETE" {
			if len(body) > 0 {
				return operatorInvalid("this read/delete does not accept request JSON")
			}
			return operatorRun(c, method, path, nil, nil)
		}
		return operatorRun(c, method, path, body, nil)
	}
	return cmd
}

func newOperatorStatusCmd() *cobra.Command {
	c := &cobra.Command{Use: "status", Short: "Inspect local service health and terminal capabilities", Args: cobra.NoArgs}
	addOperatorFlags(c)
	c.Example = "chuck status --json"
	c.RunE = func(c *cobra.Command, _ []string) error {
		if request, _ := c.Flags().GetString("request-file"); request != "" {
			return operatorInvalid("status does not accept request JSON")
		}
		client, err := operatorClientFactory()
		if err != nil {
			return err
		}
		ctx, cancel, err := operatorContext(c)
		if err != nil {
			return err
		}
		defer cancel()
		health, err := client.Request(ctx, "GET", "/api/health", nil, nil)
		if err != nil {
			return err
		}
		caps, err := client.Request(ctx, "GET", "/api/capabilities", nil, nil)
		if err != nil {
			return err
		}
		data, _ := json.Marshal(map[string]json.RawMessage{"health": health, "capabilities": caps})
		return operatorOutput(c, data)
	}
	return c
}
