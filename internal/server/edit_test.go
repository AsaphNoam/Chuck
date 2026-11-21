package server

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
	"github.com/AsaphNoam/Chuck/internal/transcript"
)

func lastRootPrompt(t *testing.T, events []runtime.Event) (runtime.Event, string) {
	t.Helper()
	for i := len(events) - 1; i >= 0; i-- {
		ev := events[i]
		if ev.Type == runtime.EvUserPrompt && ev.ActivityID == "" {
			var d runtime.UserPromptData
			_ = json.Unmarshal(ev.Data, &d)
			return ev, d.Text
		}
	}
	t.Fatal("no person message")
	return runtime.Event{}, ""
}

func waitEdit(t *testing.T, srv *Server, id, requestID string, done func(state.EditTransition) bool) state.EditTransition {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		rec, err := srv.stateStore.ReadEdit(id, requestID)
		if err == nil && done(rec) {
			return rec
		}
		if time.Now().After(deadline) {
			t.Fatalf("edit did not settle: %+v %v", rec, err)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func settled(rec state.EditTransition) bool {
	return rec.Phase == state.EditAborted || rec.Phase == state.EditCommitted && rec.Dispatched
}

func promptTexts(events []runtime.Event) []string {
	var out []string
	for _, ev := range events {
		if ev.Type == runtime.EvUserPrompt && ev.ActivityID == "" {
			var d runtime.UserPromptData
			_ = json.Unmarshal(ev.Data, &d)
			out = append(out, d.Text)
		}
	}
	return out
}

// FS-01.A43 against the fake: editing the third message keeps the first two
// turns in provider and Chuck history, replaces the third turn under the same
// agent, retains the superseded bytes, and is idempotent by request id.
func TestEditLastReplacesTheLatestTurnInPlace(t *testing.T) {
	srv, base, id, metaDump := forkServer(t, map[string]string{"FAKEACP_COMPLETE_TURN": "1"})
	before := readTranscriptAPI(t, base, id)
	latest, text := lastRootPrompt(t, before)
	if text != "third" {
		t.Fatalf("latest message = %q", text)
	}
	body := map[string]any{"request_id": "edit-1", "message_seq": latest.Seq, "expected_revision": 0, "text": "third, revised"}
	resp, raw := post(t, base+"/api/sessions/"+id+"/edit-last", body)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("edit = %d %s", resp.StatusCode, raw)
	}
	rec := waitEdit(t, srv, id, "edit-1", settled)
	if rec.Phase != state.EditCommitted || rec.Revision != 1 {
		t.Fatalf("edit record = %+v", rec)
	}
	waitStatus(t, srv, id, "idle")
	deadline := time.Now().Add(5 * time.Second)
	for len(rootTurnEnds(readEvents(t, srv, id))) < 3 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	waitTurnReleased(t, srv, id)
	meta, _ := os.ReadFile(metaDump)
	if !strings.Contains(string(meta), `"messageId":"msg-2"`) {
		t.Fatalf("edit did not fork after turn 2: %s", meta)
	}
	active := readTranscriptAPI(t, base, id)
	if got := promptTexts(active); strings.Join(got, "|") != "first|second|third, revised" && strings.Join(got[1:], "|") != "second|third, revised" {
		t.Fatalf("active prompts = %v", got)
	}
	if got := len(rootTurnEnds(active)); got != 3 {
		t.Fatalf("active turn ends = %d, want 3", got)
	}
	raw2, _ := json.Marshal(active)
	if strings.Contains(string(raw2), `"text":"third"`) {
		t.Fatalf("superseded message still active: %s", raw2)
	}
	retained, err := transcript.ReadFile(srv.configStore.Home(), id, transcript.ReadOptions{Raw: true})
	if err != nil {
		t.Fatal(err)
	}
	if got := promptTexts(retained); !strings.Contains(strings.Join(got, "|"), "third|third, revised") {
		t.Fatalf("retained prompts = %v", got)
	}
	if got, _ := srv.stateStore.HistoryRevision(id); got != 1 {
		t.Fatalf("history revision = %d", got)
	}
	// The replacement went through the ordinary receipt under the edit's id.
	if receipt, err := srv.stateStore.AttachmentReceipt(id, "edit-1"); err != nil || receipt.Delivery != "submitted" {
		t.Fatalf("receipt = %+v %v", receipt, err)
	}

	// Repeating the request returns the same operation; another payload under
	// its id conflicts, and a stale revision refuses without effect.
	if resp, raw := post(t, base+"/api/sessions/"+id+"/edit-last", body); resp.StatusCode != http.StatusAccepted || !strings.Contains(string(raw), `"phase":"committed"`) {
		t.Fatalf("replay = %d %s", resp.StatusCode, raw)
	}
	body["text"] = "other"
	if resp, raw := post(t, base+"/api/sessions/"+id+"/edit-last", body); resp.StatusCode != http.StatusConflict || !strings.Contains(string(raw), runtime.CodeRequestConflict) {
		t.Fatalf("conflicting replay = %d %s", resp.StatusCode, raw)
	}
	latest, _ = lastRootPrompt(t, readTranscriptAPI(t, base, id))
	stale := map[string]any{"request_id": "edit-2", "message_seq": latest.Seq, "expected_revision": 0, "text": "again"}
	if resp, raw := post(t, base+"/api/sessions/"+id+"/edit-last", stale); resp.StatusCode != http.StatusConflict || !strings.Contains(string(raw), runtime.CodeEditConflict) {
		t.Fatalf("stale revision = %d %s", resp.StatusCode, raw)
	}
	statusResp, err := http.Get(base + "/api/sessions/" + id + "/edit-last/edit-1")
	if err != nil {
		t.Fatal(err)
	}
	defer statusResp.Body.Close()
	var status editResult
	_ = json.NewDecoder(statusResp.Body).Decode(&status)
	if status.Phase != state.EditCommitted || status.Active != "revised" || status.Delivery != "sent" || status.Text != "third, revised" {
		t.Fatalf("status = %+v", status)
	}
}

// An adapter without the complete-turn proof cannot edit a later message, and
// the refusal changes nothing (TS-04.R113).
func TestEditLastRefusesWithoutCompleteTurnProof(t *testing.T) {
	srv, base, id, _ := forkServer(t, nil)
	latest, _ := lastRootPrompt(t, readTranscriptAPI(t, base, id))
	resp, raw := post(t, base+"/api/sessions/"+id+"/edit-last", map[string]any{"request_id": "edit-1", "message_seq": latest.Seq, "expected_revision": 0, "text": "x"})
	if resp.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(string(raw), editReasonNoProof) {
		t.Fatalf("edit = %d %s", resp.StatusCode, raw)
	}
	if _, err := srv.stateStore.ReadEdit(id, "edit-1"); err == nil {
		t.Fatal("a refused edit left a record")
	}
}

// A candidate whose proof does not match aborts before the decision: the
// original conversation and provider session stay active and running.
func TestEditLastAbortRestoresTheOriginal(t *testing.T) {
	srv, base, id, _ := forkServer(t, map[string]string{"FAKEACP_COMPLETE_TURN": "1", "FAKEACP_COMPLETE_TURN_WRONG": "1"})
	before := readTranscriptAPI(t, base, id)
	snap, _ := srv.stateStore.ReadSession(id)
	latest, _ := lastRootPrompt(t, before)
	resp, raw := post(t, base+"/api/sessions/"+id+"/edit-last", map[string]any{"request_id": "edit-1", "message_seq": latest.Seq, "expected_revision": 0, "text": "x"})
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("edit = %d %s", resp.StatusCode, raw)
	}
	rec := waitEdit(t, srv, id, "edit-1", settled)
	if rec.Phase != state.EditAborted || !strings.Contains(rec.Reason, "Nothing was changed") {
		t.Fatalf("edit = %+v", rec)
	}
	after, _ := srv.stateStore.ReadSession(id)
	// The original session was reloaded (the fake echoes a load id); the
	// refused candidate fork never became the selection.
	if after.LastSessionID == "fake-fork-1" || after.LastSessionID == "" || snap.LastSessionID == "" {
		t.Fatalf("session = %q after abort (original %q)", after.LastSessionID, snap.LastSessionID)
	}
	if rec.CandidateSession != "" {
		t.Fatalf("aborted edit recorded a candidate: %+v", rec)
	}
	if got := promptTexts(readTranscriptAPI(t, base, id)); got[len(got)-1] != "third" {
		t.Fatalf("prompts after abort = %v", got)
	}
	if !srv.registry.Owns(id) {
		t.Fatal("original runtime was not restarted")
	}
	if rev, _ := srv.stateStore.HistoryRevision(id); rev != 0 {
		t.Fatalf("revision = %d", rev)
	}
}

