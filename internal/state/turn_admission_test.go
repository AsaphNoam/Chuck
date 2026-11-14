package state

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestTurnAdmissionPreservesOriginRules(t *testing.T) {
	s, _ := newTestStore(t)
	for _, tc := range []struct {
		name    string
		check   TurnAdmission
		allowed bool
	}{
		{"mail before claim", AdmitMailAssignment, false},
		{"mail after claim", AdmitMailAssignment, false},
		{"mail under lifecycle claim", AdmitMailAssignment, false},
		{"mail quota gate", AdmitWithQuotaGate, false},
		{"task quota gate", AdmitWithQuotaGate, false},
		{"task room gate", AdmitTaskRoom, true},
		{"quota owner preflight", AdmitQuotaOwner, true},
		{"quota unassigned", AdmitQuotaUnassigned, true},
		{"room before start", AdmitRoomAssignment, true},
		{"room in start transaction", AdmitRoomAssignment, true},
		{"runtime quota gate", AdmitWithQuotaGate, false},
		{"owner nudge quota gate", AdmitWithQuotaGate, false},
		{"task restart ignores an ordinary quota intent", AdmitTaskRestart, true},
		{"room restart ignores an ordinary quota intent", AdmitRoomRestart, true},
		{"runtime admission without a durable turn", AdmitRuntimeTurn, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			due := timeNow().Truncate(time.Second)
			q := quotaFixture(t, s, "quota_"+tc.name, "episode", due)
			// A ready continuation reserves the slot for mail and task admission.
			// Room and quota use only assignments that have taken the runtime.
			task := newTask(t, s, "proj", tc.name)
			if _, err := s.db.Exec(`UPDATE tasks SET state='ready', continuation_pending=1, assigned_agent_id=? WHERE task_id=?`, q.AgentID, task.TaskID); err != nil {
				t.Fatal(err)
			}
			got, err := s.DecideTurnAdmission(TurnAdmissionRequest{AgentID: q.AgentID, Check: tc.check, Quota: q, Budget: 8, Task: Task{
				TaskID: task.TaskID, AssignedAgentID: q.AgentID, AssignedGeneration: q.SourceGeneration,
			}})
			if err != nil || got.Allowed != tc.allowed || got.WakeTask {
				t.Fatalf("admission = %+v, err=%v, want allowed=%t", got, err, tc.allowed)
			}
		})
	}
}

// TestAdmissionPredicatesStayInTheDecision pins where the cross-plane admission
// predicates may be read (TS-01.R47). AdmitReadyTask embeds the room rule in its
// claiming compare-and-swap, and AdmitQuotaTurn revalidates inside quota's own
// write; both are writes that stay with their plane.
func TestAdmissionPredicatesStayInTheDecision(t *testing.T) {
	allowed := map[string][]string{
		"noRunningRoomAttempt":     {"tasks.go", "turn_admission.go"},
		"quotaWorkEligibleTx":      {"turn_admission.go"},
		"quotaContinuationRunning": {"turn_admission.go"},
		"quotaUnassignedTx":        {"quota_admission.go"},
	}
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range allowed {
		use := regexp.MustCompile(`\b` + name + `\b`)
		def := regexp.MustCompile(`(?:func|const) ` + name + `\b`)
		var got []string
		for _, file := range files {
			if strings.HasSuffix(file, "_test.go") {
				continue
			}
			src, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			if len(use.FindAll(src, -1)) > len(def.FindAll(src, -1)) {
				got = append(got, file)
			}
		}
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Errorf("%s is read in %v, want only %v; call DecideTurnAdmission instead", name, got, want)
		}
	}
}
