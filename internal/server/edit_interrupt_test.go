package server

import (
	"context"
	"encoding/json"
	"errors"
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

func postEdit(t *testing.T, base, id string) {
	t.Helper()
	latest, _ := lastRootPrompt(t, readTranscriptAPI(t, base, id))
	if resp, raw := post(t, base+"/api/sessions/"+id+"/edit-last", map[string]any{"request_id": "edit-1", "message_seq": latest.Seq, "expected_revision": 0, "text": "third, revised"}); resp.StatusCode != http.StatusAccepted {
		t.Fatalf("edit = %d %s", resp.StatusCode, raw)
	}
}

// holdEditAfterDecision holds this agent's decided edit before it publishes
// the revision, until release is closed.
func holdEditAfterDecision(t *testing.T, id string) (reached <-chan struct{}, release chan struct{}) {
	t.Helper()
	at := make(chan struct{}, 1)
	release = make(chan struct{})
	editAfterDecision = func(agentID string) {
		if agentID == id {
			at <- struct{}{}
			<-release
		}
	}
	t.Cleanup(func() { editAfterDecision = nil })
	return at, release
}

// whileHeld runs action, checks that it waits for the held edit rather than
// conflicting or bypassing it, then releases the edit and returns the result.
func whileHeld[T any](t *testing.T, release chan struct{}, action func() T) T {
	t.Helper()
	done := make(chan T, 1)
	go func() { done <- action() }()
	select {
	case got := <-done:
		t.Fatalf("returned %v while the edit still held its claim", got)
	case <-time.After(300 * time.Millisecond):
	}
	close(release)
	select {
	case got := <-done:
		return got
	case <-time.After(15 * time.Second):
		t.Fatal("action never returned after the edit finished")
	}
	panic("unreachable")
}

func postStatus(url string, body any) int {
	b, _ := json.Marshal(body)
	resp, err := http.Post(url, "application/json", strings.NewReader(string(b)))
	if err != nil {
		return 0
	}
	resp.Body.Close()
	return resp.StatusCode
}

// EDIT-EDGE-3 (TS-01.R51): after the decision, Stop waits for the edit,
// keeps the revision, releases the undispatched replacement and stops the
// revised conversation; turn Cancel lets the replacement dispatch and then
// cancels the revised turn normally.
func TestEditStopAndCancelAfterDecision(t *testing.T) {
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
			reached, release := holdEditAfterDecision(t, id)
			postEdit(t, base, id)
			waitReached(t, reached)
			if got := whileHeld(t, release, func() int { return postStatus(base+"/api/sessions/"+id+tc.path, tc.body) }); got != tc.status {
				t.Fatalf("%s = %d", tc.name, got)
			}
			if rec := waitEdit(t, srv, id, "edit-1", settled); rec.Phase != state.EditCommitted {
				t.Fatalf("edit = %+v", rec)
			}
			receipt, err := srv.stateStore.AttachmentReceipt(id, "edit-1")
			if err != nil || (receipt.Delivery == "released") == tc.running {
				t.Fatalf("receipt = %+v %v", receipt, err)
			}
			if got := srv.registry.Owns(id); got != tc.running {
				t.Fatalf("running = %v, want %v", got, tc.running)
			}
			if snap, _ := srv.stateStore.ReadSession(id); snap.LastSessionID != "fake-fork-1" {
				t.Fatalf("session = %q", snap.LastSessionID)
			}
			if rev, _ := srv.stateStore.HistoryRevision(id); rev != 1 {
				t.Fatalf("revision = %d", rev)
			}
		})
	}
}

