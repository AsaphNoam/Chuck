package server

import (
	"context"
	"strings"
	"testing"

	"github.com/AsaphNoam/Chuck/internal/runtime"
)

// FS-01.A8: regression reproductions for the 2026-10-14 model/provider continuity
// investigation in HANDOFF.md. Each assertion is a conversation fact the
// receiving provider needs, independently of the primer's implementation.
func TestHistoryPrimerConversationContinuityInvestigation(t *testing.T) {
	t.Run("accepted user instructions", func(t *testing.T) {
		const instruction = "Use the staging database and never deploy to production."
		events := []runtime.Event{
			primerEvent(t, runtime.EvUserPrompt, runtime.UserPromptData{Text: instruction}),
			primerEvent(t, runtime.EvAssistantText, runtime.AssistantTextData{Delta: "Understood."}),
			primerEvent(t, runtime.EvTurnEnd, runtime.TurnEndData{StopReason: "end_turn"}),
		}
		got, err := synthesizeHistoryPrimer(context.Background(), events, runtime.LaunchSpec{}, 8000, defaultPrimerSummarizer)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(got, instruction) {
			t.Fatalf("receiving provider lost the user's instruction: %q", got)
		}
	})
	t.Run("tool evidence and edits", func(t *testing.T) {
		events := []runtime.Event{
			primerEvent(t, runtime.EvToolCall, runtime.ToolCallData{Name: "Read", Title: "Read config", Args: []byte(`{"path":"staging-only.json"}`)}),
			primerEvent(t, runtime.EvToolResult, runtime.ToolResultData{Status: "completed", Content: []byte(`"schema_version=42"`)}),
			primerEvent(t, runtime.EvDiff, runtime.DiffData{Path: "config.json", NewText: "deployment_enabled=false"}),
			primerEvent(t, runtime.EvTurnEnd, runtime.TurnEndData{StopReason: "end_turn"}),
		}
		got, err := synthesizeHistoryPrimer(context.Background(), events, runtime.LaunchSpec{}, 8000, defaultPrimerSummarizer)
		if err != nil {
			t.Fatal(err)
		}
		for _, fact := range []string{"staging-only.json", "schema_version=42", "deployment_enabled=false"} {
			if !strings.Contains(got, fact) {
				t.Errorf("receiving provider lost %q: %q", fact, got)
			}
		}
	})
	t.Run("latest turn survives the bound", func(t *testing.T) {
		const latest = "LATEST_STATE: migration failed; rollback is next."
		var events []runtime.Event
		for i := 0; i < 7; i++ {
			events = append(events,
				primerEvent(t, runtime.EvAssistantText, runtime.AssistantTextData{Delta: strings.Repeat("historical details ", 1000)}),
				primerEvent(t, runtime.EvTurnEnd, runtime.TurnEndData{StopReason: "end_turn"}),
			)
		}
		events = append(events,
			primerEvent(t, runtime.EvAssistantText, runtime.AssistantTextData{Delta: latest}),
			primerEvent(t, runtime.EvTurnEnd, runtime.TurnEndData{StopReason: "end_turn"}),
		)
		got, err := synthesizeHistoryPrimer(context.Background(), events, runtime.LaunchSpec{}, 8000, defaultPrimerSummarizer)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(got, latest) {
			t.Fatal("receiving provider lost the latest state while retaining historical details")
		}
	})
	t.Run("streamed text retains its meaning", func(t *testing.T) {
		events := []runtime.Event{
			primerEvent(t, runtime.EvAssistantText, runtime.AssistantTextData{Delta: "The selected file is "}),
			primerEvent(t, runtime.EvAssistantText, runtime.AssistantTextData{Delta: "staging-"}),
			primerEvent(t, runtime.EvAssistantText, runtime.AssistantTextData{Delta: "only.json."}),
			primerEvent(t, runtime.EvTurnEnd, runtime.TurnEndData{StopReason: "end_turn"}),
		}
		got, err := synthesizeHistoryPrimer(context.Background(), events, runtime.LaunchSpec{}, 8000, defaultPrimerSummarizer)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(got, "The selected file is staging-only.json.") {
			t.Fatalf("receiving provider lost the streamed sentence: %q", got)
		}
	})
}
