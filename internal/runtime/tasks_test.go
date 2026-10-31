package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/state"
)

func startTaskAgent(t *testing.T, env ...string) (*ChatRuntime, *Handle, <-chan Event) {
	t.Helper()
	c, spec := newChatTest(t, "task_flow")
	spec.Env = append(spec.Env, env...)
	ctx := context.Background()
	h, err := c.Start(ctx, spec)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = c.Stop(ctx, h.AgentID) })
	ch, unsub, err := c.Subscribe(h.AgentID)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	t.Cleanup(unsub)
	if err := c.SendPrompt(ctx, h.AgentID, "serve"); err != nil {
		t.Fatalf("SendPrompt: %v", err)
	}
	return c, h, ch
}

func taskStates(evs []Event) []string {
	var out []string
	for _, ev := range evs {
		if ev.Type == EvBackgroundTaskState {
			var d BackgroundTaskData
			_ = json.Unmarshal(ev.Data, &d)
			out = append(out, ev.ActivityID+"|"+d.TaskID+"|"+d.ToolCallID+"|"+d.State+"|"+d.Name)
		}
	}
	return out
}

// A root and a child task each produce exactly one lifecycle row per state,
// tied to their tool call; a replayed announcement adds none (FS-03.A41).
func TestBackgroundTasksAreNormalizedPerScope(t *testing.T) {
	_, _, ch := startTaskAgent(t, "FAKEACP_CAPS=1")
	child := activityIDFor("th_child_1")
	got := taskStates(drainTurn(t, ch))
	want := []string{
		"|task_1|tc_bg|running|npm run dev",
		child + "|" + child + "/task_1|" + child + "/tc_c|running|go test",
		child + "|" + child + "/task_1|" + child + "/tc_c|completed|",
	}
	if len(got) != len(want) {
		t.Fatalf("task rows = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("row %d = %q, want %q", i, got[i], want[i])
		}
	}
}

// Targeted stop addresses only the chosen task; acceptance emits nothing of its
// own, the runtime's terminal update does; a finished or unknown task is 404-
// shaped and a refusal keeps the task running and retryable (FS-03.A41).
func TestStopBackgroundTaskWaitsForTheRuntimeUpdate(t *testing.T) {
	c, h, ch := startTaskAgent(t, "FAKEACP_CAPS=1")
	drainTurn(t, ch)
	ctx := context.Background()
	child := activityIDFor("th_child_1")
	if err := c.StopBackgroundTask(ctx, h.AgentID, child+"/task_1"); !errors.Is(err, ErrUnknownBackgroundTask) {
		t.Fatalf("stop finished child task = %v", err)
	}
	if err := c.StopBackgroundTask(ctx, h.AgentID, "nope"); !errors.Is(err, ErrUnknownBackgroundTask) {
		t.Fatalf("stop unknown task = %v", err)
	}
	if err := c.StopBackgroundTask(ctx, h.AgentID, "task_1"); err != nil {
		t.Fatalf("StopBackgroundTask: %v", err)
	}
	select {
	case ev := <-ch:
		if got := taskStates([]Event{ev}); len(got) != 1 || got[0] != "|task_1|tc_bg|stopped|" {
			t.Fatalf("after stop = %q (%s)", got, ev.Type)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no terminal task update")
	}
	if err := c.StopBackgroundTask(ctx, h.AgentID, "task_1"); !errors.Is(err, ErrUnknownBackgroundTask) {
		t.Fatalf("second stop = %v", err)
	}
}

// The status row carries the generation's running count: the root task stays
// counted after the child's completes, and the runtime's stop update clears
// it without a status transition (TS-04.R104, FS-02.A64).
func TestBackgroundRunningCountIsPublished(t *testing.T) {
	c, h, ch := startTaskAgent(t, "FAKEACP_CAPS=1")
	drainTurn(t, ch)
	before := waitForBackgroundRunning(t, c, h.AgentID, 1)
	if err := c.StopBackgroundTask(context.Background(), h.AgentID, "task_1"); err != nil {
		t.Fatalf("StopBackgroundTask: %v", err)
	}
	after := waitForBackgroundRunning(t, c, h.AgentID, 0)
	if after.State != before.State || after.Detail != before.Detail || after.LastTrace != before.LastTrace || (after.BusySince == nil) != (before.BusySince == nil) {
		t.Fatalf("count republish changed status: before %+v after %+v", before, after)
	}
}

// A prompt/activation busy row built while a task was running must not restore
// that count when a real terminal update publishes before the busy write:
// writeStatus refreshes the live projections under statusMu (TS-04.R104, INV §5).
func TestBusyWriteKeepsCurrentBackgroundCount(t *testing.T) {
	c, h, ch := startTaskAgent(t, "FAKEACP_CAPS=1")
	drainTurn(t, ch)
	waitForBackgroundRunning(t, c, h.AgentID, 1)
	as, err := c.lookup(h.AgentID)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	busy := state.Status{AgentID: h.AgentID, State: "busy", Detail: "thinking", LastTrace: "UserPromptSubmit", BusySince: &now, BackgroundRunning: 1}
	if err := c.StopBackgroundTask(context.Background(), h.AgentID, "task_1"); err != nil {
		t.Fatalf("StopBackgroundTask: %v", err)
	}
	waitForBackgroundRunning(t, c, h.AgentID, 0)
	if err := c.writeStatus(as, busy); err != nil {
		t.Fatal(err)
	}
	got, err := c.store.ReadStatus(h.AgentID)
	if err != nil {
		t.Fatal(err)
	}
	if got.State != "busy" || got.BackgroundRunning != 0 {
		t.Fatalf("busy write = state %q count %d, want busy with count 0", got.State, got.BackgroundRunning)
	}
}

func waitForBackgroundRunning(t *testing.T, c *ChatRuntime, agentID string, want int) state.Status {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		got, err := c.store.ReadStatus(agentID)
		if err == nil && got.BackgroundRunning == want {
			return got
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for background_running %d (last %+v err %v)", want, got, err)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestStopBackgroundTaskRefusalAndCapability(t *testing.T) {
	c, h, ch := startTaskAgent(t, "FAKEACP_CAPS=1", "FAKEACP_TASK_STOP=refuse")
	drainTurn(t, ch)
	ctx := context.Background()
	if err := c.StopBackgroundTask(ctx, h.AgentID, "task_1"); !errors.Is(err, ErrBackgroundTaskStopRefused) {
		t.Fatalf("refused stop = %v", err)
	}
	// Still running, so the person can retry.
	if err := c.StopBackgroundTask(ctx, h.AgentID, "task_1"); !errors.Is(err, ErrBackgroundTaskStopRefused) {
		t.Fatalf("retry after refusal = %v", err)
	}

	plain, plainHandle, plainCh := startTaskAgent(t)
	for _, row := range taskStates(drainTurn(t, plainCh)) {
		t.Fatalf("non-advertising runtime produced task row %q", row)
	}
	if err := plain.StopBackgroundTask(ctx, plainHandle.AgentID, "task_1"); !errors.Is(err, ErrBackgroundTaskControlUnavailable) {
		t.Fatalf("stop without capability = %v", err)
	}
}