// The first message edits from a fresh session with the frozen initial
// context; no fork or proof is needed.
func TestEditFirstMessageStartsFreshContext(t *testing.T) {
	srv, ts, id := cloneServer(t, map[string]string{"FAKEACP_CAPS": "1"})
	waitStatus(t, srv, id, "idle")
	deadline := time.Now().Add(5 * time.Second)
	for len(rootTurnEnds(readEvents(t, srv, id))) < 1 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	first, _ := lastRootPrompt(t, readEvents(t, srv, id))
	resp, raw := post(t, ts.URL+"/api/sessions/"+id+"/edit-last", map[string]any{"request_id": "edit-1", "message_seq": first.Seq, "expected_revision": 0, "text": "a better start"})
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("edit = %d %s", resp.StatusCode, raw)
	}
	if rec := waitEdit(t, srv, id, "edit-1", settled); rec.Phase != state.EditCommitted {
		t.Fatalf("edit = %+v", rec)
	}
	if got := promptTexts(readEvents(t, srv, id)); len(got) != 1 || got[0] != "a better start" {
		t.Fatalf("active prompts = %v", got)
	}
}

// Ordinary input is refused before acceptance while an edit owns the
// conversation (TS-01.R51).
func TestSendRefusedWhileEditOwnsTheConversation(t *testing.T) {
	srv, ts, id := cloneServer(t, map[string]string{"FAKEACP_CAPS": "1"})
	waitStatus(t, srv, id, "idle")
	if err := srv.stateStore.CreateEdit(state.EditTransition{AgentID: id, RequestID: "edit-1", PayloadHash: "h", MessageJSON: "{}", MessageSeq: 1}); err != nil {
		t.Fatal(err)
	}
	resp, raw := post(t, ts.URL+"/api/sessions/"+id+"/prompt", map[string]string{"text": "hi", "request_id": "send-1"})
	if resp.StatusCode != http.StatusConflict || !strings.Contains(string(raw), runtime.CodeEditInProgress) {
		t.Fatalf("send = %d %s", resp.StatusCode, raw)
	}
	if _, err := srv.stateStore.AttachmentReceipt(id, "send-1"); err == nil {
		t.Fatal("refused send left a receipt")
	}
	if err := srv.registry.Chat().ReserveEdit(id, "edit-1"); err != nil {
		t.Fatalf("reserve: %v", err)
	}
	defer srv.registry.Chat().ReleaseEdit(id, "edit-1")
	if _, err := srv.registry.SendMessageOrHold(t.Context(), id, runtime.Message{Text: "x", RequestID: "other"}); err == nil || !strings.Contains(err.Error(), "being edited") {
		t.Fatalf("runtime admission = %v", err)
	}
}

