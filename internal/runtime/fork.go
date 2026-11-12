package runtime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/AsaphNoam/Chuck/internal/attachment"
	"hash"
	"log/slog"
	"time"
)

// Native fork (FS-01.R36, TS-04.R65). Clone launches a fresh adapter process
// that forks the source's provider session through standard ACP session/fork;
// the provider conversation already holds the source context, so no system
// prompt or primer is sent. Replayed history is dropped like session/load's:
// Chuck's own durable copy of the source transcript is the clone's record.

// EvForkBoundary marks where a clone's copied history ends and its own begins.
const EvForkBoundary = "fork_boundary"

// ForkBoundaryData links a clone back to its source.
type ForkBoundaryData struct {
	ForkedFromAgentID string `json:"forked_from_agent_id"`
	ForkedFromSeq     int64  `json:"forked_from_seq"`
}

// ErrForkUnavailable: the new peer did not advertise session fork, so no weaker
// clone path is attempted (FS-01.R36).
var ErrForkUnavailable = errors.New("runtime: this agent's runtime cannot fork its conversation")

const forkDeleteTimeout = 5 * time.Second

func sessionForkParams(spec LaunchSpec, sourceSessionID string) map[string]any {
	mcp := make([]map[string]any, 0, len(spec.MCPServers))
	for _, m := range spec.MCPServers {
		mcp = append(mcp, mcpServerParam(m))
	}
	return map[string]any{
		"sessionId":             sourceSessionID,
		"cwd":                   spec.Cwd,
		"mcpServers":            mcp,
		"additionalDirectories": spec.StartAddDirs(),
	}
}

// openFork forks the plan's source session and returns the new provider
// session id and its response. Only a non-empty returned id is trusted. A plan
// with a fork point sends it only to a verified adapter, then requires the
// fork's own replay to end at that turn's message (TS-04.R110).
func (c *ChatRuntime) openFork(ctx context.Context, as *agentState, spec LaunchSpec) (string, json.RawMessage, error) {
	if !as.capabilities().Fork {
		return "", nil, ErrForkUnavailable
	}
	point := spec.Fork.ForkPoint
	params := sessionForkParams(spec, spec.Fork.SourceSessionID)
	if point != nil {
		if !ForkPointAdapterVerified(as.adapterInfo.Name, as.adapterInfo.Version) {
			return "", nil, &ForkRefusal{Reason: ForkReasonUnverifiedAdapter}
		}
		params["_meta"] = forkMeta(point)
	}
	as.mu.Lock()
	as.loadReplay = true
	as.replayMsg.reset()
	as.mu.Unlock()
	res, err := c.startupCall(ctx, as.transport, "session/fork", params)
	as.mu.Lock()
	as.loadReplay = false
	as.mu.Unlock()
	if err != nil {
		return "", nil, err
	}
	var body struct {
		SessionID string `json:"sessionId"`
	}
	if json.Unmarshal(res, &body) != nil || body.SessionID == "" {
		return "", nil, fmt.Errorf("runtime: session/fork returned no sessionId")
	}
	// Pinned claude-agent-acp 0.85.1 forks only the SDK persistence and never
	// registers the new session, ignoring MCP/directories; open it through the
	// supported load path with this generation's registrations before any
	// configuration or prompt (TS-04.R101). A fork point also needs the fork's
	// own replay, which Codex's fork does not send, so it loads too.
	if forkOpensByLoad(spec.BackendType) || point != nil {
		as.mu.Lock()
		as.loadReplay = true
		as.replayMsg.reset()
		as.mu.Unlock()
		loaded, err := c.startupCall(ctx, as.transport, "session/load", sessionLoadParams(spec, body.SessionID))
		as.mu.Lock()
		as.loadReplay = false
		as.mu.Unlock()
		if err != nil {
			c.deleteForkedSession(as, body.SessionID)
			return "", nil, fmt.Errorf("runtime: open forked session: %w", err)
		}
		if forkOpensByLoad(spec.BackendType) {
			res = loaded
		}
	}
	if point != nil {
		as.mu.Lock()
		replayed := as.replayMsg.fingerprint()
		as.replayMsg.reset()
		as.mu.Unlock()
		if replayed != point.Fingerprint {
			c.deleteForkedSession(as, body.SessionID)
			return "", nil, &ForkRefusal{Reason: ForkReasonProofFailed}
		}
	}
	return body.SessionID, res, nil
}

// ForkRefusal is a fork Chuck declined or could not prove; the clone route
// reports its reason and leaves no partial state (TS-03.R86).
type ForkRefusal struct{ Reason string }

func (e *ForkRefusal) Error() string { return "runtime: " + e.Reason }

// forkOpensByLoad reports adapters whose fork returns an unregistered id.
func forkOpensByLoad(backendType string) bool { return backendType == "claude-acp" }

// deleteForkedSession removes a forked provider session whose local commit
// failed, best-effort, before the process is torn down (TS-01.R35, INV §15).
func (c *ChatRuntime) deleteForkedSession(as *agentState, sessionID string) {
	ctx, cancel := context.WithTimeout(context.Background(), forkDeleteTimeout)
	defer cancel()
	if _, err := as.transport.Call(ctx, "session/delete", map[string]any{"sessionId": sessionID}); err != nil {
		slog.Warn("runtime: delete forked session after failed clone", "agent", as.agentID, "err", err)
	}
}

