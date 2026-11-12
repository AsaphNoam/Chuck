package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/transcript"
)

// forkServer runs a verified fork-capable fake through three numbered turns.
func forkServer(t *testing.T, env map[string]string) (*Server, string, string, string) {
	t.Helper()
	metaDump := filepath.Join(t.TempDir(), "fork-meta.json")
	all := map[string]string{"FAKEACP_CAPS": "1", "FAKEACP_FORK_META_DUMP": metaDump,
		"FAKEACP_AGENT_NAME": "@agentclientprotocol/claude-agent-acp", "FAKEACP_AGENT_VERSION": "0.85.1",
		"FAKEACP_SCENARIO": "numbered_turns", "FAKEACP_FIRST_TURN_NO_ID": "1"}
	for k, v := range env {
		all[k] = v
	}
	srv, ts, src := cloneServer(t, all)
	for _, text := range []string{"second", "third"} {
		if resp, body := post(t, ts.URL+"/api/sessions/"+src+"/prompt", map[string]string{"text": text}); resp.StatusCode != http.StatusAccepted {
			t.Fatalf("prompt = %d %s", resp.StatusCode, body)
		}
		waitStatus(t, srv, src, "idle")
	}
	return srv, ts.URL, src, metaDump
}

type transcriptRead struct {
	Events []runtime.Event `json:"events"`
}

func readTranscriptAPI(t *testing.T, base, id string) []runtime.Event {
	t.Helper()
	resp, err := http.Get(base + "/api/sessions/" + id + "/transcript")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out transcriptRead
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	return out.Events
}

func rootTurnEnds(events []runtime.Event) []runtime.Event {
	var out []runtime.Event
	for _, ev := range events {
		if ev.Type == runtime.EvTurnEnd && ev.ActivityID == "" {
			out = append(out, ev)
		}
	}
	return out
}

// TestForkFromHereAndEditEndAtTheChosenTurn is FS-01.A40 against the fake:
// the API marks turn ends, Fork from here ends the copy and the provider fork
// at the chosen turn, Edit returns the next message's text, and the stored
// provider identity never reaches clients.
func TestForkFromHereAndEditEndAtTheChosenTurn(t *testing.T) {
	srv, base, src, metaDump := forkServer(t, nil)
	events := readTranscriptAPI(t, base, src)
	ends := rootTurnEnds(events)
	if len(ends) != 3 {
		t.Fatalf("turn ends = %d, want 3", len(ends))
	}
	// Turn 1 streamed without a messageId: no fork point.
	if ends[0].Fork == nil || ends[0].Fork.Available || ends[0].Fork.Reason != runtime.ForkReasonNoForkPoint {
		t.Fatalf("turn 1 affordance = %+v", ends[0].Fork)
	}
	if ends[1].Fork == nil || !ends[1].Fork.Available || ends[2].Fork == nil || !ends[2].Fork.Available {
		t.Fatalf("numbered turn affordances = %+v %+v", ends[1].Fork, ends[2].Fork)
	}
	raw, _ := json.Marshal(events)
	if strings.Contains(string(raw), "msg-2") || strings.Contains(string(raw), "fork_point") || strings.Contains(string(raw), "sha256:") {
		t.Fatalf("transcript read leaks provider fork identity: %s", raw)
	}

	// Fork from here on turn 2 ("msg-2").
	resp, body := post(t, base+"/api/sessions/"+src+"/clone", map[string]int64{"through_seq": ends[1].Seq})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("fork = %d %s", resp.StatusCode, body)
	}
	var out struct {
		Agent struct {
			AgentID string `json:"agent_id"`
		} `json:"agent"`
		ForkedFromSeq int64  `json:"forked_from_seq"`
		ComposeText   string `json:"compose_text"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatal(err)
	}
	if out.ForkedFromSeq != ends[1].Seq || out.ComposeText != "" || strings.Contains(string(body), "msg-") {
		t.Fatalf("fork response = %s", body)
	}
	meta, _ := os.ReadFile(metaDump)
	if !strings.Contains(string(meta), `"messageId":"msg-2"`) || !strings.Contains(string(meta), `"messageFingerprint":"sha256:`) {
		t.Fatalf("fork request _meta = %s", meta)
	}
	copied := readEvents(t, srv, out.Agent.AgentID)
	for _, ev := range copied {
		if ev.Seq > ends[1].Seq && ev.Type != runtime.EvForkBoundary && ev.Type != runtime.EvSessionMeta {
			t.Fatalf("fork copied past the chosen turn: %+v", ev)
		}
		if ev.ForkPoint != nil {
			t.Fatalf("copied event keeps the source's fork point: %+v", ev)
		}
	}
	if got := rootTurnEnds(copied); len(got) != 2 {
		t.Fatalf("fork holds %d turns, want 2", len(got))
	}

	// Edit the third message: ends after turn 2 with that text, unsent.
	var third int64
	for _, ev := range events {
		if ev.Type == runtime.EvUserPrompt && ev.Seq > ends[1].Seq && third == 0 {
			third = ev.Seq
		}
	}
	waitStatus(t, srv, src, "idle")
	resp, body = post(t, base+"/api/sessions/"+src+"/clone", map[string]int64{"through_seq": ends[1].Seq, "compose_from_seq": third})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("edit = %d %s", resp.StatusCode, body)
	}
	if err := json.Unmarshal(body, &out); err != nil || out.ComposeText != "third" {
		t.Fatalf("edit compose_text = %q (%s)", out.ComposeText, body)
	}
	if src := readEvents(t, srv, src); len(rootTurnEnds(src)) != 3 {
		t.Fatalf("source lost turns: %d", len(rootTurnEnds(src)))
	}
}

