package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/messaging"
	"github.com/AsaphNoam/Chuck/internal/recurrence"
	"github.com/AsaphNoam/Chuck/internal/state"
)

func decodeSchedule(t *testing.T, body []byte) state.TaskSchedule {
	t.Helper()
	var sched state.TaskSchedule
	if err := json.Unmarshal(body, &sched); err != nil {
		t.Fatalf("decode schedule: %v (%s)", err, body)
	}
	return sched
}

func putJSON(t *testing.T, url string, body any) (*http.Response, []byte) {
	t.Helper()
	raw, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPut, url, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PUT %s: %v", url, err)
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	return resp, out
}

// makeDue moves a schedule's next occurrence into the recent past, standing in
// for the clock reaching it, and runs one due pass. The live tick may run the
// pass first; either way the durable row is the result.
func makeDue(t *testing.T, srv *Server, scheduleID string) state.TaskSchedule {
	t.Helper()
	// Clearing the occurrence name makes the rewritten instant authoritative.
	if _, err := srv.stateStore.DB().Exec(`UPDATE task_schedules SET next_due_at = ?, next_due_wall = '' WHERE schedule_id = ?`,
		time.Now().Add(-time.Second).UTC().Format(time.RFC3339), scheduleID); err != nil {
		t.Fatal(err)
	}
	srv.processDueSchedules()
	sched, err := srv.stateStore.ReadSchedule(scheduleID)
	if err != nil {
		t.Fatal(err)
	}
	return sched
}

func scheduleTasks(t *testing.T, srv *Server, scheduleID string) []state.Task {
	t.Helper()
	all, err := srv.stateStore.ListTasks("tmpproj")
	if err != nil {
		t.Fatal(err)
	}
	var out []state.Task
	for _, task := range all {
		if task.ScheduleID == scheduleID {
			out = append(out, task)
		}
	}
	return out
}

