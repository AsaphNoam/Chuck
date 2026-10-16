package messaging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/AsaphNoam/Chuck/internal/contextref"
	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
)

func historyAgent(t *testing.T, st *state.Store, id, project string, created time.Time, archived bool) {
	t.Helper()
	if err := st.WriteAgent(state.Agent{
		AgentID: id, Name: "Name " + id, Role: "worker", Project: project, Backend: "claude",
		Model: "sonnet", Interface: "chat", CreatedAt: created, Archived: archived,
	}); err != nil {
		t.Fatalf("WriteAgent %s: %v", id, err)
	}
}

func countSideEffects(t *testing.T, st *state.Store) (messages, activations, grants int) {
	t.Helper()
	if err := st.DB().QueryRow(`SELECT (SELECT COUNT(*) FROM messages), (SELECT COUNT(*) FROM activations),
		(SELECT COUNT(*) FROM context_grants)`).Scan(&messages, &activations, &grants); err != nil {
		t.Fatalf("count side effects: %v", err)
	}
	return
}

// FS-06.A28: a chat caller reads another project's archived, stopped history
// with attribution and no grant, mail, activation or transcript side effects.
func TestReadAgentTranscriptCrossProjectArchived(t *testing.T) {
	f := newContextFixture(t)
	liveAgent(t, f.store, "a_caller", "Caller", "lead", "alpha")
	historyAgent(t, f.store, "a_old", "beta", time.Now().Add(-time.Hour).UTC(), true)
	f.transcriptEvents(t, "a_old",
		contextEvent(t, runtime.EvUserPrompt, runtime.UserPromptData{Text: "plan the schema"}),
		contextEvent(t, runtime.EvAssistantText, runtime.AssistantTextData{Delta: "use "}),
		contextEvent(t, runtime.EvAssistantText, runtime.AssistantTextData{Delta: "two tables"}),
		contextEvent(t, runtime.EvTurnEnd, runtime.TurnEndData{StopReason: "end_turn"}),
	)
	before, _ := os.ReadFile(filepath.Join(f.home, "sessions", "a_old", "transcript.ndjson"))
	f.srv.RegisterSession("tok", "a_caller", "gen")
	cs := connect(t, f.srv, "tok")

	res, isErr := call(t, cs, "read_agent_transcript", map[string]any{"agent_id": "a_old"})
	if isErr || res["ok"] != true || res["complete"] != true || res["next_cursor"] != nil {
		t.Fatalf("read = %v", res)
	}
	text := res["text"].(string)
	for _, want := range []string{"user:\nplan the schema", "assistant:\nuse two tables", "turn end (end_turn)"} {
		if !strings.Contains(text, want) {
			t.Errorf("text missing %q:\n%s", want, text)
		}
	}
	source := res["source"].(map[string]any)
	if source["agent_id"] != "a_old" || source["project"] != "beta" || source["archived"] != true || source["running"] != false {
		t.Fatalf("source = %v", source)
	}
	if res["snapshot_last_seq"].(float64) != 4 {
		t.Fatalf("snapshot_last_seq = %v", res["snapshot_last_seq"])
	}
	if m, a, g := countSideEffects(t, f.store); m+a+g != 0 {
		t.Fatalf("read produced messages=%d activations=%d grants=%d", m, a, g)
	}
	after, _ := os.ReadFile(filepath.Join(f.home, "sessions", "a_old", "transcript.ndjson"))
	if string(before) != string(after) {
		t.Fatal("read changed the target transcript")
	}

	// A fresh server over the same store and home reads the same history.
	restarted := New(f.store, nil)
	restarted.SetContextService(contextref.New(f.store, f.home))
	restarted.RegisterSession("tok2", "a_caller", "gen2")
	again, isErr := call(t, connect(t, restarted, "tok2"), "read_agent_transcript", map[string]any{"agent_id": "a_old"})
	if isErr || again["text"] != text {
		t.Fatalf("read after restart = %v", again)
	}
}