// EDIT-EDGE-3 (TS-01.R51): group release and an approved peer Stop reach a
// preparing edit the way the Stop route does instead of conflicting with it.
func TestGroupAndPeerStopReachPreparingEdit(t *testing.T) {
	for _, via := range []string{"group", "peer"} {
		t.Run(via, func(t *testing.T) {
			srv, base, id, _ := forkServer(t, map[string]string{"FAKEACP_COMPLETE_TURN": "1"})
			agent, err := srv.stateStore.ReadAgent(id)
			if err != nil {
				t.Fatal(err)
			}
			agent.Group = "editing"
			if err := srv.stateStore.WriteAgent(agent); err != nil {
				t.Fatal(err)
			}
			reached, _ := holdEditAtDecision(t, id)
			postEdit(t, base, id)
			waitReached(t, reached)
			if via == "group" {
				if resp, raw := post(t, base+"/api/projects/"+agent.Project+"/groups/stop", map[string]string{"group": "editing"}); resp.StatusCode != http.StatusOK {
					t.Fatalf("group stop = %d %s", resp.StatusCode, raw)
				}
			} else if got := srv.executePeerStop(context.Background(), "peer-1", state.Agent{AgentID: "a_requester"},
				state.HistoryAgent{AgentID: id}, peerExecution{}); got.Outcome != "already_stopped" {
				t.Fatalf("peer stop = %+v", got)
			}
			if rec, _ := srv.stateStore.ReadEdit(id, "edit-1"); rec.Phase != state.EditAborted {
				t.Fatalf("edit = %+v", rec)
			}
			if srv.registry.Owns(id) {
				t.Fatal("agent still running")
			}
		})
	}
}

// EDIT-EDGE-3 (TS-01.R51): after the decision, an approved peer Stop waits
// for the edit and does not stop the revised run it was not approved for.
func TestPeerStopAfterEditDecisionIsStale(t *testing.T) {
	srv, base, id, _ := forkServer(t, map[string]string{"FAKEACP_COMPLETE_TURN": "1"})
	before, _, err := srv.peerExecution(id)
	if err != nil {
		t.Fatal(err)
	}
	reached, release := holdEditAfterDecision(t, id)
	postEdit(t, base, id)
	waitReached(t, reached)
	got := whileHeld(t, release, func() string {
		return srv.executePeerStop(context.Background(), "peer-1", state.Agent{AgentID: "a_requester"},
			state.HistoryAgent{AgentID: id}, before).Code
	})
	if got != "peer_stop_stale" {
		t.Fatalf("peer stop = %q", got)
	}
	waitEdit(t, srv, id, "edit-1", settled)
	if receipt, err := srv.stateStore.AttachmentReceipt(id, "edit-1"); err != nil || receipt.Delivery == "released" {
		t.Fatalf("receipt = %+v %v", receipt, err)
	}
	if !srv.registry.Owns(id) {
		t.Fatal("the revised run was stopped")
	}
}

// EDIT-EDGE-1 (TS-01.R51, TS-02.R69): a candidate that cannot publish after
// the commit is stopped, and its undispatched replacement goes to recovery.
func TestEditFailedPublishStopsCandidateAndReleasesReplacement(t *testing.T) {
	srv, base, id, _ := forkServer(t, map[string]string{"FAKEACP_COMPLETE_TURN": "1"})
	editBeforePublish = func(agentID string) error {
		if agentID == id {
			return errors.New("injected publish failure")
		}
		return nil
	}
	t.Cleanup(func() { editBeforePublish = nil })
	postEdit(t, base, id)
	if rec := waitEdit(t, srv, id, "edit-1", settled); rec.Phase != state.EditCommitted {
		t.Fatalf("edit = %+v", rec)
	}
	for deadline := time.Now().Add(10 * time.Second); srv.editRunning(id); time.Sleep(20 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatal("edit worker never finished")
		}
	}
	if receipt, err := srv.stateStore.AttachmentReceipt(id, "edit-1"); err != nil || receipt.Delivery != "released" {
		t.Fatalf("receipt = %+v %v", receipt, err)
	}
	if srv.registry.Owns(id) {
		t.Fatal("unpublished candidate still running")
	}
	if rec, _ := srv.stateStore.ReadEdit(id, "edit-1"); rec.Reason != editReasonNotStarted {
		t.Fatalf("reason = %q", rec.Reason)
	}
	if snap, _ := srv.stateStore.ReadSession(id); snap.LastSessionID != "fake-fork-1" {
		t.Fatalf("session = %q", snap.LastSessionID)
	}
}
