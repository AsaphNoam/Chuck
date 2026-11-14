package state

import (
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