// TestForkRefusalsLeaveNoPartialAgent: an unmarked turn, a mismatched
// compose message and a failed replay proof each create nothing.
func TestForkRefusalsLeaveNoPartialAgent(t *testing.T) {
	srv, base, src, _ := forkServer(t, map[string]string{"FAKEACP_FORK_WRONG_CUT": "1"})
	ends := rootTurnEnds(readTranscriptAPI(t, base, src))
	before, _ := srv.stateStore.ListAgents()
	cases := []struct {
		name string
		body map[string]int64
		want int
	}{
		{"no fork point", map[string]int64{"through_seq": ends[0].Seq}, http.StatusUnprocessableEntity},
		{"not a turn end", map[string]int64{"through_seq": ends[1].Seq - 1}, http.StatusUnprocessableEntity},
		{"compose without boundary", map[string]int64{"compose_from_seq": 3}, http.StatusUnprocessableEntity},
		{"compose not the next message", map[string]int64{"through_seq": ends[1].Seq, "compose_from_seq": ends[2].Seq}, http.StatusUnprocessableEntity},
		{"replay proof fails", map[string]int64{"through_seq": ends[1].Seq}, http.StatusUnprocessableEntity},
	}
	for _, tc := range cases {
		resp, body := post(t, base+"/api/sessions/"+src+"/clone", tc.body)
		if resp.StatusCode != tc.want {
			t.Fatalf("%s = %d %s", tc.name, resp.StatusCode, body)
		}
		waitStatus(t, srv, src, "idle")
	}
	if after, _ := srv.stateStore.ListAgents(); len(after) != len(before) {
		t.Fatalf("refused forks left agents: %d -> %d", len(before), len(after))
	}
	if evs, _ := transcript.ReadFile(srv.configStore.Home(), src, transcript.ReadOptions{}); len(rootTurnEnds(evs)) != 3 {
		t.Fatal("refusals changed the source")
	}
}

func TestForkAffordanceReasons(t *testing.T) {
	ended := json.RawMessage(`{"stop_reason":"end_turn"}`)
	point := func(session, name, version string) *runtime.ForkPoint {
		return &runtime.ForkPoint{SessionID: session, MessageID: "m", Fingerprint: "sha256:x", AdapterName: name, AdapterVersion: version}
	}
	codex := point("s1", "@agentclientprotocol/codex-acp", "2.1.1")
	cases := []struct {
		name string
		ev   runtime.Event
		want string
	}{
		{"verified codex", runtime.Event{Type: runtime.EvTurnEnd, Data: ended, ForkPoint: codex}, ""},
		{"cancelled turn", runtime.Event{Type: runtime.EvTurnEnd, Data: json.RawMessage(`{"stop_reason":"cancelled"}`), ForkPoint: codex}, runtime.ForkReasonNotTurnBoundary},
		{"switched session", runtime.Event{Type: runtime.EvTurnEnd, Data: ended, ForkPoint: point("s0", "@agentclientprotocol/codex-acp", "2.1.1")}, runtime.ForkReasonOtherSession},
		{"unverified version", runtime.Event{Type: runtime.EvTurnEnd, Data: ended, ForkPoint: point("s1", "@agentclientprotocol/codex-acp", "2.2.0")}, runtime.ForkReasonUnverifiedAdapter},
		{"no fork point", runtime.Event{Type: runtime.EvTurnEnd, Data: ended}, runtime.ForkReasonNoForkPoint},
	}
	for _, tc := range cases {
		got := forkAffordance(tc.ev, "s1")
		if got == nil || got.Available != (tc.want == "") || got.Reason != tc.want {
			t.Fatalf("%s: affordance = %+v, want reason %q", tc.name, got, tc.want)
		}
	}
	if forkAffordance(runtime.Event{Type: runtime.EvTurnEnd, Data: ended, ActivityID: "child"}, "s1") != nil {
		t.Fatal("a child turn end is marked")
	}
}
