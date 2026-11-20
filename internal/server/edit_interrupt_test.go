package server

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
	"github.com/AsaphNoam/Chuck/internal/transcript"
)

// holdEditAtDecision holds this agent's prepared edit at its decision until
// release is closed or the edit's own context ends (Stop or Cancel).
func holdEditAtDecision(t *testing.T, id string) (reached <-chan struct{}, release chan struct{}) {
	t.Helper()
	at := make(chan struct{}, 1)
	release = make(chan struct{})
	editBeforeDecision = func(ctx context.Context, agentID string) {
		if agentID != id {
			return
		}
		at <- struct{}{}
		select {
		case <-release:
		case <-ctx.Done():
		}
	}
	t.Cleanup(func() { editBeforeDecision = nil })
	return at, release
}

func waitReached(t *testing.T, reached <-chan struct{}) {
	t.Helper()
	select {
	case <-reached:
	case <-time.After(10 * time.Second):
		t.Fatal("edit never reached its decision")
	}
}

func countSessionMeta(t *testing.T, srv *Server, id string) int {
	t.Helper()
	events, err := transcript.ReadFile(srv.configStore.Home(), id, transcript.ReadOptions{Raw: true, IncludeMeta: true})
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, ev := range events {
		if ev.Type == runtime.EvSessionMeta && ev.Seq > 0 {
			n++
		}
	}
	return n
}

// EDIT-2 (TS-01.R51): a prepared candidate publishes no session selection,
// session_meta or running session id before the decision; commit publishes
// them.
func TestEditCandidateStaysUnpublishedUntilDecision(t *testing.T) {
	srv, base, id, _ := forkServer(t, map[string]string{"FAKEACP_COMPLETE_TURN": "1"})
	original, _ := srv.stateStore.ReadSession(id)
	metas := countSessionMeta(t, srv, id)
	reached, release := holdEditAtDecision(t, id)
	latest, _ := lastRootPrompt(t, readTranscriptAPI(t, base, id))
	if resp, raw := post(t, base+"/api/sessions/"+id+"/edit-last", map[string]any{"request_id": "edit-1", "message_seq": latest.Seq, "expected_revision": 0, "text": "third, revised"}); resp.StatusCode != http.StatusAccepted {
		t.Fatalf("edit = %d %s", resp.StatusCode, raw)
	}
	waitReached(t, reached)
	if snap, _ := srv.stateStore.ReadSession(id); snap.LastSessionID != original.LastSessionID {
		t.Fatalf("candidate selected before decision: %q (original %q)", snap.LastSessionID, original.LastSessionID)
	}
	if row, err := srv.stateStore.ReadRunning(id); err != nil || row.SessionID != "" {
		t.Fatalf("staged running row = %+v %v", row, err)
	}
	if got := countSessionMeta(t, srv, id); got != metas {
		t.Fatalf("session_meta published before decision: %d -> %d", metas, got)
	}
	close(release)
	if rec := waitEdit(t, srv, id, "edit-1", settled); rec.Phase != state.EditCommitted {
		t.Fatalf("edit = %+v", rec)
	}
	snap, _ := srv.stateStore.ReadSession(id)
	row, _ := srv.stateStore.ReadRunning(id)
	if snap.LastSessionID != "fake-fork-1" || row.SessionID != "fake-fork-1" || countSessionMeta(t, srv, id) != metas+1 {
		t.Fatalf("after commit: session %q running %q metas %d", snap.LastSessionID, row.SessionID, countSessionMeta(t, srv, id))
	}
}

// EDIT-1 (TS-01.R51): before the decision, Stop aborts the edit and leaves
// the agent stopped; turn Cancel aborts it and restores the running source.
func TestEditStopAndCancelAbortPreparation(t *testing.T) {
	for _, tc := range []struct {
		name, path string
		body       any
		status     int
		running    bool
	}{
		{"stop", "/stop", nil, http.StatusOK, false},
		{"cancel", "/cancel", map[string]string{"target": "turn"}, http.StatusAccepted, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv, base, id, _ := forkServer(t, map[string]string{"FAKEACP_COMPLETE_TURN": "1"})
			original, _ := srv.stateStore.ReadSession(id)
			reached, _ := holdEditAtDecision(t, id)
			latest, _ := lastRootPrompt(t, readTranscriptAPI(t, base, id))
			if resp, raw := post(t, base+"/api/sessions/"+id+"/edit-last", map[string]any{"request_id": "edit-1", "message_seq": latest.Seq, "expected_revision": 0, "text": "third, revised"}); resp.StatusCode != http.StatusAccepted {
				t.Fatalf("edit = %d %s", resp.StatusCode, raw)
			}
			waitReached(t, reached)
			if resp, raw := post(t, base+"/api/sessions/"+id+tc.path, tc.body); resp.StatusCode != tc.status {
				t.Fatalf("%s = %d %s", tc.name, resp.StatusCode, raw)
			}
			rec, _ := srv.stateStore.ReadEdit(id, "edit-1")
			if rec.Phase != state.EditAborted || !strings.Contains(rec.Reason, "Nothing was changed") {
				t.Fatalf("edit = %+v", rec)
			}
			if got := srv.registry.Owns(id); got != tc.running {
				t.Fatalf("running = %v, want %v", got, tc.running)
			}
			// The source session is kept; a restarted source reloads it, and the
			// fake echoes its own load id. The candidate fork never wins.
			if snap, _ := srv.stateStore.ReadSession(id); snap.LastSessionID == "fake-fork-1" || snap.LastSessionID == "" ||
				!tc.running && snap.LastSessionID != original.LastSessionID {
				t.Fatalf("session = %q (original %q)", snap.LastSessionID, original.LastSessionID)
			}
			if rev, _ := srv.stateStore.HistoryRevision(id); rev != 0 {
				t.Fatalf("revision = %d", rev)
			}
			if got := promptTexts(readTranscriptAPI(t, base, id)); got[len(got)-1] != "third" {
				t.Fatalf("prompts = %v", got)
			}
		})
	}
}

