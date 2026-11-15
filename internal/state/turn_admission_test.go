package state

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// TestTurnAdmissionMatchesPreChangeReads is the independent oracle for TS-01.R47.
// Every row builds its own store and the one state that makes its check differ,
// and its outcome is what the read it replaced answered before commit 7e2539a3
// (the change unit that introduced turn_admission.go; compare 7e2539a3^):
//
//   - mail: server/messaging_loops.go decided from Store.TaskAssignment
//     (tasks.go: state IN starting/running/waiting, or ready with
//     continuation_pending=1). Pending yield or release, starting, and a ready
//     continuation left the mail pending; waiting woke the task; running and
//     no assignment proceeded.
//   - quota gate: server/quota.go quotaBlocksAgent and runtime/chat.go blocked
//     unless no intent, or the intent was continued or cancelled.
//   - room assignment: think_tank_turns.go refused when a task assigned to the
//     agent was starting, running or waiting (a ready continuation did not).
//   - task room: tasks.go claim CAS required no running room attempt for the agent.
//   - quota unassigned: quota_admission.go quotaUnassignedTx returned
//     ErrQuotaOwner for a starting/running/waiting task or a running room attempt.
//   - quota owner: quota_admission.go quotaWorkEligibleTx (moved unchanged):
//     ErrQuotaOwner for a missing agent or unknown owner, otherwise the
//     unassigned rule for an ordinary owner.
//   - task restart: the quota_intents read keyed on the agent, any of the
//     source/continued/manual generations, state != continued, owner task_id.
//   - room restart: quotaContinuationRunningTx, a join of the quota intent
//     (state != continued, continued generation/turn, room owner) with a
//     failed "Quota reached" source attempt of the same agent, room and turn.
//   - runtime turn: runtime/chat.go refused while the in-memory turn was
//     active; the durable form is the agent's running turn for the generation.
func TestTurnAdmissionMatchesPreChangeReads(t *testing.T) {
	const agent = "adm_agent"
	type want struct {
		allowed, wake bool
		err           error
	}
	task := func(t *testing.T, s *Store, set string) Task {
		t.Helper()
		tk := newTask(t, s, "proj", "adm")
		if _, err := s.db.Exec(`UPDATE tasks SET assigned_agent_id=?, `+set+` WHERE task_id=?`, agent, tk.TaskID); err != nil {
			t.Fatal(err)
		}
		return tk
	}
	room := func(t *testing.T, s *Store, roomID string) {
		t.Helper()
		now := formatTime(timeNow())
		if _, err := s.db.Exec(`INSERT INTO think_tanks(room_id,command_id,title,goal,origin_project,openings,phase,control,created_at,updated_at)
VALUES(?,?,'room','goal','proj',0,'discussion','running',?,?)`, roomID, roomID+"-cmd", now, now); err != nil {
			t.Fatal(err)
		}
	}
	attempt := func(t *testing.T, s *Store, id, roomID, agentID, state, failure string) {
		t.Helper()
		if _, err := s.db.Exec(`INSERT INTO think_tank_attempts(attempt_id,room_id,agent_id,turn,token,generation,turn_id,head,checkpoint,delivered_to,state,failure,created_at)
VALUES(?,?,?,'discussion',?,'rg','rt',0,0,0,?,?,?)`, id, roomID, agentID, "tok-"+id, state, failure, formatTime(timeNow())); err != nil {
			t.Fatal(err)
		}
	}
	quota := func(t *testing.T, s *Store, set string, args ...any) QuotaIntent {
		t.Helper()
		due := timeNow().Add(time.Hour).UTC().Truncate(time.Second)
		q := quotaFixture(t, s, agent, "episode", due)
		if set != "" {
			if _, err := s.db.Exec(`UPDATE quota_intents SET `+set+` WHERE agent_id=?`, append(args, agent)...); err != nil {
				t.Fatal(err)
			}
		}
		return q
	}
	owner := func(kind string) string { return `{"kind":"` + kind + `","task_id":"T1"}` }

	for _, tc := range []struct {
		name  string
		check TurnAdmission
		setup func(t *testing.T, s *Store) TurnAdmissionRequest
		want  want
	}{
		// Mail: TaskAssignment.
		{"mail no assignment", AdmitMailAssignment, func(t *testing.T, s *Store) TurnAdmissionRequest { return TurnAdmissionRequest{} }, want{allowed: true}},
		{"mail running task", AdmitMailAssignment, func(t *testing.T, s *Store) TurnAdmissionRequest {
			task(t, s, `state='running'`)
			return TurnAdmissionRequest{}
		}, want{allowed: true}},
		{"mail running task pending yield", AdmitMailAssignment, func(t *testing.T, s *Store) TurnAdmissionRequest {
			task(t, s, `state='running', pending_yield=1`)
			return TurnAdmissionRequest{}
		}, want{}},
		{"mail running task pending release", AdmitMailAssignment, func(t *testing.T, s *Store) TurnAdmissionRequest {
			task(t, s, `state='running', pending_release=1`)
			return TurnAdmissionRequest{}
		}, want{}},
		{"mail waiting task wakes it", AdmitMailAssignment, func(t *testing.T, s *Store) TurnAdmissionRequest {
			task(t, s, `state='waiting'`)
			return TurnAdmissionRequest{}
		}, want{wake: true}},
		{"mail starting task", AdmitMailAssignment, func(t *testing.T, s *Store) TurnAdmissionRequest {
			task(t, s, `state='starting'`)
			return TurnAdmissionRequest{}
		}, want{}},
		{"mail ready continuation", AdmitMailAssignment, func(t *testing.T, s *Store) TurnAdmissionRequest {
			task(t, s, `state='ready', continuation_pending=1`)
			return TurnAdmissionRequest{}
		}, want{}},
		{"mail ready without continuation", AdmitMailAssignment, func(t *testing.T, s *Store) TurnAdmissionRequest {
			task(t, s, `state='ready', continuation_pending=0`)
			return TurnAdmissionRequest{}
		}, want{allowed: true}},
		{"mail finished task", AdmitMailAssignment, func(t *testing.T, s *Store) TurnAdmissionRequest {
			task(t, s, `state='finished'`)
			return TurnAdmissionRequest{}
		}, want{allowed: true}},

		// Quota gate: blocked unless no intent, continued or cancelled.
		{"quota gate no intent", AdmitWithQuotaGate, func(t *testing.T, s *Store) TurnAdmissionRequest { return TurnAdmissionRequest{} }, want{allowed: true}},
		{"quota gate scheduled", AdmitWithQuotaGate, func(t *testing.T, s *Store) TurnAdmissionRequest { quota(t, s, ""); return TurnAdmissionRequest{} }, want{}},
		{"quota gate claimed", AdmitWithQuotaGate, func(t *testing.T, s *Store) TurnAdmissionRequest {
			quota(t, s, `state='claimed'`)
			return TurnAdmissionRequest{}
		}, want{}},
		{"quota gate needs attention", AdmitWithQuotaGate, func(t *testing.T, s *Store) TurnAdmissionRequest {
			quota(t, s, `state='needs_attention'`)
			return TurnAdmissionRequest{}
		}, want{}},
		{"quota gate continued", AdmitWithQuotaGate, func(t *testing.T, s *Store) TurnAdmissionRequest {
			quota(t, s, `state='continued'`)
			return TurnAdmissionRequest{}
		}, want{allowed: true}},
		{"quota gate cancelled", AdmitWithQuotaGate, func(t *testing.T, s *Store) TurnAdmissionRequest {
			quota(t, s, `state='cancelled'`)
			return TurnAdmissionRequest{}
		}, want{allowed: true}},

		// Room assignment: starting/running/waiting tasks only.
		{"room assignment no task", AdmitRoomAssignment, func(t *testing.T, s *Store) TurnAdmissionRequest { return TurnAdmissionRequest{} }, want{allowed: true}},
		{"room assignment running task", AdmitRoomAssignment, func(t *testing.T, s *Store) TurnAdmissionRequest {
			task(t, s, `state='running'`)
			return TurnAdmissionRequest{}
		}, want{}},
		{"room assignment starting task", AdmitRoomAssignment, func(t *testing.T, s *Store) TurnAdmissionRequest {
			task(t, s, `state='starting'`)
			return TurnAdmissionRequest{}
		}, want{}},
		{"room assignment waiting task", AdmitRoomAssignment, func(t *testing.T, s *Store) TurnAdmissionRequest {
			task(t, s, `state='waiting'`)
			return TurnAdmissionRequest{}
		}, want{}},
		{"room assignment ready continuation", AdmitRoomAssignment, func(t *testing.T, s *Store) TurnAdmissionRequest {
			task(t, s, `state='ready', continuation_pending=1`)
			return TurnAdmissionRequest{}
		}, want{allowed: true}},
		{"room assignment finished task", AdmitRoomAssignment, func(t *testing.T, s *Store) TurnAdmissionRequest {
			task(t, s, `state='finished'`)
			return TurnAdmissionRequest{}
		}, want{allowed: true}},

		// Task room: no running room attempt for this agent.
		{"task room no attempt", AdmitTaskRoom, func(t *testing.T, s *Store) TurnAdmissionRequest { return TurnAdmissionRequest{} }, want{allowed: true}},
		{"task room running attempt", AdmitTaskRoom, func(t *testing.T, s *Store) TurnAdmissionRequest {
			room(t, s, "r1")
			attempt(t, s, "a1", "r1", agent, "running", "")
			return TurnAdmissionRequest{}
		}, want{}},
		{"task room failed attempt", AdmitTaskRoom, func(t *testing.T, s *Store) TurnAdmissionRequest {
			room(t, s, "r1")
			attempt(t, s, "a1", "r1", agent, "failed", "")
			return TurnAdmissionRequest{}
		}, want{allowed: true}},
		{"task room other agent running", AdmitTaskRoom, func(t *testing.T, s *Store) TurnAdmissionRequest {
			room(t, s, "r1")
			attempt(t, s, "a1", "r1", "someone_else", "running", "")
			return TurnAdmissionRequest{}
		}, want{allowed: true}},

		// Quota unassigned: quotaUnassignedTx.
		{"quota unassigned free", AdmitQuotaUnassigned, func(t *testing.T, s *Store) TurnAdmissionRequest { return TurnAdmissionRequest{} }, want{allowed: true}},
		{"quota unassigned running task", AdmitQuotaUnassigned, func(t *testing.T, s *Store) TurnAdmissionRequest {
			task(t, s, `state='running'`)
			return TurnAdmissionRequest{}
		}, want{err: ErrQuotaOwner}},
		{"quota unassigned waiting task", AdmitQuotaUnassigned, func(t *testing.T, s *Store) TurnAdmissionRequest {
			task(t, s, `state='waiting'`)
			return TurnAdmissionRequest{}
		}, want{err: ErrQuotaOwner}},
		{"quota unassigned ready continuation", AdmitQuotaUnassigned, func(t *testing.T, s *Store) TurnAdmissionRequest {
			task(t, s, `state='ready', continuation_pending=1`)
			return TurnAdmissionRequest{}
		}, want{allowed: true}},
		{"quota unassigned running room", AdmitQuotaUnassigned, func(t *testing.T, s *Store) TurnAdmissionRequest {
			room(t, s, "r1")
			attempt(t, s, "a1", "r1", agent, "running", "")
			return TurnAdmissionRequest{}
		}, want{err: ErrQuotaOwner}},

		// Quota owner: quotaWorkEligibleTx.
		{"quota owner ordinary free", AdmitQuotaOwner, func(t *testing.T, s *Store) TurnAdmissionRequest {
			return TurnAdmissionRequest{Quota: quota(t, s, ""), Budget: 8}
		}, want{allowed: true}},
		{"quota owner ordinary with running task", AdmitQuotaOwner, func(t *testing.T, s *Store) TurnAdmissionRequest {
			q := quota(t, s, "")
			task(t, s, `state='running'`)
			return TurnAdmissionRequest{Quota: q, Budget: 8}
		}, want{err: ErrQuotaOwner}},
		{"quota owner ordinary with running room", AdmitQuotaOwner, func(t *testing.T, s *Store) TurnAdmissionRequest {
			q := quota(t, s, "")
			room(t, s, "r1")
			attempt(t, s, "a1", "r1", agent, "running", "")
			return TurnAdmissionRequest{Quota: q, Budget: 8}
		}, want{err: ErrQuotaOwner}},
		{"quota owner archived agent", AdmitQuotaOwner, func(t *testing.T, s *Store) TurnAdmissionRequest {
			q := quota(t, s, "")
			if _, err := s.db.Exec(`UPDATE agents SET archived=1 WHERE agent_id=?`, agent); err != nil {
				t.Fatal(err)
			}
			return TurnAdmissionRequest{Quota: q, Budget: 8}
		}, want{err: ErrQuotaOwner}},
		{"quota owner unknown kind", AdmitQuotaOwner, func(t *testing.T, s *Store) TurnAdmissionRequest {
			q := quota(t, s, "")
			q.Owner.Kind = "other"
			return TurnAdmissionRequest{Quota: q, Budget: 8}
		}, want{err: ErrQuotaOwner}},

		// Task restart: a non-continued intent owned by this task and generation.
		{"task restart no intent", AdmitTaskRestart, func(t *testing.T, s *Store) TurnAdmissionRequest {
			return TurnAdmissionRequest{Task: Task{TaskID: "T1", AssignedAgentID: agent, AssignedGeneration: "gen"}}
		}, want{allowed: true}},
		{"task restart ordinary intent", AdmitTaskRestart, func(t *testing.T, s *Store) TurnAdmissionRequest {
			quota(t, s, `source_generation='gen'`)
			return TurnAdmissionRequest{Task: Task{TaskID: "T1", AssignedAgentID: agent, AssignedGeneration: "gen"}}
		}, want{allowed: true}},
		{"task restart held by source generation", AdmitTaskRestart, func(t *testing.T, s *Store) TurnAdmissionRequest {
			quota(t, s, `source_generation='gen', owner_json=?`, owner("task"))
			return TurnAdmissionRequest{Task: Task{TaskID: "T1", AssignedAgentID: agent, AssignedGeneration: "gen"}}
		}, want{}},
		{"task restart held by continued generation", AdmitTaskRestart, func(t *testing.T, s *Store) TurnAdmissionRequest {
			quota(t, s, `continued_generation='gen', owner_json=?`, owner("task"))
			return TurnAdmissionRequest{Task: Task{TaskID: "T1", AssignedAgentID: agent, AssignedGeneration: "gen"}}
		}, want{}},
		{"task restart held by manual generation", AdmitTaskRestart, func(t *testing.T, s *Store) TurnAdmissionRequest {
			quota(t, s, `manual_generation='gen', owner_json=?`, owner("task"))
			return TurnAdmissionRequest{Task: Task{TaskID: "T1", AssignedAgentID: agent, AssignedGeneration: "gen"}}
		}, want{}},
		{"task restart continued intent", AdmitTaskRestart, func(t *testing.T, s *Store) TurnAdmissionRequest {
			quota(t, s, `source_generation='gen', state='continued', owner_json=?`, owner("task"))
			return TurnAdmissionRequest{Task: Task{TaskID: "T1", AssignedAgentID: agent, AssignedGeneration: "gen"}}
		}, want{allowed: true}},
		{"task restart other generation", AdmitTaskRestart, func(t *testing.T, s *Store) TurnAdmissionRequest {
			quota(t, s, `source_generation='old', owner_json=?`, owner("task"))
			return TurnAdmissionRequest{Task: Task{TaskID: "T1", AssignedAgentID: agent, AssignedGeneration: "gen"}}
		}, want{allowed: true}},
		{"task restart other task", AdmitTaskRestart, func(t *testing.T, s *Store) TurnAdmissionRequest {
			quota(t, s, `source_generation='gen', owner_json=?`, owner("task"))
			return TurnAdmissionRequest{Task: Task{TaskID: "T2", AssignedAgentID: agent, AssignedGeneration: "gen"}}
		}, want{allowed: true}},

		// Room restart: quotaContinuationRunningTx.
		{"room restart no intent", AdmitRoomRestart, func(t *testing.T, s *Store) TurnAdmissionRequest {
			room(t, s, "r1")
			return TurnAdmissionRequest{RoomAttempt: ThinkTankAttempt{AgentID: agent, RoomID: "r1", Turn: "discussion", Generation: "gen", TurnID: "turn"}}
		}, want{allowed: true}},
		{"room restart held by its quota continuation", AdmitRoomRestart, func(t *testing.T, s *Store) TurnAdmissionRequest {
			return roomRestartFixture(t, s, agent, "", "gen", "turn", room, attempt, quota, "Quota reached", "failed")
		}, want{}},
		{"room restart continued intent", AdmitRoomRestart, func(t *testing.T, s *Store) TurnAdmissionRequest {
			return roomRestartFixture(t, s, agent, "continued", "gen", "turn", room, attempt, quota, "Quota reached", "failed")
		}, want{allowed: true}},
		{"room restart source failed for another reason", AdmitRoomRestart, func(t *testing.T, s *Store) TurnAdmissionRequest {
			return roomRestartFixture(t, s, agent, "", "gen", "turn", room, attempt, quota, "crashed", "failed")
		}, want{allowed: true}},
		{"room restart source not failed", AdmitRoomRestart, func(t *testing.T, s *Store) TurnAdmissionRequest {
			return roomRestartFixture(t, s, agent, "", "gen", "turn", room, attempt, quota, "Quota reached", "running")
		}, want{allowed: true}},
		{"room restart other continued turn", AdmitRoomRestart, func(t *testing.T, s *Store) TurnAdmissionRequest {
			req := roomRestartFixture(t, s, agent, "", "gen", "turn", room, attempt, quota, "Quota reached", "failed")
			req.RoomAttempt.TurnID = "other-turn"
			return req
		}, want{allowed: true}},

		// Runtime turn: the agent's running turn for this generation.
		{"runtime no durable turn", AdmitRuntimeTurn, func(t *testing.T, s *Store) TurnAdmissionRequest {
			return TurnAdmissionRequest{Generation: "gen"}
		}, want{allowed: true}},
		{"runtime running turn", AdmitRuntimeTurn, func(t *testing.T, s *Store) TurnAdmissionRequest {
			if err := s.BeginConversationTurn(agent, "turn", TurnOrigin{Generation: "gen", Kind: TurnOriginPerson}); err != nil {
				t.Fatal(err)
			}
			return TurnAdmissionRequest{Generation: "gen"}
		}, want{}},
		{"runtime ended turn", AdmitRuntimeTurn, func(t *testing.T, s *Store) TurnAdmissionRequest {
			if err := s.BeginConversationTurn(agent, "turn", TurnOrigin{Generation: "gen", Kind: TurnOriginPerson}); err != nil {
				t.Fatal(err)
			}
			if err := s.EndConversationTurn(agent, TurnEnd{Generation: "gen", TurnID: "turn", Outcome: "done"}); err != nil {
				t.Fatal(err)
			}
			return TurnAdmissionRequest{Generation: "gen"}
		}, want{allowed: true}},
		{"runtime running turn of another generation", AdmitRuntimeTurn, func(t *testing.T, s *Store) TurnAdmissionRequest {
			if err := s.BeginConversationTurn(agent, "turn", TurnOrigin{Generation: "old", Kind: TurnOriginPerson}); err != nil {
				t.Fatal(err)
			}
			return TurnAdmissionRequest{Generation: "gen"}
		}, want{allowed: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _ := newTestStore(t)
			if err := s.WriteAgent(testAgent(agent, timeNow())); err != nil {
				t.Fatal(err)
			}
			req := tc.setup(t, s)
			req.AgentID, req.Check = agent, tc.check
			got, err := s.DecideTurnAdmission(req)
			if !errors.Is(err, tc.want.err) || (tc.want.err == nil) != (err == nil) ||
				got.Allowed != tc.want.allowed || got.WakeTask != tc.want.wake {
				t.Fatalf("admission = %+v, err=%v; want allowed=%t wake=%t err=%v", got, err, tc.want.allowed, tc.want.wake, tc.want.err)
			}
		})
	}
}

// roomRestartFixture builds a room whose source attempt hit quota and a quota
// intent that continued into the current attempt's generation and turn.
func roomRestartFixture(t *testing.T, s *Store, agent, intentState, generation, turn string,
	room func(*testing.T, *Store, string), attempt func(*testing.T, *Store, string, string, string, string, string),
	quota func(*testing.T, *Store, string, ...any) QuotaIntent, failure, sourceState string) TurnAdmissionRequest {
	t.Helper()
	room(t, s, "r1")
	attempt(t, s, "src", "r1", agent, sourceState, failure)
	set := `owner_json=?, continued_generation=?, continued_turn=?`
	args := []any{`{"kind":"room","room_id":"r1","attempt_id":"src","turn":"discussion"}`, generation, turn}
	if intentState != "" {
		set += `, state=?`
		args = append(args, intentState)
	}
	quota(t, s, set, args...)
	return TurnAdmissionRequest{RoomAttempt: ThinkTankAttempt{AgentID: agent, RoomID: "r1", Turn: "discussion", Generation: generation, TurnID: turn}}
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
