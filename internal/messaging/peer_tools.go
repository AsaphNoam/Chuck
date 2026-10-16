package messaging

import (
	"context"
	"encoding/base64"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/AsaphNoam/Chuck/internal/contextref"
	"github.com/AsaphNoam/Chuck/internal/state"
)

// --- list_agents scope:"history" (FS-06.R40, TS-04.R93) ---

// listAgentHistory pages every durable identity for peer operations. It is
// directory metadata only: listing confers neither mail eligibility nor
// authority to stop, and returns no transcript content.
func (s *Server) listAgentHistory(identity SessionIdentity, in listAgentsArgs) (*mcp.CallToolResult, any, error) {
	limit := intOr(in.Limit, contextref.DefaultListLimit)
	if limit < contextref.MinListLimit || limit > contextref.MaxListLimit {
		return errResult(map[string]any{"ok": false, "error": "validation",
			"message": "limit must be between 1 and 50."})
	}
	filter := state.HistoryFilter{Project: in.Project, State: in.State, Limit: limit}
	if !in.IncludeSelf {
		filter.ExcludeAgentID = identity.AgentID
	}
	binding := historyCursorBinding(identity.AgentID, in)
	if in.Cursor != "" {
		at, id, ok := decodeHistoryCursor(binding, in.Cursor)
		if !ok {
			return errResult(map[string]any{"ok": false, "error": "invalid_cursor",
				"message": "Cursor does not belong to this history listing."})
		}
		filter.AfterCreatedAt, filter.AfterID = at, id
	}
	agents, err := s.store.AgentHistory(filter)
	if err != nil {
		return storeUnavailable(err)
	}
	out := map[string]any{"agents": agents}
	if len(agents) == limit {
		last := agents[len(agents)-1]
		out["next_cursor"] = encodeHistoryCursor(binding, last.CreatedAt, last.AgentID)
	}
	return jsonResult(out)
}

// A history cursor is a keyset position bound to its caller and filters, so
// it cannot be replayed against another listing. It confers no authority.
func historyCursorBinding(caller string, in listAgentsArgs) string {
	return strings.Join([]string{"h1", caller, in.Project, in.State, strconv.FormatBool(in.IncludeSelf)}, "\x00")
}

func encodeHistoryCursor(binding, createdAt, agentID string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(binding + "\x00" + createdAt + "\x00" + agentID))
}

func decodeHistoryCursor(binding, cursor string) (createdAt, agentID string, ok bool) {
	if len(cursor) > contextref.MaxCursorBytes {
		return "", "", false
	}
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return "", "", false
	}
	rest, found := strings.CutPrefix(string(raw), binding+"\x00")
	if !found {
		return "", "", false
	}
	createdAt, agentID, found = strings.Cut(rest, "\x00")
	if !found || createdAt == "" || agentID == "" || strings.Contains(agentID, "\x00") {
		return "", "", false
	}
	return createdAt, agentID, true
}

// --- read_agent_transcript (FS-06.R39, TS-04.R93) ---

type readAgentTranscriptArgs struct {
	AgentID  string `json:"agent_id" jsonschema:"exact agent_id of the agent whose recorded conversation to read"`
	SinceSeq int64  `json:"since_seq,omitempty" jsonschema:"only events after this sequence; pass a previous snapshot_last_seq to read newer activity. Not with cursor"`
	Cursor   string `json:"cursor,omitempty" jsonschema:"next_cursor from the previous page of the same read"`
}

func (s *Server) handleReadAgentTranscript(ctx context.Context, req *mcp.CallToolRequest, in readAgentTranscriptArgs) (*mcp.CallToolResult, any, error) {
	_, svc, ok, result, extra, err := s.contextCaller(req)
	if !ok {
		return result, extra, err
	}
	res, readErr := svc.ReadPeerTranscript(ctx, in.AgentID, in.SinceSeq, in.Cursor)
	if readErr != nil {
		return contextToolError(readErr)
	}
	out := map[string]any{
		"ok":                true,
		"agent_id":          res.AgentID,
		"source":            res.Source,
		"snapshot_last_seq": res.SnapshotLastSeq,
		"text":              res.Text,
		"complete":          res.Complete,
	}
	if res.NextCursor != "" {
		out["next_cursor"] = res.NextCursor
	}
	return jsonResult(out)
}

// --- stop_agent (FS-06.R41–R44, TS-04.R94–R95) ---

// MaxStopReasonRunes bounds the requester's attributed reason.
const MaxStopReasonRunes = 1000

// PeerStopResult is the control plane's answer to one peer Stop request.
// Code is empty for a confirmed outcome ("stopped" or "already_stopped").
// Effect accompanies peer_stop_failed: "unchanged", "stopped" or "unknown".
type PeerStopResult struct {
	Outcome   string
	RequestID string
	Code      string
	Message   string
	Effect    string
}

// PeerControl is the lifecycle plane behind stop_agent. This package owns the
// token-bound identity and tool shape; target identity, the host approval and
// the ordinary Stop path stay with the plane that owns them.
type PeerControl interface {
	StopPeer(ctx context.Context, requester SessionIdentity, targetID, reason string) PeerStopResult
}

// SetPeerControl wires stop_agent to the lifecycle plane.
func (s *Server) SetPeerControl(control PeerControl) {
	s.mu.Lock()
	s.peers = control
	s.mu.Unlock()
}

type stopAgentArgs struct {
	AgentID string `json:"agent_id" jsonschema:"exact agent_id of the other agent to stop"`
	Reason  string `json:"reason" jsonschema:"why it should stop, shown to the user who decides; 1..1000 characters"`
}

func (s *Server) handleStopAgent(ctx context.Context, req *mcp.CallToolRequest, in stopAgentArgs) (*mcp.CallToolResult, any, error) {
	identity, ok := s.caller(req)
	if !ok {
		return sessionUnknown()
	}
	reason := strings.TrimSpace(in.Reason)
	switch {
	case strings.TrimSpace(in.AgentID) == "":
		return errResult(map[string]any{"ok": false, "error": "validation", "message": "agent_id is required."})
	case in.AgentID == identity.AgentID:
		return errResult(map[string]any{"ok": false, "error": "validation",
			"message": "stop_agent stops another agent; it cannot stop you."})
	case reason == "" || utf8.RuneCountInString(reason) > MaxStopReasonRunes:
		return errResult(map[string]any{"ok": false, "error": "validation",
			"message": "reason is required and must be at most 1000 characters."})
	}
	s.mu.RLock()
	control := s.peers
	s.mu.RUnlock()
	if control == nil {
		return errResult(map[string]any{"ok": false, "error": "peer_stop_failed", "effect": "unchanged",
			"message": "Peer Stop is unavailable."})
	}
	res := control.StopPeer(ctx, identity, in.AgentID, reason)
	if res.Code != "" {
		out := map[string]any{"ok": false, "error": res.Code, "agent_id": in.AgentID, "message": res.Message}
		if res.RequestID != "" {
			out["request_id"] = res.RequestID
		}
		if res.Effect != "" {
			out["effect"] = res.Effect
		}
		return errResult(out)
	}
	out := map[string]any{"ok": true, "agent_id": in.AgentID, "outcome": res.Outcome}
	if res.RequestID != "" {
		out["request_id"] = res.RequestID
	}
	return jsonResult(out)
}
