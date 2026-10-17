package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
	"github.com/AsaphNoam/Chuck/internal/transcript"
)

// writeAnnotationPair seeds an inactive chat source and a running, idle chat
// recipient — the FS-13.R7 assign-to-another-agent shape.
func writeAnnotationPair(t *testing.T, srv *Server) (state.Agent, state.Agent) {
	t.Helper()
	now := time.Now().UTC()
	source := state.Agent{AgentID: "a_source", Name: "Source", Role: "implementer", Project: "demo", Backend: "claude", Model: "sonnet", Interface: "chat", CreatedAt: now}
	target := state.Agent{AgentID: "a_target", Name: "Target", Role: "reviewer", Project: "demo", Backend: "claude", Model: "sonnet", Interface: "chat", CreatedAt: now}
	for _, agent := range []state.Agent{source, target} {
		if err := srv.stateStore.WriteAgent(agent); err != nil {
			t.Fatalf("WriteAgent %s: %v", agent.AgentID, err)
		}
	}
	if err := srv.stateStore.WriteRunning(state.RunningEntry{AgentID: target.AgentID, PID: 42, Interface: "chat", StartedAt: now}); err != nil {
		t.Fatalf("WriteRunning: %v", err)
	}
	if err := srv.stateStore.WriteStatus(state.Status{AgentID: target.AgentID, State: "idle"}); err != nil {
		t.Fatalf("WriteStatus: %v", err)
	}
	// The source needs an indexed session row: appending its annotation event
	// flushes that event's text into the session's searchable content (FS-13.R10).
	if err := srv.indexer.UpsertSessionMeta(source.AgentID, runtime.SessionMetaData{
		Name: source.Name, Role: source.Role, Project: source.Project, Backend: source.Backend,
		Model: source.Model, Interface: source.Interface, CreatedAt: now.Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("UpsertSessionMeta: %v", err)
	}
	return source, target
}

func TestAnnotationMailBodyClipsOnlyExcerpts(t *testing.T) {
	data := runtime.AnnotationData{
		Annotations: []runtime.Annotation{
			{Seq: 1, Path: "main.go", Side: "new", StartLine: 3, EndLine: 4, Excerpt: strings.Repeat("x", 2000), Instruction: "inspect the first change"},
			{Seq: 2, Excerpt: strings.Repeat("y", 2000), Instruction: "inspect the second change"},
			{Seq: 3, Excerpt: strings.Repeat("z", 2000), Instruction: "inspect the final change"},
			{Seq: 4, Excerpt: strings.Repeat("w", 2000), Instruction: "keep every instruction intact"},
		},
		OverallInstruction: "send a concise review",
		Target:             runtime.AnnotationTarget{Kind: "agent", AgentID: "a_target"},
	}
	body, err := annotationMailBody(data)
	if err != nil {
		t.Fatalf("annotationMailBody: %v", err)
	}
	if len(body) > maxAnnotationMailBytes {
		t.Fatalf("mail body length = %d, want <= %d", len(body), maxAnnotationMailBytes)
	}
	for _, text := range []string{"inspect the first change", "inspect the second change", "inspect the final change", "keep every instruction intact", "send a concise review"} {
		if !strings.Contains(body, text) {
			t.Fatalf("mail body clipped protected text %q", text)
		}
	}
}

func TestValidateAnnotationsClipsLongExcerptAndRejectsEmptyInstruction(t *testing.T) {
	data := runtime.AnnotationData{Annotations: []runtime.Annotation{{Seq: 1, Excerpt: strings.Repeat("x", 2100), Instruction: "check it"}}, Target: runtime.AnnotationTarget{Kind: "self"}}
	if err := validateAnnotations(&data); err != nil {
		t.Fatalf("validateAnnotations: %v", err)
	}
	if got := runeCount(data.Annotations[0].Excerpt); got != maxAnnotationChars {
		t.Fatalf("excerpt runes = %d, want %d", got, maxAnnotationChars)
	}
	data.Annotations[0].Instruction = "  "
	if err := validateAnnotations(&data); err == nil {
		t.Fatal("validateAnnotations accepted an empty instruction")
	}
}

func TestValidateAnnotationsAcceptsFileAnchorsAndRejectsMixedShapes(t *testing.T) {
	valid := runtime.AnnotationData{Annotations: []runtime.Annotation{{AnchorKind: "file", Path: "/tmp/note.md", StartLine: 2, EndLine: 4, Excerpt: "selected", Instruction: "revise"}}}
	if err := validateAnnotations(&valid); err != nil {
		t.Fatalf("valid file anchor: %v", err)
	}
	if block := runtime.FormatAnnotationBlock(valid); !strings.Contains(block, "File /tmp/note.md (lines 2–4)") || strings.Contains(block, "event 0") {
		t.Fatalf("file annotation block = %q", block)
	}
	for _, annotation := range []runtime.Annotation{
		{AnchorKind: "file", Seq: 1, Path: "note.md", Excerpt: "x", Instruction: "y"},
		{AnchorKind: "file", Path: "note.md", Side: "new", Excerpt: "x", Instruction: "y"},
		{AnchorKind: "file", Path: "note.md", StartLine: 2, Excerpt: "x", Instruction: "y"},
		{AnchorKind: "other", Path: "note.md", Excerpt: "x", Instruction: "y"},
	} {
		data := runtime.AnnotationData{Annotations: []runtime.Annotation{annotation}}
		if err := validateAnnotations(&data); err == nil {
			t.Fatalf("accepted mixed file anchor: %+v", annotation)
		}
	}
}

func TestAnnotationPreflightRejectsInvalidBatchWithoutSideEffects(t *testing.T) {
	srv := testServer(t, true)
	source, target := writeAnnotationPair(t, srv)
	path := "/api/sessions/" + source.AgentID + "/annotations/validate"
	for _, tc := range []struct {
		name, body string
		status     int
	}{
		{"empty instruction", `{"annotations":[{"seq":1,"excerpt":"selected","instruction":"  "}],"target":{"kind":"self"}}`, http.StatusUnprocessableEntity},
		{"oversized request", `{"annotations":[{"seq":1,"excerpt":"` + strings.Repeat("x", remoteBodyLimit) + `","instruction":"review"}],"target":{"kind":"self"}}`, http.StatusUnprocessableEntity},
		{"valid", `{"annotations":[{"seq":1,"excerpt":"selected","instruction":"review"}],"target":{"kind":"self"}}`, http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			srv.routes().ServeHTTP(rec, newLocalRequest(http.MethodPost, path, strings.NewReader(tc.body)))
			if rec.Code != tc.status {
				t.Fatalf("preflight = %d: %s", rec.Code, rec.Body.String())
			}
		})
	}
	mail, err := srv.stateStore.ListMessages(target.AgentID, true, 10)
	if err != nil || len(mail) != 0 {
		t.Fatalf("preflight inserted mail: %d, %v", len(mail), err)
	}
	events, err := transcript.ReadFile(srv.configStore.Home(), source.AgentID, transcript.ReadOptions{})
	if err != nil || len(events) != 0 {
		t.Fatalf("preflight appended transcript: %d, %v", len(events), err)
	}
}