// FS-16.A38/A39 — over HTTP a person creates, edits, pauses, resumes and
// deletes a schedule. An occurrence for an existing agent is created even while
// the previous one is unfinished; archiving that agent pauses the schedule with
// a reason when the next occurrence comes due, and resuming revalidates it.
// Deleting keeps the tasks.
func TestSchedulesOverHTTP(t *testing.T) {
	srv, ts := wakeTestServer(t)
	if err := srv.stateStore.WriteAgent(state.Agent{
		AgentID: "a_rev", Name: "Nova", Role: "impl", Project: "tmpproj", Interface: "chat", CreatedAt: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	body := map[string]any{
		"project": "tmpproj", "display_name": "daily review", "instruction": "review yesterday",
		"target_kind": "agent", "target_agent_id": "a_rev",
		"rule": map[string]any{"start_date": tomorrow, "times": []string{"09:00"}, "frequency": "daily", "end_kind": "never"},
	}
	resp, raw := post(t, ts.URL+"/api/schedules", body)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create = %d: %s", resp.StatusCode, raw)
	}
	sched := decodeSchedule(t, raw)
	if sched.State != state.ScheduleActive || !sched.OnTimeOnly || sched.RuleText == "" || sched.CreatedByKind != "person" {
		t.Fatalf("created = %+v", sched)
	}

	bad := map[string]any{}
	for k, v := range body {
		bad[k] = v
	}
	bad["rule"] = map[string]any{"start_date": tomorrow, "times": []string{"09:00", "09:00"}, "frequency": "daily", "end_kind": "never"}
	if resp, raw := post(t, ts.URL+"/api/schedules", bad); resp.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(string(raw), "invalid_rule") {
		t.Fatalf("repeated time = %d: %s", resp.StatusCode, raw)
	}
	bad["rule"], bad["target_kind"], bad["target_agent_id"] = body["rule"], "launch", ""
	if resp, raw := post(t, ts.URL+"/api/schedules", bad); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("launch without role = %d: %s", resp.StatusCode, raw)
	}

	first := makeDue(t, srv, sched.ScheduleID)
	if first.LastDisposition != state.OccurrenceCreated || first.OccurrencesElapsed != 1 {
		t.Fatalf("first occurrence = %+v", first)
	}
	// The first occurrence's task is still unfinished (it holds the agent); the
	// next one is created anyway and waits rather than failing.
	if _, err := srv.stateStore.DB().Exec(`UPDATE tasks SET state = 'running', assigned_agent_id = 'a_rev' WHERE task_id = ?`, first.LastTaskID); err != nil {
		t.Fatal(err)
	}
	second := makeDue(t, srv, sched.ScheduleID)
	if second.OccurrencesElapsed != 2 || second.LastTaskID == first.LastTaskID {
		t.Fatalf("second occurrence = %+v", second)
	}
	if task, _ := srv.stateStore.ReadTask(second.LastTaskID); task.State != state.TaskReady {
		t.Fatalf("overlapping occurrence = %s, want ready and waiting", task.State)
	}

	edit := map[string]any{}
	for k, v := range body {
		edit[k] = v
	}
	edit["instruction"], edit["revision"] = "review the last two days", second.Revision
	resp, raw = putJSON(t, ts.URL+"/api/schedules/"+sched.ScheduleID, edit)
	if resp.StatusCode != http.StatusOK || decodeSchedule(t, raw).Instruction != "review the last two days" || decodeSchedule(t, raw).OccurrencesElapsed != 2 {
		t.Fatalf("edit = %d: %s", resp.StatusCode, raw)
	}
	if resp, raw := putJSON(t, ts.URL+"/api/schedules/"+sched.ScheduleID, edit); resp.StatusCode != http.StatusConflict {
		t.Fatalf("stale edit = %d: %s", resp.StatusCode, raw)
	}

	agent, _ := srv.stateStore.ReadAgent("a_rev")
	agent.Archived = true
	if err := srv.stateStore.WriteAgent(agent); err != nil {
		t.Fatal(err)
	}
	paused := makeDue(t, srv, sched.ScheduleID)
	if paused.State != state.SchedulePaused || paused.PauseReason != "the target agent is archived" || len(scheduleTasks(t, srv, sched.ScheduleID)) != 2 {
		t.Fatalf("archived target = %+v", paused)
	}
	if resp, raw := post(t, ts.URL+"/api/schedules/"+sched.ScheduleID+"/resume", nil); resp.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(string(raw), "target_ineligible") {
		t.Fatalf("resume with archived target = %d: %s", resp.StatusCode, raw)
	}
	agent.Archived = false
	if err := srv.stateStore.WriteAgent(agent); err != nil {
		t.Fatal(err)
	}
	resp, raw = post(t, ts.URL+"/api/schedules/"+sched.ScheduleID+"/resume", nil)
	if resp.StatusCode != http.StatusOK || decodeSchedule(t, raw).State != state.ScheduleActive {
		t.Fatalf("resume = %d: %s", resp.StatusCode, raw)
	}
	resp, raw = post(t, ts.URL+"/api/schedules/"+sched.ScheduleID+"/pause", nil)
	if resp.StatusCode != http.StatusOK || decodeSchedule(t, raw).State != state.SchedulePaused {
		t.Fatalf("pause = %d: %s", resp.StatusCode, raw)
	}

	resp, raw = deleteJSON(t, ts.URL+"/api/schedules/"+sched.ScheduleID)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete = %d: %s", resp.StatusCode, raw)
	}
	if got := scheduleTasks(t, srv, sched.ScheduleID); len(got) != 2 {
		t.Fatalf("tasks after delete = %d", len(got))
	}
	if resp, _ := deleteJSON(t, ts.URL+"/api/schedules/"+sched.ScheduleID); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("delete twice = %d", resp.StatusCode)
	}
}

