package server

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
	"github.com/AsaphNoam/Chuck/internal/transcript"
)

func TestHistoryPrimerRespectsBudget(t *testing.T) {
	var events []runtime.Event
	for i := 0; i < 24; i++ {
		events = append(events, primerEvent(t, runtime.EvAssistantText, runtime.AssistantTextData{
			Delta: strings.Repeat("long assistant content ", 20),
		}))
		events = append(events, primerEvent(t, runtime.EvTurnEnd, runtime.TurnEndData{StopReason: "end_turn"}))
	}
	spec := runtime.LaunchSpec{Agent: state.Agent{AgentID: "a_1", Backend: "codex", Model: "gpt-5.5"}, BackendType: "codex-acp", ModelID: "gpt-5.5"}
	got, err := synthesizeHistoryPrimer(context.Background(), events, spec, 80, nil)
	if err != nil {
		t.Fatalf("synthesizeHistoryPrimer: %v", err)
	}
	if tokenEstimate(got) > 80 {
		t.Fatalf("primer estimate = %d, want <= 80\n%s", tokenEstimate(got), got)
	}
	if !strings.Contains(got, "truncated") {
		t.Fatalf("primer should include truncation marker under tight budget:\n%s", got)
	}
}

func TestHistoryPrimerFallsBackWhenSummaryFails(t *testing.T) {
	events := []runtime.Event{}
	for i := 0; i < primerTailTurns+2; i++ {
		events = append(events, primerEvent(t, runtime.EvAssistantText, runtime.AssistantTextData{Delta: "turn text"}))
		events = append(events, primerEvent(t, runtime.EvTurnEnd, runtime.TurnEndData{StopReason: "end_turn"}))
	}
	spec := runtime.LaunchSpec{Agent: state.Agent{AgentID: "a_1", Backend: "codex", Model: "gpt-5.5"}, BackendType: "codex-acp", ModelID: "gpt-5.5"}
	got, err := synthesizeHistoryPrimer(context.Background(), events, spec, 8000, func(context.Context, primerSummaryRequest) (string, error) {
		return "", errors.New("summary failed")
	})
	if err != nil {
		t.Fatalf("synthesizeHistoryPrimer: %v", err)
	}
	if !strings.Contains(got, "Older context excerpts") || !strings.Contains(got, "Recent turns") || strings.Contains(got, "verbatim") || strings.Contains(got, "summary:") {
		t.Fatalf("primer missing honest fallback excerpt or tail labels:\n%s", got)
	}
}

func TestHistoryPrimerUTF8AndStreaming(t *testing.T) {
	b := newPrimerBuilder(100)
	for i := 0; i < 1000; i++ {
		if err := b.visit(primerEvent(t, runtime.EvAssistantText, runtime.AssistantTextData{Delta: strings.Repeat("世界", 1000)})); err != nil {
			t.Fatal(err)
		}
		if err := b.visit(primerEvent(t, runtime.EvTurnEnd, runtime.TurnEndData{})); err != nil {
			t.Fatal(err)
		}
		if len(b.tail) > 6 || len(b.current) > 400 || len(b.older) > 400 {
			t.Fatal("unbounded retained transcript")
		}
		for _, turn := range b.tail {
			if len(turn) > 400 || !utf8.ValidString(turn) {
				t.Fatal("invalid bounded turn")
			}
		}
	}
	if err := b.visit(primerEvent(t, runtime.EvUserPrompt, runtime.UserPromptData{Text: "LATEST: keep staging"})); err != nil {
		t.Fatal(err)
	}
	got := b.render(context.Background(), runtime.LaunchSpec{}, defaultPrimerSummarizer)
	if !utf8.ValidString(got) || tokenEstimate(got) > 100 || !strings.Contains(got, "LATEST: keep staging") {
		t.Fatalf("invalid newest bounded primer: %q", got)
	}
	for budget := 1; budget < 50; budget++ {
		if got := truncateToBudget(strings.Repeat("界", 100), budget); !utf8.ValidString(got) || tokenEstimate(got) > budget {
			t.Fatalf("invalid UTF-8 prefix for budget %d", budget)
		}
	}
}