// FS-13.A3/A5 and FS-06.A10: an inactive source can assign annotations to a
// running chat agent; the source event is durable/indexed and user mail neither
// impersonates an agent nor creates a turn-budget row.
func TestAnnotationAgentDeliveryPersistsUserMailAndTranscriptEvent(t *testing.T) {
	srv := testServer(t, true)
	source, target := writeAnnotationPair(t, srv)
	body := runtime.AnnotationData{Annotations: []runtime.Annotation{
		{Seq: 4, Path: "main.go", Side: "new", StartLine: 9, EndLine: 10, Excerpt: "target phrase", Instruction: "review this branch"},
		{AnchorKind: "file", Path: "/tmp/notes.md", StartLine: 2, EndLine: 2, Excerpt: "point in time", Instruction: "clarify this"},
	}, Target: runtime.AnnotationTarget{Kind: "agent", AgentID: target.AgentID}}
	raw, _ := json.Marshal(body)
	req := newLocalRequest(http.MethodPost, "/api/sessions/a_source/annotations", bytes.NewReader(raw))
	rec := httptest.NewRecorder()
	srv.routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("annotation status = %d: %s", rec.Code, rec.Body.String())
	}
	mail, err := srv.stateStore.ListMessages(target.AgentID, true, 10)
	if err != nil || len(mail) != 1 {
		t.Fatalf("ListMessages = %d, %v; want 1", len(mail), err)
	}
	if mail[0].FromAgent != "user" || mail[0].FromAddress != "user@dashboard" || !strings.Contains(mail[0].Body, "target phrase") || !strings.Contains(mail[0].Body, "File /tmp/notes.md (lines 2)") {
		t.Fatalf("unexpected reserved-sender mail: %+v", mail[0])
	}
	var budgetRows int
	if err := srv.stateStore.DB().QueryRow(`SELECT COUNT(*) FROM turn_budget WHERE agent_id = ?`, target.AgentID).Scan(&budgetRows); err != nil || budgetRows != 0 {
		t.Fatalf("reserved user mail budget rows = %d, %v; want none", budgetRows, err)
	}
	events, err := transcript.ReadFile(srv.configStore.Home(), source.AgentID, transcript.ReadOptions{})
	if err != nil || len(events) != 1 || events[0].Type != runtime.EvAnnotation {
		t.Fatalf("source annotation event = %#v, %v", events, err)
	}
	var indexed string
	if err := srv.stateStore.DB().QueryRow(`SELECT content FROM sessions_fts WHERE agent_id = ? AND document_id <> 'metadata'`, source.AgentID).Scan(&indexed); err != nil || !strings.Contains(indexed, "review this branch") {
		t.Fatalf("annotation index = %q, %v", indexed, err)
	}
}