// FS-16.A39 (deleted project) — a schedule whose project was deleted pauses
// when its occurrence comes due instead of creating a task the project's own
// task create would refuse, and resume keeps refusing it.
func TestDeletedProjectPausesSchedule(t *testing.T) {
	srv, ts := wakeTestServer(t)
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	resp, raw := post(t, ts.URL+"/api/schedules", map[string]any{
		"project": "tmpproj", "display_name": "nightly", "instruction": "build",
		"target_kind": "launch", "role": "impl",
		"rule": map[string]any{"start_date": tomorrow, "times": []string{"09:00"}, "frequency": "daily", "end_kind": "never"},
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create = %d: %s", resp.StatusCode, raw)
	}
	sched := decodeSchedule(t, raw)
	if err := srv.configStore.DeleteProject("tmpproj"); err != nil {
		t.Fatal(err)
	}
	paused := makeDue(t, srv, sched.ScheduleID)
	if paused.State != state.SchedulePaused || paused.PauseReason != "the project was deleted" || len(scheduleTasks(t, srv, sched.ScheduleID)) != 0 {
		t.Fatalf("deleted project = %+v", paused)
	}
	if resp, raw := post(t, ts.URL+"/api/schedules/"+sched.ScheduleID+"/resume", nil); resp.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(string(raw), "target_ineligible") {
		t.Fatalf("resume with deleted project = %d: %s", resp.StatusCode, raw)
	}
}

// FS-16.A39 (authority) — an agent may pause, resume and delete only schedules
// it created; another creator's schedule and an unknown one get the same
// refusal.
func TestAgentsControlOnlyTheirOwnSchedules(t *testing.T) {
	srv, _ := wakeTestServer(t)
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	sched, err := srv.CreateAgentSchedule(messaging.AgentScheduleRequest{
		Template: messaging.AgentTaskRequest{
			CreatorAgentID: "a_lead", CreatorGeneration: "g1", Project: "tmpproj",
			DisplayName: "nightly", Instruction: "summarize", Role: "impl",
		},
	})
	if err == nil {
		t.Fatalf("schedule without a rule was created: %+v", sched)
	}
	sched, err = srv.CreateAgentSchedule(messaging.AgentScheduleRequest{
		Template: messaging.AgentTaskRequest{
			CreatorAgentID: "a_lead", CreatorGeneration: "g1", Project: "tmpproj",
			DisplayName: "nightly", Instruction: "summarize", Role: "impl",
		},
		Rule: recurrenceDaily(tomorrow),
	})
	if err != nil || sched.CreatedByAgentID != "a_lead" || sched.TargetKind != state.TargetLaunch {
		t.Fatalf("create = %+v, %v", sched, err)
	}

	for _, id := range []string{sched.ScheduleID, "sc_unknown"} {
		if _, err := srv.PauseAgentSchedule(id, "a_other"); !errors.Is(err, messaging.ErrScheduleNotCreator) {
			t.Fatalf("pause %s by another agent: %v", id, err)
		}
		if err := srv.DeleteAgentSchedule(id, "a_other"); !errors.Is(err, messaging.ErrScheduleNotCreator) {
			t.Fatalf("delete %s by another agent: %v", id, err)
		}
	}
	if paused, err := srv.PauseAgentSchedule(sched.ScheduleID, "a_lead"); err != nil || paused.State != state.SchedulePaused {
		t.Fatalf("pause own = %+v, %v", paused, err)
	}
	if resumed, err := srv.ResumeAgentSchedule(sched.ScheduleID, "a_lead"); err != nil || resumed.State != state.ScheduleActive {
		t.Fatalf("resume own = %+v, %v", resumed, err)
	}
	if err := srv.DeleteAgentSchedule(sched.ScheduleID, "a_lead"); err != nil {
		t.Fatalf("delete own: %v", err)
	}
}

func recurrenceDaily(start string) recurrence.Rule {
	return recurrence.Rule{StartDate: start, Times: []string{"09:00"}, Frequency: recurrence.FreqDaily, EndKind: recurrence.EndNever}
}