// EDIT-3 (TS-03.R90): a kept attachment whose retained bytes no longer verify
// refuses the edit before acceptance, with nothing recorded.
func TestEditRefusesUnavailableAttachmentBeforeAcceptance(t *testing.T) {
	srv, h, id := attachmentTestServer(t, map[string]string{"FAKEACP_CAPS": "1"})
	part := uploadPart{field: "asset1", name: "kept.txt", data: []byte("kept bytes")}
	if resp, body := postAttachmentMessage(t, h, "/api/sessions/"+id+"/prompt", messageMetadata(t, "send-1", "look", part), part); resp.StatusCode != http.StatusAccepted {
		t.Fatalf("send = %d %s", resp.StatusCode, body)
	} else {
		kept := decodeAttachmentAcceptance(t, body).Attachments[0]
		waitStatus(t, srv, id, "idle")
		deadline := time.Now().Add(5 * time.Second)
		for len(rootTurnEnds(readEvents(t, srv, id))) < 1 && time.Now().Before(deadline) {
			time.Sleep(10 * time.Millisecond)
		}
		blob := filepath.Join(srv.configStore.Home(), "sessions", id, "attachments", kept.ID, "blob")
		if err := os.WriteFile(blob, []byte("corrupt!!!"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	latest, _ := lastRootPrompt(t, readEvents(t, srv, id))
	resp, raw := postJSONHandler(t, h, "/api/sessions/"+id+"/edit-last", map[string]any{"request_id": "edit-1", "message_seq": latest.Seq, "expected_revision": 0, "text": "look again"})
	if resp.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(string(raw), "no longer available") {
		t.Fatalf("edit = %d %s", resp.StatusCode, raw)
	}
	if _, err := srv.stateStore.ReadEdit(id, "edit-1"); err == nil {
		t.Fatal("a refused edit left a record")
	}
	if !srv.registry.Owns(id) {
		t.Fatal("refusal stopped the agent")
	}
}

// EDIT-4 (TS-02.R70): a committed revision whose search rebuild fails
// withdraws the conversation's indexed text and reports the edit's reason.
func TestEditFailedReindexWithdrawsSearch(t *testing.T) {
	srv, _, id := cloneServer(t, map[string]string{"FAKEACP_CAPS": "1"})
	waitStatus(t, srv, id, "idle")
	first, _ := lastRootPrompt(t, readEvents(t, srv, id))
	if err := srv.indexer.ReindexAgent(srv.configStore.Home(), id); err != nil {
		t.Fatal(err)
	}
	if searchRows(t, srv, id) == 0 {
		t.Fatal("no indexed search text before the edit")
	}
	// A malformed session record makes every rebuild of this conversation fail.
	if _, err := srv.registry.AppendHistory(id, runtime.CommitRequest{Event: runtime.Event{
		Type: runtime.EvSessionMeta, Data: json.RawMessage(`"broken"`), Ts: time.Now().UTC().Format(time.RFC3339),
	}}); err != nil {
		t.Fatal(err)
	}
	msg, _ := json.Marshal(runtime.Message{Text: "replacement", RequestID: "edit-1"})
	if err := srv.stateStore.CreateEdit(state.EditTransition{AgentID: id, RequestID: "edit-1", PayloadHash: "h", MessageJSON: string(msg), MessageSeq: first.Seq}); err != nil {
		t.Fatal(err)
	}
	if _, err := srv.stateStore.DecideEdit(id, "edit-1", first.Seq, first.Seq+10, "candidate"); err != nil {
		t.Fatal(err)
	}
	srv.recoverEdits()
	rec, _ := srv.stateStore.ReadEdit(id, "edit-1")
	if rec.Phase != state.EditCommitted || rec.Reason != editReasonSearchUnavailable {
		t.Fatalf("edit = %+v", rec)
	}
	if got := searchRows(t, srv, id); got != 0 {
		t.Fatalf("superseded search text still indexed: %d rows", got)
	}
}

// searchRows counts the conversation's indexed search text: span provenance
// in every build, plus full-text rows where FTS is compiled in.
func searchRows(t *testing.T, srv *Server, id string) int {
	t.Helper()
	db := srv.stateStore.DB()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM session_search_spans WHERE agent_id=?`, id).Scan(&n); err != nil {
		t.Fatal(err)
	}
	var fts int
	if db.QueryRow(`SELECT COUNT(*) FROM sessions_fts WHERE agent_id=?`, id).Scan(&fts) == nil {
		n += fts
	}
	return n
}