// FS-20.R32 and TS-13.R16: a paired phone assigns diff-line annotations through
// the shared FS-13 handler; the tailnet chain rejects any body field the phone
// form does not send before the handler runs.
func TestRemoteAnnotationUsesSharedDelivery(t *testing.T) {
	srv := testServer(t, true)
	source, target := writeAnnotationPair(t, srv)
	h := srv.remoteRoutes(testDomain, testWhoIs(map[string]string{"100.64.0.2:5000": "n"}))
	token := pairTestDevice(t, srv, "d1", "n")
	path := "/api/sessions/" + source.AgentID + "/annotations"

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, phoneRequest(http.MethodPost, path, `{"annotations":[{"seq":4,"excerpt":"x"}],"target":{"kind":"self"},"interface":"terminal"}`, token))
	if rec.Code != http.StatusBadRequest || errorCode(t, rec) != codeRemoteFieldNotAllowed {
		t.Fatalf("extra field = %d %s", rec.Code, rec.Body)
	}
	// The shared validation still owns nested limits (FS-13.R11).
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, phoneRequest(http.MethodPost, path, `{"annotations":[{"seq":4,"excerpt":"x","instruction":" "}],"target":{"kind":"agent","agent_id":"a_target"}}`, token))
	if rec.Code != http.StatusUnprocessableEntity || errorCode(t, rec) != "validation" {
		t.Fatalf("blank instruction = %d %s", rec.Code, rec.Body)
	}

	body := `{"annotations":[{"seq":4,"path":"main.go","side":"new","start_line":9,"end_line":10,"excerpt":"phone phrase","instruction":"tighten this"}],"target":{"kind":"agent","agent_id":"` + target.AgentID + `"}}`
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, phoneRequest(http.MethodPost, path, body, token))
	if rec.Code != http.StatusAccepted {
		t.Fatalf("phone annotation = %d %s", rec.Code, rec.Body)
	}
	mail, err := srv.stateStore.ListMessages(target.AgentID, true, 10)
	if err != nil || len(mail) != 1 || mail[0].FromAgent != "user" || !strings.Contains(mail[0].Body, "phone phrase") {
		t.Fatalf("delivered mail = %+v, %v", mail, err)
	}
	events, err := transcript.ReadFile(srv.configStore.Home(), source.AgentID, transcript.ReadOptions{})
	if err != nil || len(events) != 1 || events[0].Type != runtime.EvAnnotation {
		t.Fatalf("source annotation event = %#v, %v", events, err)
	}
}

