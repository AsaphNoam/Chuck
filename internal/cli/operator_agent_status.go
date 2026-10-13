package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const operatorStatusPermissionWindow = 500

type operatorAgentStatusResult struct {
	Session       json.RawMessage            `json:"session"`
	Snapshot      json.RawMessage            `json:"snapshot"`
	Held          json.RawMessage            `json:"held"`
	HeldAvailable bool                       `json:"held_available"`
	Permissions   operatorPendingPermissions `json:"permissions"`
}

type operatorPendingPermissions struct {
	Pending  []json.RawMessage `json:"pending"`
	Complete bool              `json:"complete"`
}

func newOperatorAgentStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "status <id>",
		Short:   "Show current conversation capabilities and pending work",
		Long:    "Compose the authoritative session, hydrated live capability snapshot, held or recovered prompt, and pending permissions from existing local API reads. Permission completeness is false when older transcript events fall outside the bounded window. This command only reads state.",
		Example: "chuck agent status AGENT_ID --json",
		Args:    cobra.ExactArgs(1),
	}
	cmd.Flags().Bool("json", false, "write machine-readable JSON")
	cmd.Flags().Duration("timeout", 30*time.Second, "request timeout")
	cmd.RunE = func(c *cobra.Command, args []string) error {
		ctx, cancel, err := operatorContext(c)
		if err != nil {
			return err
		}
		defer cancel()
		id := args[0]
		if strings.TrimSpace(id) == "" {
			return operatorInvalid("agent id is required")
		}
		client, err := operatorClientFactory()
		if err != nil {
			return err
		}

		session, err := client.Request(ctx, http.MethodGet, "/api/sessions/"+url.PathEscape(id), nil, nil)
		if err != nil {
			return err
		}
		if err := operatorValidateStatusSession(session, id); err != nil {
			return err
		}
		snapshot, err := operatorStatusSnapshot(ctx, client, id)
		if err != nil {
			return err
		}

		result := operatorAgentStatusResult{
			Session:       session,
			Snapshot:      snapshot,
			Held:          json.RawMessage("null"),
			HeldAvailable: true,
		}
		held, err := client.Request(ctx, http.MethodGet, "/api/sessions/"+url.PathEscape(id)+"/prompt", nil, nil)
		if err != nil {
			var apiErr *operatorError
			if !errors.As(err, &apiErr) || apiErr.HTTPStatus != http.StatusNotFound {
				return err
			}
			// The session detail already established the target. A 404 can now
			// mean a stopped runtime or a live-to-stopped race; neither proves
			// that a held or recoverable message is currently available.
			result.HeldAvailable = false
		} else {
			result.Held = held
		}

		permissions, err := operatorStatusPermissions(ctx, client, id)
		if err != nil {
			return err
		}
		result.Permissions = permissions
		data, err := json.Marshal(result)
		if err != nil {
			return operatorInvalid("could not compose agent status")
		}
		return operatorOutput(c, data)
	}
	return cmd
}

func operatorValidateStatusSession(session json.RawMessage, id string) error {
	var value struct {
		Agent struct {
			AgentID string `json:"agent_id"`
		} `json:"agent"`
	}
	if json.Unmarshal(session, &value) != nil || value.Agent.AgentID != id {
		return &operatorError{Code: "invalid_response", Message: "dashboard returned session data for a different agent", ExitCode: 1}
	}
	return nil
}

func operatorStatusSnapshot(ctx context.Context, client *operatorClient, id string) (json.RawMessage, error) {
	// operatorSubscribe owns and closes one global stream; it keeps only the
	// bounded event channel and the exact target's snapshot survives here.
	end := time.Now().Add(30 * time.Second)
	observer, err := operatorSubscribe(ctx, client, id, end)
	if err != nil {
		return nil, err
	}
	defer observer.close()
	var snapshot json.RawMessage
	for {
		select {
		case <-ctx.Done():
			return nil, &operatorError{Code: "dashboard_unavailable", Message: "timed out waiting for the live status snapshot", ExitCode: 3}
		case frame, ok := <-observer.frames:
			if !ok {
				return nil, operatorStatusStreamError(<-observer.done)
			}
			if frame.event == "state_update" && operatorHydrated(frame.data) {
				observer.release(frame)
				if len(snapshot) == 0 {
					return json.RawMessage("null"), nil
				}
				return snapshot, nil
			}
			if frame.event == "state_update" && operatorFrameAgent(frame.data) == id {
				var envelope struct {
					Data json.RawMessage `json:"data"`
				}
				if json.Unmarshal(frame.data, &envelope) != nil || len(envelope.Data) == 0 || !json.Valid(envelope.Data) {
					observer.release(frame)
					return nil, &operatorError{Code: "invalid_response", Message: "dashboard returned an invalid live status snapshot", ExitCode: 1}
				}
				snapshot = append(snapshot[:0], envelope.Data...)
			}
			observer.release(frame)
		}
	}
}