// FS-06.A28: pages are bounded, valid UTF-8 and fixed at the first page's
// boundary while the source keeps growing; since_seq then reads only newer
// activity, and a truncated prefix is reported rather than remapped.
func TestReadAgentTranscriptPagesAFixedBoundary(t *testing.T) {
	f := newContextFixture(t)
	liveAgent(t, f.store, "a_caller", "Caller", "lead", "alpha")
	liveAgent(t, f.store, "a_peer", "Peer", "worker", "alpha")
	var events []runtime.Event
	for i := 0; i < 40; i++ {
		events = append(events,
			contextEvent(t, runtime.EvUserPrompt, runtime.UserPromptData{Text: strings.Repeat("é", 700)}),
			contextEvent(t, runtime.EvAssistantText, runtime.AssistantTextData{Delta: strings.Repeat("ü", 500)}),
			contextEvent(t, runtime.EvTurnEnd, runtime.TurnEndData{StopReason: "end_turn"}))
	}
	f.transcriptEvents(t, "a_peer", events...)
	f.srv.RegisterSession("tok", "a_caller", "gen")
	cs := connect(t, f.srv, "tok")

	page, isErr := call(t, cs, "read_agent_transcript", map[string]any{"agent_id": "a_peer"})
	if isErr || page["complete"] != false {
		t.Fatalf("first page = %v", page)
	}
	snapshot := page["snapshot_last_seq"].(float64)
	f.transcriptEvents(t, "a_peer", contextEvent(t, runtime.EvUserPrompt, runtime.UserPromptData{Text: "LATER-ACTIVITY"}))

	var all strings.Builder
	pages := 0
	for {
		text := page["text"].(string)
		if len(text) > contextref.MaxPageBytes || !utf8.ValidString(text) {
			t.Fatalf("page %d: %d bytes, valid=%v", pages, len(text), utf8.ValidString(text))
		}
		if page["snapshot_last_seq"].(float64) != snapshot {
			t.Fatalf("page %d moved the boundary: %v", pages, page["snapshot_last_seq"])
		}
		all.WriteString(text)
		pages++
		if page["complete"] == true {
			break
		}
		cursor := page["next_cursor"].(string)
		if len(cursor) > contextref.MaxCursorBytes {
			t.Fatalf("cursor is %d bytes", len(cursor))
		}
		page, isErr = call(t, cs, "read_agent_transcript", map[string]any{"agent_id": "a_peer", "cursor": cursor})
		if isErr {
			t.Fatalf("page %d: %v", pages, page)
		}
	}
	if pages < 2 || strings.Contains(all.String(), "LATER-ACTIVITY") {
		t.Fatalf("pages=%d, boundary leaked later activity=%v", pages, strings.Contains(all.String(), "LATER-ACTIVITY"))
	}
	if got := strings.Count(all.String(), "user:\n"); got != 40 {
		t.Fatalf("paged read has %d prompts, want 40", got)
	}

	newer, isErr := call(t, cs, "read_agent_transcript", map[string]any{"agent_id": "a_peer", "since_seq": snapshot})
	if isErr || newer["text"] != "user:\nLATER-ACTIVITY\n\n" || newer["complete"] != true {
		t.Fatalf("since_seq read = %v", newer)
	}

	// Truncating the source invalidates an outstanding cursor's prefix.
	first, _ := call(t, cs, "read_agent_transcript", map[string]any{"agent_id": "a_peer"})
	if err := os.Truncate(filepath.Join(f.home, "sessions", "a_peer", "transcript.ndjson"), 10); err != nil {
		t.Fatal(err)
	}
	gone, isErr := call(t, cs, "read_agent_transcript", map[string]any{"agent_id": "a_peer", "cursor": first["next_cursor"]})
	if !isErr || gone["error"] != "transcript_unavailable" {
		t.Fatalf("truncated prefix = %v", gone)
	}
}

