package server

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/state"
)

// TS-10.R65: the largest legal assignment, built from the authoring limits at
// their worst encoding, renders whole and within the derived ceiling; the
// ceiling refuses rather than clips. The fixture is written from the limits
// themselves (16,000-rune instruction, 32 attachments, 64 watched changes, the
// 32-step plan), not from the ceiling's own terms (INV §17).
func TestLargestLegalAssignmentFitsItsCeiling(t *testing.T) {
	escaped := func(n int) string { return strings.Repeat("< ", n/2) } // both escape to 6 bytes
	attachments := make([]map[string]any, 32)
	for i := range attachments {
		attachments[i] = map[string]any{"context_ref_id": strings.Repeat("r", 64), "label": escaped(200), "description": escaped(1000)}
	}
	observed := make([]state.TaskObservedChange, 64)
	for i := range observed {
		observed[i] = state.TaskObservedChange{TaskID: strings.Repeat("t", 64), Revision: 1 << 30, State: state.TaskFinished,
			Outcome: state.OutcomeFailure, Summary: escaped(2000), AttentionReason: escaped(1000), Changed: true}
	}
	steps := make([]state.PipelineStep, 32)
	checks := map[string]string{}
	for i := range steps {
		steps[i] = state.PipelineStep{ID: strings.Repeat("s", 60) + string(rune('A'+i)), Title: escaped(120), ObjectiveSummary: escaped(240)}
		checks[steps[i].ID] = strings.Repeat("t", 64)
	}
	facts := map[string]any{
		"task_id": strings.Repeat("t", 64), "display_name": escaped(200), "execution_handle": strings.Repeat("h", 64),
		"attachments": attachments, "observed_changes": observed,
		"pipeline":          state.PipelineAssignment{RunID: strings.Repeat("p", 64), CurrentStageID: steps[0].ID, Steps: steps, RequiredChecks: checks},
		"pipeline_contract": strings.Repeat("<", 4000),
	}
	started := time.Now()
	task := state.Task{Instruction: strings.Repeat("\U0001F600", 16000), CreatedByKind: "agent", CreatedByAgentID: strings.Repeat("a", 64), StartedAt: &started}
	text, err := renderTaskAssignment(task, facts, false)
	if err != nil {
		t.Fatalf("largest legal assignment refused: %v", err)
	}
	if !strings.Contains(text, task.Instruction) || len(text) > maxAssignmentBytes {
		t.Fatalf("rendered %d bytes against ceiling %d, instruction intact=%v", len(text), maxAssignmentBytes, strings.Contains(text, task.Instruction))
	}
	_, block, _ := strings.Cut(text, "Chuck assignment context:\n")
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal([]byte(block), &decoded); err != nil || len(decoded) != len(facts) {
		t.Fatalf("context block did not survive whole: %v (%d keys)", err, len(decoded))
	}

	task.Instruction = strings.Repeat("x", maxAssignmentBytes)
	if _, err := renderTaskAssignment(task, facts, true); !errors.Is(err, errAssignmentTooLarge) {
		t.Fatalf("over-ceiling assignment = %v, want a refusal", err)
	}
}
