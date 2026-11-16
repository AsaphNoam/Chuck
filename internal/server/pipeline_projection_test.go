package server

import (
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/pipeline"
	"github.com/AsaphNoam/Chuck/internal/state"
)

// CONFIG-3, FS-02.R69 — a retained run whose standing owner, coordinator and
// attempt agent no longer exist keeps their IDs for routing but projects no
// name, so no surface can render the raw ID as one.
func TestPipelineProjectionLeavesMissingAgentNamesEmpty(t *testing.T) {
	srv := testServer(t, true)
	const runID = "pr_missing_agents"
	task := createPipelineSnapshotTask(t, srv, runID, "tk_missing_owner", inputSnapshot)
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := srv.stateStore.DB().Exec(`
INSERT INTO tasks(task_id, project, display_name, instruction, target_kind, state, created_by_kind, assigned_agent_id, created_at, updated_at)
VALUES ('tk_missing_coordinator', 'tmpproj', 'Coordinator', '', 'agent', 'finished', 'pipeline', 'a_gone_coordinator', ?, ?)`, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := srv.stateStore.DB().Exec(`UPDATE pipeline_stage_tasks SET standing_agent_id = 'a_gone_owner', coordinator_task_id = 'tk_missing_coordinator' WHERE task_id = ?`, task.TaskID); err != nil {
		t.Fatal(err)
	}
	detail, err := srv.pipelineMgr.Detail(runID)
	if err != nil {
		t.Fatal(err)
	}
	stages, _, err := srv.pipelineTaskRunProjection(detail)
	if err != nil || len(stages) != 1 {
		t.Fatalf("projection = %+v, %v", stages, err)
	}
	owner := stages[0].StandingOwner
	if owner.AgentID != "a_gone_owner" || owner.Name != "" || owner.Route != "unavailable" {
		t.Fatalf("standing owner = %+v", owner)
	}
	coordinator := stages[0].Coordinator
	if coordinator == nil || coordinator.AgentID != "a_gone_coordinator" || coordinator.Name != "" || coordinator.Route != "unavailable" {
		t.Fatalf("coordinator = %+v", coordinator)
	}

	agents, err := srv.pipelineAttemptAgents(pipeline.RunDetail{
		Run:      state.PipelineRunRecord{Project: "tmpproj"},
		Attempts: []state.PipelineAttemptRecord{{AttemptID: "pa_gone", AgentID: "a_gone_owner", CreatedAt: time.Now().UTC()}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if stage := agents["pa_gone"].StageAgent; stage == nil || stage.AgentID != "a_gone_owner" || stage.Name != "" || stage.Route != "unavailable" {
		t.Fatalf("attempt stage agent = %+v", stage)
	}
}
