package messaging

import (
	"context"
	"encoding/base64"
	"strconv"
	"strings"

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
