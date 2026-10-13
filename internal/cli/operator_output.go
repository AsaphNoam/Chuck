package cli

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// Detail reads retain every field. Lists and action receipts use compact rows;
// every state word comes from the response rather than inferred model progress.
func operatorHumanOutput(cmd *cobra.Command, value any) (string, bool) {
	if cmd.Name() == "show" || cmd.Name() == "status" || cmd.Name() == "held" {
		return "", false
	}
	if object, ok := value.(map[string]any); ok {
		if results, ok := object["results"].([]any); ok && cmd.Name() != "search" {
			lines := []string{}
			if target := operatorRow(object, operatorOutputTarget(cmd)); target != "" {
				lines = append(lines, target)
			}
			for _, item := range results {
				row, ok := item.(map[string]any)
				if !ok {
					return "", false
				}
				line := operatorRow(row, "")
				if line == "" {
					return "", false
				}
				lines = append(lines, line)
			}
			return strings.Join(lines, "\n"), true
		}
		if cmd.Name() == "list" {
			collection := object
			if nested, ok := object["backends"].(map[string]any); ok {
				collection = nested
			}
			keys := make([]string, 0, len(collection))
			for id, item := range collection {
				if _, ok := item.(map[string]any); !ok {
					return "", false
				}
				keys = append(keys, id)
			}
			sort.Strings(keys)
			if len(keys) == 0 {
				return "No results.", true
			}
			lines := make([]string, 0, len(keys))
			for _, id := range keys {
				row := collection[id].(map[string]any)
				lines = append(lines, operatorRow(row, id))
			}
			return strings.Join(lines, "\n"), true
		}
		if len(object) == 0 {
			return strings.TrimSpace(cmd.Name() + " accepted " + operatorOutputTarget(cmd)), true
		}
		if cmd.Name() == "send" || cmd.Name() == "steer" || cmd.Name() == "stop" || cmd.Name() == "cancel" || cmd.Name() == "clone" || cmd.Name() == "launch" || cmd.Name() == "resume" || cmd.Name() == "rename" || cmd.Name() == "archive" || cmd.Name() == "restore" || cmd.Name() == "approve" || cmd.Name() == "deny" || cmd.Name() == "fork" || cmd.Name() == "background-task-stop" || cmd.Name() == "fire" {
			row := object
			if nested, ok := object["agent"].(map[string]any); ok {
				row = make(map[string]any, len(nested)+len(object))
				for k, v := range nested {
					row[k] = v
				}
				for k, v := range object {
					if k != "agent" {
						row[k] = v
					}
				}
			}
			if line := operatorRow(row, operatorOutputTarget(cmd)); line != "" {
				return line, true
			}
		}
	}
	if list, ok := value.([]any); ok && cmd.Name() == "list" {
		if len(list) == 0 {
			return "No results.", true
		}
		lines := make([]string, 0, len(list))
		for _, item := range list {
			row, ok := item.(map[string]any)
			if !ok {
				return "", false
			}
			line := operatorRow(row, "")
			if line == "" {
				return "", false
			}
			lines = append(lines, line)
		}
		return strings.Join(lines, "\n"), true
	}
	return "", false
}

func operatorOutputTarget(cmd *cobra.Command) string {
	if args := cmd.Flags().Args(); len(args) > 0 {
		return args[0]
	}
	for _, flag := range []string{"id", "project", "backend-id"} {
		if value, err := cmd.Flags().GetString(flag); err == nil && value != "" {
			return value
		}
	}
	return ""
}

func operatorMutationOutput(cmd *cobra.Command) bool {
	if method := cmd.Annotations["operator-method"]; method != "" {
		return isMutationMethod(method)
	}
	switch cmd.Name() {
	case "list", "show", "status", "held", "search", "transcript", "files", "commands", "messages", "entries", "activity", "source-file", "available-commands", "file-search", "file", "attachments", "attachment", "image", "think-tank-results", "devices":
		return false
	}
	return true
}

func operatorRow(row map[string]any, fallback string) string {
	target := fallback
	for _, key := range []string{"agent_id", "task_id", "run_id", "room_id", "backend_id", "role", "project", "id"} {
		if value, ok := row[key].(string); ok && value != "" {
			target = value
			break
		}
	}
	parts := []string{}
	if target != "" {
		parts = append(parts, target)
	}
	for _, key := range []string{"name", "title", "display_name"} {
		if value, ok := row[key].(string); ok && value != "" && value != target {
			parts = append(parts, value)
			break
		}
	}
	for _, key := range []string{"state", "delivery", "outcome", "decision", "branch", "default_model", "type", "group", "tool_call_id", "request_id"} {
		if value, ok := row[key].(string); ok && value != "" {
			parts = append(parts, key+"="+value)
		}
	}
	if status, ok := row["status"].(map[string]any); ok {
		if state, ok := status["state"].(string); ok && state != "" {
			parts = append(parts, "state="+state)
		}
	}
	for _, key := range []string{"accepted", "queued", "stopped", "archived", "restored", "cancelled", "withdrawn", "resumed", "resolved"} {
		if value, ok := row[key].(bool); ok && value {
			parts = append(parts, key)
		}
	}
	if value, ok := row["released"].(json.Number); ok {
		parts = append(parts, "released="+value.String())
	}
	if value, ok := row["ok"].(bool); ok {
		if value {
			parts = append(parts, "ok")
		} else {
			parts = append(parts, "failed")
		}
	}
	if failure, ok := row["error"].(map[string]any); ok {
		for _, key := range []string{"code", "message"} {
			if value, ok := failure[key].(string); ok && value != "" {
				parts = append(parts, value)
			}
		}
	}
	if warning, ok := row["warning"].(string); ok && warning != "" {
		parts = append(parts, "warning: "+warning)
	}
	if worktree, ok := row["worktree"].(map[string]any); ok {
		if warning, ok := worktree["warning"].(string); ok && warning != "" {
			parts = append(parts, "worktree warning: "+warning)
		}
	}
	if warnings, ok := row["warnings"].([]any); ok {
		for _, item := range warnings {
			if w, ok := item.(map[string]any); ok {
				if message, ok := w["message"].(string); ok {
					parts = append(parts, "warning: "+message)
				} else {
					data, _ := json.Marshal(w)
					parts = append(parts, "warning: "+string(data))
				}
			} else {
				parts = append(parts, "warning: "+fmt.Sprint(item))
			}
		}
	}
	for i, part := range parts {
		parts[i] = safeOperatorText(part)
	}
	return strings.Join(parts, " · ")
}