// TS-13.R31 / FS-13.A20: phone annotation parity exercises every anchor variant
// and the preflight failures through the authenticated remote chain, comparing
// the actual source event and recipient mail rather than only HTTP status.
func TestRemoteAnnotationParityCoversAnchorsPreflightAndContent(t *testing.T) {
	srv := testServer(t, true)
	source, target := writeAnnotationPair(t, srv)
	h := srv.remoteRoutes(testDomain, testWhoIs(map[string]string{"100.64.0.2:5000": "n"}))
	token := pairTestDevice(t, srv, "d1", "n")
	base := "/api/sessions/" + source.AgentID + "/annotations"
	tgt := `"target":{"kind":"agent","agent_id":"` + target.AgentID + `"}`

	assertNoSideEffects := func() {
		t.Helper()
		mail, err := srv.stateStore.ListMessages(target.AgentID, true, 10)
		if err != nil || len(mail) != 0 {
			t.Fatalf("mail = %d, %v; want none", len(mail), err)
		}
		events, err := transcript.ReadFile(srv.configStore.Home(), source.AgentID, transcript.ReadOptions{})
		if err != nil || len(events) != 0 {
			t.Fatalf("source events = %d, %v; want none", len(events), err)
		}
	}

	for name, annotation := range map[string]string{
		"blank instruction":      `{"seq":1,"excerpt":"x","instruction":" "}`,
		"file anchor with seq":   `{"anchor_kind":"file","seq":1,"path":"note.md","excerpt":"x","instruction":"y"}`,
		"file anchor with side":  `{"anchor_kind":"file","path":"note.md","side":"new","excerpt":"x","instruction":"y"}`,
		"file start without end": `{"anchor_kind":"file","path":"note.md","start_line":2,"excerpt":"x","instruction":"y"}`,
	} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, phoneRequest(http.MethodPost, base+"/validate", `{"annotations":[`+annotation+`],`+tgt+`}`, token))
		if rec.Code != http.StatusUnprocessableEntity || errorCode(t, rec) != "validation" {
			t.Fatalf("%s preflight = %d %s", name, rec.Code, rec.Body)
		}
	}
	assertNoSideEffects()

	annotations := `[
		{"seq":2,"excerpt":"whole event text","instruction":"whole instruction"},
		{"seq":3,"excerpt":"selected excerpt","instruction":"selected instruction"},
		{"seq":4,"path":"main.go","side":"new","start_line":9,"end_line":10,"excerpt":"diff excerpt","instruction":"diff instruction"},
		{"anchor_kind":"file","path":"docs/notes.md","start_line":5,"end_line":7,"excerpt":"file range excerpt","instruction":"file range instruction"},
		{"anchor_kind":"file","path":"docs/rendered.md","excerpt":"rendered capture","instruction":"path only instruction"}
	]`
	body := `{"annotations":` + annotations + `,"overall_instruction":"overall please",` + tgt + `}`

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, phoneRequest(http.MethodPost, base+"/validate", body, token))
	if rec.Code != http.StatusOK {
		t.Fatalf("valid preflight = %d %s", rec.Code, rec.Body)
	}
	assertNoSideEffects()

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, phoneRequest(http.MethodPost, base, body, token))
	if rec.Code != http.StatusAccepted {
		t.Fatalf("phone annotation = %d %s", rec.Code, rec.Body)
	}

	mail, err := srv.stateStore.ListMessages(target.AgentID, true, 10)
	if err != nil || len(mail) != 1 || mail[0].FromAgent != "user" {
		t.Fatalf("delivered mail = %+v, %v", mail, err)
	}
	for _, want := range []string{
		"Transcript event 2", "whole event text", "whole instruction",
		"Transcript event 3", "selected excerpt", "selected instruction",
		"Transcript event 4 — main.go (new lines 9–10)", "diff excerpt", "diff instruction",
		"File docs/notes.md (lines 5–7)", "file range excerpt", "file range instruction",
		"File docs/rendered.md\n", "rendered capture", "path only instruction",
		"Overall instruction: overall please",
	} {
		if !strings.Contains(mail[0].Body, want) {
			t.Fatalf("mail body missing %q:\n%s", want, mail[0].Body)
		}
	}
	if strings.Contains(mail[0].Body, "File docs/rendered.md (") {
		t.Fatalf("path-only file anchor rendered a line range:\n%s", mail[0].Body)
	}

	events, err := transcript.ReadFile(srv.configStore.Home(), source.AgentID, transcript.ReadOptions{})
	if err != nil || len(events) != 1 || events[0].Type != runtime.EvAnnotation {
		t.Fatalf("source annotation event = %#v, %v", events, err)
	}
	var got runtime.AnnotationData
	if err := json.Unmarshal(events[0].Data, &got); err != nil {
		t.Fatalf("decode annotation event: %v", err)
	}
	want := []runtime.Annotation{
		{Seq: 2, Excerpt: "whole event text", Instruction: "whole instruction"},
		{Seq: 3, Excerpt: "selected excerpt", Instruction: "selected instruction"},
		{Seq: 4, Path: "main.go", Side: "new", StartLine: 9, EndLine: 10, Excerpt: "diff excerpt", Instruction: "diff instruction"},
		{AnchorKind: "file", Path: "docs/notes.md", StartLine: 5, EndLine: 7, Excerpt: "file range excerpt", Instruction: "file range instruction"},
		{AnchorKind: "file", Path: "docs/rendered.md", Excerpt: "rendered capture", Instruction: "path only instruction"},
	}
	if !reflect.DeepEqual(got.Annotations, want) || got.OverallInstruction != "overall please" || got.Target.AgentID != target.AgentID {
		t.Fatalf("source event data = %+v", got)
	}
}