// FS-06.A28, FS-17.A13: empty, unknown, failed and malformed reads each have
// their own truthful outcome and retry class.
func TestReadAgentTranscriptOutcomes(t *testing.T) {
	f := newContextFixture(t)
	liveAgent(t, f.store, "a_caller", "Caller", "lead", "alpha")
	historyAgent(t, f.store, "a_empty", "alpha", time.Now().UTC(), false)
	historyAgent(t, f.store, "a_broken", "alpha", time.Now().UTC(), false)
	if err := os.MkdirAll(filepath.Join(f.home, "sessions", "a_broken", "transcript.ndjson"), 0o700); err != nil {
		t.Fatal(err)
	}
	f.srv.RegisterSession("tok", "a_caller", "gen")
	cs := connect(t, f.srv, "tok")

	empty, isErr := call(t, cs, "read_agent_transcript", map[string]any{"agent_id": "a_empty"})
	if isErr || empty["text"] != "" || empty["complete"] != true {
		t.Fatalf("empty = %v", empty)
	}
	cases := []struct {
		args        map[string]any
		code, class string
	}{
		{map[string]any{"agent_id": "a_missing"}, "agent_not_found", "after_change"},
		{map[string]any{"agent_id": "a_broken"}, "transcript_unavailable", "transient"},
		{map[string]any{"agent_id": "a_empty", "cursor": "!!"}, "invalid_cursor", "never"},
		{map[string]any{"agent_id": "a_empty", "cursor": "abc", "since_seq": 2}, "validation", "never"},
		{map[string]any{"agent_id": "a_empty", "since_seq": -1}, "validation", "never"},
	}
	for _, c := range cases {
		res, isErr := call(t, cs, "read_agent_transcript", c.args)
		retry, _ := res["retry"].(map[string]any)
		if !isErr || res["error"] != c.code || retry["class"] != c.class {
			t.Errorf("%v = %v, want %s/%s", c.args, res, c.code, c.class)
		}
	}
}

// FS-06.A32: history discovery pages durable identities across projects and
// archives, while the default directory and mail addressability are unchanged.
func TestListAgentsHistoryScope(t *testing.T) {
	f := newContextFixture(t)
	base := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	liveAgent(t, f.store, "a_caller", "Caller", "lead", "alpha")
	historyAgent(t, f.store, "a_one", "alpha", base, false)
	historyAgent(t, f.store, "a_two", "beta", base.Add(time.Minute), true)
	historyAgent(t, f.store, "a_three", "beta", base.Add(2*time.Minute), false)
	f.srv.RegisterSession("tok", "a_caller", "gen")
	cs := connect(t, f.srv, "tok")

	def, _ := call(t, cs, "list_agents", nil)
	for _, a := range def["agents"].([]any) {
		if a.(map[string]any)["agent_id"] == "a_two" {
			t.Fatal("default directory lists an archived agent")
		}
	}

	var got []string
	var cursor any
	for {
		args := map[string]any{"scope": "history", "limit": 1}
		if cursor != nil {
			args["cursor"] = cursor
		}
		res, isErr := call(t, cs, "list_agents", args)
		if isErr {
			t.Fatalf("history page = %v", res)
		}
		for _, a := range res["agents"].([]any) {
			got = append(got, a.(map[string]any)["agent_id"].(string))
		}
		if cursor = res["next_cursor"]; cursor == nil {
			break
		}
	}
	if strings.Join(got, ",") != "a_three,a_two,a_one" {
		t.Fatalf("history = %v", got)
	}

	beta, _ := call(t, cs, "list_agents", map[string]any{"scope": "history", "project": "beta", "limit": 1})
	refused, isErr := call(t, cs, "list_agents", map[string]any{"scope": "history", "project": "alpha", "cursor": beta["next_cursor"]})
	if !isErr || refused["error"] != "invalid_cursor" {
		t.Fatalf("cursor replayed across filters = %v", refused)
	}
	for _, args := range []map[string]any{{"project": "beta"}, {"scope": "everything"}, {"scope": "history", "limit": 51}} {
		res, isErr := call(t, cs, "list_agents", args)
		if !isErr || res["error"] != "validation" {
			t.Errorf("%v = %v", args, res)
		}
	}
	if _, _, err := state.ResolveRecipient(mustAddressable(t, f.store), "a_two"); err == nil {
		t.Fatal("an archived history entry became a mail recipient")
	}
}

func mustAddressable(t *testing.T, st *state.Store) []state.LiveAgent {
	t.Helper()
	agents, err := st.AddressableAgents()
	if err != nil {
		t.Fatal(err)
	}
	return agents
}