// Restart recovery completes a recorded decision exactly once and releases
// the undispatched replacement to explicit recovery (TS-02.R69).
func TestRecoverEditsCompletesCommittingOnce(t *testing.T) {
	srv, _, id := cloneServer(t, map[string]string{"FAKEACP_CAPS": "1"})
	waitStatus(t, srv, id, "idle")
	first, _ := lastRootPrompt(t, readEvents(t, srv, id))
	msg, _ := json.Marshal(runtime.Message{Text: "replacement", RequestID: "edit-1"})
	if err := srv.stateStore.CreateEdit(state.EditTransition{AgentID: id, RequestID: "edit-1", PayloadHash: "h", MessageJSON: string(msg), MessageSeq: first.Seq}); err != nil {
		t.Fatal(err)
	}
	if _, err := srv.stateStore.DecideEdit(id, "edit-1", first.Seq, first.Seq+10, "candidate"); err != nil {
		t.Fatal(err)
	}
	srv.recoverEdits()
	srv.recoverEdits()
	revisions := 0
	for _, ev := range readEventsRaw(t, srv, id) {
		if ev.Type == runtime.EvConversationRevision {
			revisions++
		}
	}
	rec, _ := srv.stateStore.ReadEdit(id, "edit-1")
	if revisions != 1 || rec.Phase != state.EditCommitted || !rec.Dispatched {
		t.Fatalf("revisions=%d record=%+v", revisions, rec)
	}
	if receipt, err := srv.stateStore.AttachmentReceipt(id, "edit-1"); err != nil || receipt.Delivery != "released" {
		t.Fatalf("receipt = %+v %v", receipt, err)
	}
	// EDIT-EDGE-1: with no live candidate to publish it, recovery journals the
	// selected session's record itself.
	var last runtime.SessionMetaData
	metas, err := transcript.ReadFile(srv.configStore.Home(), id, transcript.ReadOptions{Raw: true, IncludeMeta: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, ev := range metas {
		if ev.Type == runtime.EvSessionMeta && ev.Seq > 0 {
			_ = json.Unmarshal(ev.Data, &last)
		}
	}
	if snap, _ := srv.stateStore.ReadSession(id); snap.LastSessionID != "candidate" || last.SessionID != "candidate" {
		t.Fatalf("selected %q, last session record %q", snap.LastSessionID, last.SessionID)
	}
}

func readEventsRaw(t *testing.T, srv *Server, id string) []runtime.Event {
	t.Helper()
	evs, err := transcript.ReadFile(srv.configStore.Home(), id, transcript.ReadOptions{Raw: true})
	if err != nil {
		t.Fatal(err)
	}
	return evs
}

// EDIT-REVIEW-11 (FS-01.R61): a mail-woken turn that completed after the
// last person message is not that message's turn. Edit refuses rather than
// superseding it, and the later reply stays visible.
func TestEditLastRefusesWhenALaterTurnFollowed(t *testing.T) {
	srv, base, id, _ := forkServer(t, map[string]string{"FAKEACP_COMPLETE_TURN": "1"})
	latest, _ := lastRootPrompt(t, readTranscriptAPI(t, base, id))
	insertMail(t, srv, id, "wake")
	pending := pendingActivations(t, srv, id)
	if len(pending) != 1 {
		t.Fatalf("activations = %+v, want one", pending)
	}
	srv.executeMailActivation(context.Background(), pending[0])
	deadline := time.Now().Add(5 * time.Second)
	for len(rootTurnEnds(readEvents(t, srv, id))) < 4 {
		if time.Now().After(deadline) {
			t.Fatal("mail turn did not complete")
		}
		time.Sleep(20 * time.Millisecond)
	}
	waitStatus(t, srv, id, "idle")
	waitTurnReleased(t, srv, id)
	if again, _ := lastRootPrompt(t, readTranscriptAPI(t, base, id)); again.Seq != latest.Seq {
		t.Fatalf("mail turn wrote a person message at %d", again.Seq)
	}
	resp, raw := post(t, base+"/api/sessions/"+id+"/edit-last", map[string]any{"request_id": "edit-1", "message_seq": latest.Seq, "expected_revision": 0, "text": "x"})
	if resp.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(string(raw), editReasonLaterTurn) {
		t.Fatalf("edit = %d %s", resp.StatusCode, raw)
	}
	if got := len(rootTurnEnds(readTranscriptAPI(t, base, id))); got != 4 {
		t.Fatalf("active turn ends after refusal = %d, want 4", got)
	}
}

// EDIT-REVIEW-1 (FS-01.A43, TS-04.R113): after an Edit the kept turn's fork
// point still names the original native session. The committed edit's
// lineage relates it to the replacement, so the replacement message can be
// edited again, forking at the same kept turn of the original session.
func TestEditLastTwiceForksTheKeptTurnAgain(t *testing.T) {
	srv, base, id, metaDump := forkServer(t, map[string]string{"FAKEACP_COMPLETE_TURN": "1"})
	original, _ := srv.stateStore.ReadSession(id)
	edit := func(requestID, text string, revision int64) {
		t.Helper()
		latest, _ := lastRootPrompt(t, readTranscriptAPI(t, base, id))
		_ = os.Remove(metaDump)
		body := map[string]any{"request_id": requestID, "message_seq": latest.Seq, "expected_revision": revision - 1, "text": text}
		if resp, raw := post(t, base+"/api/sessions/"+id+"/edit-last", body); resp.StatusCode != http.StatusAccepted {
			t.Fatalf("%s = %d %s", requestID, resp.StatusCode, raw)
		}
		if rec := waitEdit(t, srv, id, requestID, settled); rec.Phase != state.EditCommitted || rec.Revision != revision {
			t.Fatalf("%s record = %+v", requestID, rec)
		}
		waitStatus(t, srv, id, "idle")
		deadline := time.Now().Add(5 * time.Second)
		for {
			active := readTranscriptAPI(t, base, id)
			if got := promptTexts(active); len(rootTurnEnds(active)) == 3 && got[len(got)-1] == text {
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("%s: active prompts = %v", requestID, promptTexts(active))
			}
			time.Sleep(20 * time.Millisecond)
		}
		waitTurnReleased(t, srv, id)
		if meta, _ := os.ReadFile(metaDump); !strings.Contains(string(meta), `"messageId":"msg-2"`) {
			t.Fatalf("%s did not fork after turn 2: %s", requestID, meta)
		}
	}
	edit("edit-1", "third, revised", 1)
	// The kept second turn's affordance stays available after the first Edit.
	if kept := rootTurnEnds(readTranscriptAPI(t, base, id))[1]; kept.Fork == nil || !kept.Fork.Available {
		t.Fatalf("kept turn end %d after edit: %+v", kept.Seq, kept.Fork)
	}
	edit("edit-2", "third, revised again", 2)
	if lineage, _ := srv.stateStore.EditLineage(id); len(lineage) != 2 || lineage[1].OldSession != original.LastSessionID {
		t.Fatalf("lineage = %+v, want two edits from %s", lineage, original.LastSessionID)
	}
}