// FS-13.R5 and invariant §15: the durable source event precedes delivery. A
// failed append must leave nothing delivered, because the tray is preserved on
// failure (FS-13.R3) and the natural retry would otherwise insert a second copy
// of the same mail for the recipient to act on twice.
func TestAnnotationAppendFailureDeliversNoMailAndRetrySendsOnce(t *testing.T) {
	srv := testServer(t, true)
	source, target := writeAnnotationPair(t, srv)
	// transcript.Open must MkdirAll the per-agent session directory; a regular
	// file at that path fails it the way a full disk or a permission error would.
	blocked := filepath.Join(srv.configStore.Home(), "sessions", source.AgentID)
	if err := os.MkdirAll(filepath.Dir(blocked), 0o700); err != nil {
		t.Fatalf("MkdirAll sessions: %v", err)
	}
	if err := os.WriteFile(blocked, nil, 0o600); err != nil {
		t.Fatalf("block transcript dir: %v", err)
	}

	body := runtime.AnnotationData{Annotations: []runtime.Annotation{{Seq: 4, Excerpt: "target phrase", Instruction: "review this branch"}}, Target: runtime.AnnotationTarget{Kind: "agent", AgentID: target.AgentID}}
	raw, _ := json.Marshal(body)
	send := func() *httptest.ResponseRecorder {
		req := newLocalRequest(http.MethodPost, "/api/sessions/"+source.AgentID+"/annotations", bytes.NewReader(raw))
		rec := httptest.NewRecorder()
		srv.routes().ServeHTTP(rec, req)
		return rec
	}
	mailCount := func() int {
		t.Helper()
		mail, err := srv.stateStore.ListMessages(target.AgentID, false, 10)
		if err != nil {
			t.Fatalf("ListMessages: %v", err)
		}
		return len(mail)
	}

	if rec := send(); rec.Code != http.StatusInternalServerError {
		t.Fatalf("blocked annotation status = %d: %s", rec.Code, rec.Body.String())
	}
	if got := mailCount(); got != 0 {
		t.Fatalf("mail after a failed append = %d; want 0", got)
	}

	// The preserved tray is re-sent once the transcript is writable again.
	if err := os.Remove(blocked); err != nil {
		t.Fatalf("unblock transcript dir: %v", err)
	}
	if rec := send(); rec.Code != http.StatusAccepted {
		t.Fatalf("retried annotation status = %d: %s", rec.Code, rec.Body.String())
	}
	if got := mailCount(); got != 1 {
		t.Fatalf("mail after the retry = %d; want exactly 1", got)
	}
}

// annotationBlockSentinelFromUI reads the one constant the browser borrows from
// FormatAnnotationBlock. Reading it here rather than restating it is the point:
// the client recognizes a self-targeted send's prompt by this prefix alone
// (FS-13.R23), so a change to the Go writer that moves the first line has to
// fail a Go test instead of silently un-quieting every transcript.
func annotationBlockSentinelFromUI(t *testing.T) string {
	t.Helper()
	path := filepath.Join("..", "..", "ui", "src", "lib", "annotations.ts")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	match := regexp.MustCompile(`annotationBlockSentinel\s*=\s*"([^"]+)"`).FindSubmatch(raw)
	if match == nil {
		t.Fatalf("no annotationBlockSentinel declaration in %s", path)
	}
	return string(match[1])
}