func operatorStatusStreamError(err error) error {
	var op *operatorError
	if errors.As(err, &op) {
		return err
	}
	return &operatorError{Code: "stream_gap", Message: "event stream ended before its hydrated status marker", ExitCode: 1}
}

func operatorStatusPermissions(ctx context.Context, client *operatorClient, id string) (operatorPendingPermissions, error) {
	query := url.Values{"include_meta": {"true"}, "limit": {fmt.Sprint(operatorStatusPermissionWindow)}}
	data, err := client.Request(ctx, http.MethodGet, "/api/sessions/"+url.PathEscape(id)+"/transcript?"+query.Encode(), nil, nil)
	if err != nil {
		return operatorPendingPermissions{}, err
	}
	var page struct {
		AgentID           string            `json:"agent_id"`
		Events            []json.RawMessage `json:"events"`
		HasMore           *bool             `json:"has_more"`
		PendingPermission json.RawMessage   `json:"pending_permission"`
	}
	if json.Unmarshal(data, &page) != nil || page.AgentID != id || page.Events == nil || len(page.Events) > operatorStatusPermissionWindow {
		return operatorPendingPermissions{}, &operatorError{Code: "invalid_response", Message: "dashboard returned an invalid bounded permission history", ExitCode: 1}
	}

	pending := make([]json.RawMessage, 0, 4)
	toolIDs := make([]string, 0, 4)
	for _, raw := range page.Events {
		var event struct {
			AgentID    string          `json:"agent_id"`
			Type       string          `json:"type"`
			Data       json.RawMessage `json:"data"`
			ActivityID string          `json:"activity_id"`
		}
		if json.Unmarshal(raw, &event) != nil || event.Type == "" || (event.AgentID != "" && event.AgentID != id) {
			return operatorPendingPermissions{}, &operatorError{Code: "invalid_response", Message: "dashboard returned an invalid transcript event", ExitCode: 1}
		}
		if event.Type != "permission_request" && event.Type != "permission_resolved" {
			continue
		}
		var permission struct {
			ToolCallID string `json:"tool_call_id"`
		}
		if json.Unmarshal(event.Data, &permission) != nil || permission.ToolCallID == "" {
			return operatorPendingPermissions{}, &operatorError{Code: "invalid_response", Message: "dashboard returned an invalid permission event", ExitCode: 1}
		}
		index := -1
		for i, toolID := range toolIDs {
			if toolID == permission.ToolCallID {
				index = i
				break
			}
		}
		switch event.Type {
		case "permission_request":
			if index >= 0 {
				pending = append(pending[:index], pending[index+1:]...)
				toolIDs = append(toolIDs[:index], toolIDs[index+1:]...)
			}
			pending = append(pending, append(json.RawMessage(nil), raw...))
			toolIDs = append(toolIDs, permission.ToolCallID)
		case "permission_resolved":
			if index >= 0 {
				pending = append(pending[:index], pending[index+1:]...)
				toolIDs = append(toolIDs[:index], toolIDs[index+1:]...)
			}
		}
	}
	if len(page.PendingPermission) > 0 && !bytes.Equal(bytes.TrimSpace(page.PendingPermission), []byte("null")) {
		var event struct {
			AgentID string          `json:"agent_id"`
			Type    string          `json:"type"`
			Data    json.RawMessage `json:"data"`
		}
		var permission struct {
			ToolCallID string `json:"tool_call_id"`
		}
		if json.Unmarshal(page.PendingPermission, &event) != nil || event.Type != "permission_request" || event.AgentID != id || json.Unmarshal(event.Data, &permission) != nil || permission.ToolCallID == "" {
			return operatorPendingPermissions{}, &operatorError{Code: "invalid_response", Message: "dashboard returned an invalid pending permission", ExitCode: 1}
		}
		found := false
		for _, toolID := range toolIDs {
			if toolID == permission.ToolCallID {
				found = true
				break
			}
		}
		if !found {
			pending = append(pending, append(json.RawMessage(nil), page.PendingPermission...))
		}
	}
	return operatorPendingPermissions{
		Pending:  pending,
		Complete: page.HasMore != nil && !*page.HasMore,
	}, nil
}
