package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/transcript"
)

// OUTCOME-3 (FS-03.R74/R117, TS-01.R42, INV §2/§11/§20): one stored turn end
// reads as the same ending on the live stream, the whole-transcript read and
// the windowed read every phone uses, while its stored payload is untouched.
// The expected labels are the requirement's enumeration, not the decoder's.
func TestTurnEndOutcomeAgreesOnLiveAndRetainedPaths(t *testing.T) {
	cases := []struct {
		stored string // the turn_end payload as transcripts keep it
		want   string // the outcome every client surface must be given
	}{
		{`{"stop_reason":"end_turn","context_pct":0}`, "completed"},
		{`{"stop_reason":"cancelled","context_pct":0}`, "cancelled"},
		{`{"stop_reason":"max_tokens","context_pct":0}`, "provider_truncated"},
		{`{"stop_reason":"max_turn_requests","context_pct":0}`, "request_limit"},
		{`{"stop_reason":"refusal","context_pct":0}`, "refused"},
		{`{"stop_reason":"error","context_pct":0}`, "failed"},
		{`{"stop_reason":"error","context_pct":0,"process_ended":true}`, "process_ended"},
		{`{"stop_reason":"error","context_pct":0,"quota":{"failure_id":"f","revision":1}}`, "quota_blocked"},
		// A retained end_turn that carries quota evidence is still a quota stop.
		{`{"stop_reason":"end_turn","context_pct":0,"quota":{"failure_id":"f","revision":1}}`, "quota_blocked"},
		{`{"stop_reason":"future_reason","context_pct":0}`, ""},
	}
	srv := testServer(t, true)
	id := seedHookAgent(t, srv)
	w, err := transcript.Open(srv.configStore.Home(), id, nil)
	if err != nil {
		t.Fatal(err)
	}
	var stored []runtime.Event
	for _, tc := range cases {
		ev := runtime.Event{AgentID: id, Type: runtime.EvTurnEnd, Seq: w.NextSeq(), Ts: "2026-11-15T00:00:00Z", Data: json.RawMessage(tc.stored)}
		if err := w.Append(ev); err != nil {
			t.Fatal(err)
		}
		stored = append(stored, ev)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	check := func(path string, got []runtime.Event) {
		t.Helper()
		if len(got) != len(cases) {
			t.Fatalf("%s: %d turn ends, want %d", path, len(got), len(cases))
		}
		for i, tc := range cases {
			if got[i].Outcome != tc.want {
				t.Errorf("%s: %s read as outcome %q, want %q", path, tc.stored, got[i].Outcome, tc.want)
			}
			var gotData, wantData map[string]any
			if json.Unmarshal(got[i].Data, &gotData) != nil || json.Unmarshal([]byte(tc.stored), &wantData) != nil ||
				gotData["stop_reason"] != wantData["stop_reason"] {
				t.Errorf("%s: stored label changed: %s -> %s", path, tc.stored, got[i].Data)
			}
		}
	}
	read := func(query string) []runtime.Event {
		t.Helper()
		rec := httptest.NewRecorder()
		srv.routes().ServeHTTP(rec, newLocalRequest(http.MethodGet, "/api/sessions/"+id+"/transcript"+query, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("transcript%s = %d %s", query, rec.Code, rec.Body.String())
		}
		var body struct {
			Events []runtime.Event `json:"events"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		return rootTurnEnds(body.Events)
	}
	check("whole transcript", read(""))
	check("windowed transcript", read("?limit=200"))

	// Live: what the bus sends, through the client's own JSON decode.
	ch, unsub := srv.eventBus.Subscribe()
	defer unsub()
	for _, ev := range stored {
		srv.publishHistory(ev)
	}
	var live []runtime.Event
	deadline := time.After(5 * time.Second)
	for len(live) < len(cases) {
		select {
		case msg := <-ch:
			if msg.Type != "new_message" {
				continue
			}
			raw, err := json.Marshal(msg.Data)
			if err != nil {
				t.Fatal(err)
			}
			var ev runtime.Event
			if err := json.Unmarshal(raw, &ev); err != nil {
				t.Fatal(err)
			}
			live = append(live, ev)
		case <-deadline:
			t.Fatalf("live stream delivered %d of %d turn ends", len(live), len(cases))
		}
	}
	check("live stream", live)

	// The decoded outcome is a read projection: the file keeps only its labels.
	retained, err := transcript.ReadFile(srv.configStore.Home(), id, transcript.ReadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, ev := range rootTurnEnds(retained) {
		if ev.Outcome != "" {
			t.Fatalf("stored transcript carries a decoded outcome: %+v", ev)
		}
	}
}