// TS-08.R54 — the cross-language pair is pinned, not assumed.
func TestAnnotationBlockStartsWithTheSentinelTheClientMatches(t *testing.T) {
	block := runtime.FormatAnnotationBlock(runtime.AnnotationData{
		Annotations: []runtime.Annotation{{Seq: 4, Excerpt: "target phrase", Instruction: "review this branch"}},
		Target:      runtime.AnnotationTarget{Kind: "self"},
	})
	if sentinel := annotationBlockSentinelFromUI(t); !strings.HasPrefix(block, sentinel) {
		t.Fatalf("annotation block %q does not start with the client's sentinel %q", block, sentinel)
	}
	// FS-13.A15: the pre-rename first line is recognized but never written.
	if strings.Contains(block, "[AgentDeck annotations]") {
		t.Fatalf("annotation block still writes the legacy first line: %q", block)
	}
}

// FS-13.A14 — quieting the duplicate prompt is display-only. The prompt event a
// self-targeted send produces stays in the transcript the API serves, because
// the agent acted on it and search, replay, and export all read from here.
func TestSelfAnnotationTranscriptStillReturnsThePromptEvent(t *testing.T) {
	srv, ts := wakeTestServer(t)
	id := launchAndWaitIdle(t, ts, "impl", "tmpproj")
	waitForStatus(t, srv, id, "idle")

	body := runtime.AnnotationData{
		Annotations: []runtime.Annotation{{Seq: 1, Excerpt: "target phrase", Instruction: "review this branch"}},
		Target:      runtime.AnnotationTarget{Kind: "self"},
	}
	raw, _ := json.Marshal(body)
	resp, respBody := post(t, ts.URL+"/api/sessions/"+id+"/annotations", json.RawMessage(raw))
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("self annotation status = %d: %s", resp.StatusCode, respBody)
	}

	sentinel := annotationBlockSentinelFromUI(t)
	deadline := time.Now().Add(10 * time.Second)
	for {
		events, err := transcript.ReadFile(srv.configStore.Home(), id, transcript.ReadOptions{})
		if err != nil {
			t.Fatalf("ReadFile: %v", err)
		}
		var annotated, prompted bool
		for i, ev := range events {
			if ev.Type != runtime.EvAnnotation {
				continue
			}
			annotated = true
			// The prompt the client hides is the one immediately after the
			// annotation event; the endpoint must still hand it over.
			for _, later := range events[i+1:] {
				if later.Type != runtime.EvUserPrompt {
					continue
				}
				var prompt runtime.UserPromptData
				if err := json.Unmarshal(later.Data, &prompt); err != nil {
					t.Fatalf("decode user prompt: %v", err)
				}
				if strings.HasPrefix(prompt.Text, sentinel) && strings.Contains(prompt.Text, "review this branch") {
					prompted = true
				}
			}
		}
		if annotated && prompted {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("transcript never carried both the annotation event and its prompt: %#v", events)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// TS-02.R38/TS-03.R49/TS-08.R80: a mixed batch's exact fields must survive the
// full HTTP round trip unchanged, and the event the bus publishes live must be
// byte-for-byte the same annotation the archive replays from disk — a card
// drawn live and a card drawn after reload must never disagree (INV §2).
func TestAnnotationRoundTripMatchesLiveAndReplayedEventForMixedAnchors(t *testing.T) {
	srv := testServer(t, true)
	source, target := writeAnnotationPair(t, srv)
	ch, unsub := srv.eventBus.Subscribe()
	defer unsub()

	sent := runtime.AnnotationData{
		Annotations: []runtime.Annotation{
			{Seq: 3, Path: "diff.go", Side: "old", StartLine: 5, EndLine: 6, Excerpt: "transcript excerpt", Instruction: "check the diff"},
			{AnchorKind: "file", Path: "/tmp/anchor.md", StartLine: 10, EndLine: 12, Excerpt: "file excerpt", Instruction: "check the file"},
		},
		OverallInstruction: "review both",
		Target:             runtime.AnnotationTarget{Kind: "agent", AgentID: target.AgentID},
	}
	raw, err := json.Marshal(sent)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	req := newLocalRequest(http.MethodPost, "/api/sessions/"+source.AgentID+"/annotations", bytes.NewReader(raw))
	rec := httptest.NewRecorder()
	srv.routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("annotation status = %d: %s", rec.Code, rec.Body.String())
	}

	var live runtime.AnnotationData
	select {
	case ev := <-ch:
		if ev.Type != "new_message" {
			t.Fatalf("published event type = %q, want new_message", ev.Type)
		}
		runtimeEv, ok := ev.Data.(runtime.Event)
		if !ok || runtimeEv.Type != runtime.EvAnnotation {
			t.Fatalf("published event data = %#v, want a runtime.Event annotation", ev.Data)
		}
		if err := json.Unmarshal(runtimeEv.Data, &live); err != nil {
			t.Fatalf("decode live annotation: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no live annotation event published on the bus")
	}

	events, err := transcript.ReadFile(srv.configStore.Home(), source.AgentID, transcript.ReadOptions{})
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var replayed runtime.AnnotationData
	var found bool
	for _, ev := range events {
		if ev.Type != runtime.EvAnnotation {
			continue
		}
		if err := json.Unmarshal(ev.Data, &replayed); err != nil {
			t.Fatalf("decode replayed annotation: %v", err)
		}
		found = true
	}
	if !found {
		t.Fatalf("no annotation event in replayed transcript: %#v", events)
	}

	want := runtime.AnnotationData{
		Annotations: []runtime.Annotation{
			{Seq: 3, Path: "diff.go", Side: "old", StartLine: 5, EndLine: 6, Excerpt: "transcript excerpt", Instruction: "check the diff"},
			{AnchorKind: "file", Path: "/tmp/anchor.md", StartLine: 10, EndLine: 12, Excerpt: "file excerpt", Instruction: "check the file"},
		},
		OverallInstruction: "review both",
		Target:             runtime.AnnotationTarget{Kind: "agent", AgentID: target.AgentID},
	}
	if !reflect.DeepEqual(live, want) {
		t.Fatalf("live annotation = %#v, want %#v", live, want)
	}
	if !reflect.DeepEqual(replayed, want) {
		t.Fatalf("replayed annotation = %#v, want %#v", replayed, want)
	}
	if !reflect.DeepEqual(live, replayed) {
		t.Fatalf("live annotation %#v disagrees with replayed annotation %#v", live, replayed)
	}
}

// TS-08.R80: a legacy transcript-only annotation (no anchor_kind, the shape
// sent before file anchors existed) must serialize to disk identically to how
// it always did — no anchor_kind/path/side/start_line/end_line fields
// appearing where none were sent — so an old draft renders exactly as before.
func TestLegacyTranscriptOnlyAnnotationSerializesUnchanged(t *testing.T) {
	srv := testServer(t, true)
	source, target := writeAnnotationPair(t, srv)

	legacyBody := `{"annotations":[{"seq":5,"excerpt":"legacy excerpt","instruction":"legacy instruction"}],"target":{"kind":"agent","agent_id":"` + target.AgentID + `"}}`
	req := newLocalRequest(http.MethodPost, "/api/sessions/"+source.AgentID+"/annotations", strings.NewReader(legacyBody))
	rec := httptest.NewRecorder()
	srv.routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("annotation status = %d: %s", rec.Code, rec.Body.String())
	}

	events, err := transcript.ReadFile(srv.configStore.Home(), source.AgentID, transcript.ReadOptions{})
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var raw json.RawMessage
	var found bool
	for _, ev := range events {
		if ev.Type == runtime.EvAnnotation {
			raw = ev.Data
			found = true
		}
	}
	if !found {
		t.Fatalf("no annotation event in transcript: %#v", events)
	}
	want := `{"annotations":[{"seq":5,"excerpt":"legacy excerpt","instruction":"legacy instruction"}],"target":{"kind":"agent","agent_id":"` + target.AgentID + `"}}`
	if string(raw) != want {
		t.Fatalf("legacy annotation serialized as %s, want %s", raw, want)
	}
}
