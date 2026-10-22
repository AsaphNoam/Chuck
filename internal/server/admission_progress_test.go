package server

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/state"
)

// CR-A2-6: more than one read batch of quota-blocked work must not hide a
// later eligible provider turn, even when its fast-path notification was lost.
func TestBlockedPrefixDoesNotHideEligibleWork(t *testing.T) {
	for _, kind := range []string{"task", "mail"} {
		t.Run(kind, func(t *testing.T) {
			srv, ts, promptLog := activationTestServer(t)
			eligibleAgent := launchAndWaitIdle(t, ts, "impl", "tmpproj")
			var tasks []state.Task
			for i := 0; i < taskDispatchBatch; i++ {
				id := fmt.Sprintf("a_blocked_%d", i)
				if err := srv.stateStore.WriteAgent(state.Agent{AgentID: id, Name: id, Role: "impl", Project: "tmpproj", Backend: "claude", Model: "sonnet", Interface: "chat", CreatedAt: time.Now().UTC()}); err != nil {
					t.Fatal(err)
				}
				if _, err := srv.stateStore.ObserveQuota(state.QuotaObservation{AgentID: id, EpisodeID: "blocked-episode-" + id, SourceGeneration: "blocked-generation", ExecutingTurn: "blocked-turn", Owner: state.QuotaOwner{Kind: "ordinary"}, ObservedAt: time.Now().UTC(), State: state.QuotaUnknownReset}); err != nil {
					t.Fatal(err)
				}
				if kind == "task" {
					tasks = append(tasks, newAgentTask(t, srv, id, "blocked"))
				} else {
					insertMail(t, srv, id, "blocked")
					if _, err := srv.stateStore.DB().Exec(`UPDATE activations SET created_at=? WHERE agent_id=?`, fmt.Sprintf("2026-01-01T00:00:%02dZ", i), id); err != nil {
						t.Fatal(err)
					}
				}
			}
			if kind == "task" {
				ninth := newAgentTask(t, srv, eligibleAgent, "eligible")
				tasks = append(tasks, ninth)
				stampReadyOrder(t, srv, tasks...)
				srv.dispatchReadyTasks(context.Background())
				if _, err := srv.stateStore.DeleteTask(tasks[0].TaskID); err != nil {
					t.Fatal(err)
				}
				srv.dispatchReadyTasks(context.Background())
				waitTaskState(t, srv, ninth.TaskID, state.TaskRunning)
			} else {
				insertMail(t, srv, eligibleAgent, "eligible")
				srv.executePendingMailActivations(context.Background(), "")
				// The first batch exits before the next bounded pass, so no
				// worker-capacity timing can stand in for the selection promise.
				deadline := time.Now().Add(5 * time.Second)
				for len(srv.activationSlots) != 0 {
					if time.Now().After(deadline) {
						t.Fatal("blocked mail workers did not finish")
					}
					time.Sleep(time.Millisecond)
				}
				if _, err := srv.stateStore.DB().Exec(`DELETE FROM activations WHERE agent_id='a_blocked_0'`); err != nil {
					t.Fatal(err)
				}
				srv.executePendingMailActivations(context.Background(), "")
			}
			waitPrompts(t, promptLog, 1)
			for i := 1; i < taskDispatchBatch; i++ {
				if kind == "task" {
					row, err := srv.stateStore.ReadTask(tasks[i].TaskID)
					if err != nil || row.State != state.TaskReady || row.StartAttemptCount != 0 {
						t.Fatalf("blocked task changed retry policy: %+v %v", row, err)
					}
				} else if len(pendingActivations(t, srv, fmt.Sprintf("a_blocked_%d", i))) != 1 {
					t.Fatal("blocked mail was discarded")
				}
			}
		})
	}
}