// writeForkPrefix commits the durable copy of the source's visible transcript
// under the clone's identity and sequence, then marks the boundary. The copy
// is persisted and indexed but not published: it is history, not live work.
// A failed append stops the copy so the launch can roll the clone back.
func (c *ChatRuntime) writeForkPrefix(as *agentState, plan *ForkPlan) error {
	c.mu.Lock()
	home := c.transcriptHome
	c.mu.Unlock()
	ids := map[string]bool{}
	var collect func(any)
	collect = func(v any) {
		switch v := v.(type) {
		case map[string]any:
			if id, ok := v["id"].(string); ok {
				if _, descriptor := v["sha256"]; descriptor {
					ids[id] = true
				}
			}
			for _, x := range v {
				collect(x)
			}
		case []any:
			for _, x := range v {
				collect(x)
			}
		}
	}
	for _, ev := range plan.Prefix {
		var v any
		if err := json.Unmarshal(ev.Data, &v); err != nil {
			return err
		}
		collect(v)
	}
	selected := make([]string, 0, len(ids))
	for id := range ids {
		selected = append(selected, id)
	}
	if len(selected) > 0 {
		if err := attachment.New(home).Clone(plan.SourceAgentID, as.agentID, selected); err != nil {
			return err
		}
	}
	for _, ev := range plan.Prefix {
		ev.Generation = as.generation
		// A copied turn end belongs to the source's native session (TS-02.R65).
		ev.ForkPoint, ev.Fork = nil, nil
		// Clone history is persisted, not replayed as live activity.
		obs := as.journal.Commit(CommitRequest{Event: ev, Committed: as.recordCommitted, Unpublished: true})
		if !obs.Committed() || !obs.IndexOK {
			return fmt.Errorf("runtime: write clone history at seq %d", obs.Event.Seq)
		}
	}
	c.emit(as, EvForkBoundary, ForkBoundaryData{ForkedFromAgentID: plan.SourceAgentID, ForkedFromSeq: plan.SourceSeq})
	if len(selected) > 0 {
		if err := attachment.New(home).Retain(as.agentID, selected); err != nil {
			return err
		}
	}
	return nil
}

// discardForkPersistence removes the clone's partial transcript and index rows
// after a failed launch, so no half-copied clone history survives (INV §15).
func (c *ChatRuntime) discardForkPersistence(as *agentState) {
	c.mu.Lock()
	ix := c.indexer
	c.mu.Unlock()
	defer as.releaseJournal()
	as.closeJournal(true)
	if ix != nil {
		if err := ix.RemoveAgent(as.agentID); err != nil {
			slog.Warn("runtime: remove clone index rows", "agent", as.agentID, "err", err)
		}
	}
}

// ForkPoint is the provider identity of one completed root turn's last agent
// message (TS-04.R110). It is persisted only inside the transcript record of
// that turn_end (TS-02.R65) and never serialized to clients: Event carries it
// as a json:"-" field that only the transcript writer and reader encode.
type ForkPoint struct {
	SessionID      string `json:"session_id"`
	MessageID      string `json:"message_id"`
	Fingerprint    string `json:"fingerprint"` // "sha256:<hex>" of the message text
	AdapterName    string `json:"adapter_name,omitempty"`
	AdapterVersion string `json:"adapter_version,omitempty"`
}

// forkPointAdapters is Chuck's verified fork-point list: adapter identities
// shown to honour `_meta.jetbrains.air.fork` (TS-04.R110). Extend only with a
// recorded receipt.
var forkPointAdapters = map[string]string{
	"@agentclientprotocol/claude-agent-acp": "0.85.1",
	"@agentclientprotocol/codex-acp":        "2.1.1",
}

// ForkPointAdapterVerified reports whether an initialize identity may receive
// a fork point.
func ForkPointAdapterVerified(name, version string) bool {
	v, ok := forkPointAdapters[name]
	return ok && v == version
}

// Fork refusal reasons (TS-03.R86), shown as the action's description.
const (
	ForkReasonNoForkPoint       = "Chuck has no fork point for this turn; it was recorded before forking at a turn was available."
	ForkReasonOtherSession      = "This turn belongs to an earlier provider session that a runtime switch replaced."
	ForkReasonUnverifiedAdapter = "This agent's adapter version is not verified to fork at a chosen turn."
	ForkReasonNotTurnBoundary   = "Forking needs a completed turn."
	ForkReasonProofFailed       = "The provider's fork did not end at the chosen turn, so no agent was created."
)

// messageTracker fingerprints the latest agent message by its standard ACP
// messageId. A chunk without an id invalidates the message: a turn whose last
// message has no id records no fork point.
type messageTracker struct {
	id   string
	hash hash.Hash
}

func (t *messageTracker) reset() { t.id, t.hash = "", nil }

// observe folds one agent_message_chunk's text into the tracker.
func (t *messageTracker) observe(params json.RawMessage) {
	var su struct {
		Update struct {
			MessageID *string `json:"messageId"`
			Content   struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"update"`
	}
	if json.Unmarshal(params, &su) != nil {
		return
	}
	if su.Update.MessageID == nil || *su.Update.MessageID == "" {
		t.reset()
		return
	}
	if *su.Update.MessageID != t.id || t.hash == nil {
		t.id, t.hash = *su.Update.MessageID, sha256.New()
	}
	if su.Update.Content.Type == "text" {
		t.hash.Write([]byte(su.Update.Content.Text))
	}
}

// fingerprint is the adapters' own message fingerprint form, "" when unknown.
func (t *messageTracker) fingerprint() string {
	if t.id == "" || t.hash == nil {
		return ""
	}
	return "sha256:" + hex.EncodeToString(t.hash.Sum(nil))
}

// forkMeta is the only fork-point form both pinned adapters read.
func forkMeta(p *ForkPoint) map[string]any {
	return map[string]any{"jetbrains": map[string]any{"air": map[string]any{"fork": map[string]any{
		"version": 1, "messageId": p.MessageID, "messageFingerprint": p.Fingerprint,
	}}}}
}