func TestHistoryPrimerOmissionsAndRoles(t *testing.T) {
	events := []runtime.Event{
		primerEvent(t, runtime.EvUserPrompt, runtime.UserPromptData{Text: "Use staging"}),
		{Type: runtime.EvToolResult, Data: []byte(`{"status":`)},
		{Type: "future_event"},
		primerEvent(t, runtime.EvAssistantText, runtime.AssistantTextData{Delta: "Acknowledged"}),
	}
	child := primerEvent(t, runtime.EvUserPrompt, runtime.UserPromptData{Text: "PRIVATE_CHILD"})
	child.ActivityID = "child"
	events = append(events, child)
	got, err := synthesizeHistoryPrimer(context.Background(), events, runtime.LaunchSpec{}, 8000, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"user prompt: Use staging", "assistant: Acknowledged", "malformed transcript event omitted", "unknown transcript event omitted"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %q", want, got)
		}
	}
	if strings.Contains(got, "PRIVATE_CHILD") {
		t.Fatal("child activity leaked into primer")
	}
}

func TestHistoryPrimerUsesSummarizerForOlderTurns(t *testing.T) {
	events := []runtime.Event{}
	for i := 0; i < primerTailTurns+2; i++ {
		events = append(events, primerEvent(t, runtime.EvAssistantText, runtime.AssistantTextData{Delta: "turn text"}))
		events = append(events, primerEvent(t, runtime.EvTurnEnd, runtime.TurnEndData{StopReason: "end_turn"}))
	}
	spec := runtime.LaunchSpec{Agent: state.Agent{AgentID: "a_1", Backend: "codex", Model: "gpt-5.5"}, BackendType: "codex-acp", ModelID: "gpt-5.5"}
	called := false
	got, err := synthesizeHistoryPrimer(context.Background(), events, spec, 8000, func(_ context.Context, req primerSummaryRequest) (string, error) {
		called = true
		if req.Target != "codex/gpt-5.5" || req.Backend != "codex-acp" || req.Model != "gpt-5.5" || req.Transcript == "" {
			t.Fatalf("bad summary request: %+v", req)
		}
		return "summary from target model", nil
	})
	if err != nil {
		t.Fatalf("synthesizeHistoryPrimer: %v", err)
	}
	if !called || !strings.Contains(got, "summary from target model") {
		t.Fatalf("summarizer not reflected in primer; called=%v\n%s", called, got)
	}
}

func TestAppendBackendSwitchMarker(t *testing.T) {
	srv := testServer(t, true)
	home := srv.configStore.Home()
	if err := appendBackendSwitchMarker(srv.registry, "a_1", "claude/sonnet", "codex/gpt", time.Unix(123, 0).UTC()); err != nil {
		t.Fatalf("appendBackendSwitchMarker: %v", err)
	}
	events, err := transcript.ReadFile(home, "a_1", transcript.ReadOptions{})
	if err != nil {
		t.Fatalf("readTranscriptEvents: %v", err)
	}
	if len(events) != 1 || events[0].Type != runtime.EvBackendSwitch {
		t.Fatalf("events = %+v, want one backend_switch", events)
	}
	var d runtime.BackendSwitchData
	if err := json.Unmarshal(events[0].Data, &d); err != nil {
		t.Fatalf("marker data: %v", err)
	}
	if d.From != "claude/sonnet" || d.To != "codex/gpt" || d.At == "" {
		t.Fatalf("marker data = %+v", d)
	}
}

func primerEvent(t *testing.T, typ string, data any) runtime.Event {
	t.Helper()
	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal primer event: %v", err)
	}
	return runtime.Event{AgentID: "a_1", Type: typ, Data: raw}
}

func tokenEstimate(s string) int {
	return (len(s) + 3) / 4
}
