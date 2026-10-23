package server

import (
	"context"
	"testing"

	"github.com/AsaphNoam/Chuck/internal/pipeline"
	"github.com/AsaphNoam/Chuck/internal/state"
)

// FS-14.R72/R74, TS-09.R35/R46, INV §17 — pipeline Start freezes the
// standing stage assignment as a task, and the ordinary task dispatcher owns
// the provider handoff. The task keeps the requested fast mode; the launched
// agent enters the ordinary run-name group and records what the fake provider
// actually applied.
func TestPipelineStageTaskUsesDispatcherForFrozenRuntimeAndGrouping(t *testing.T) {
	for _, tt := range []struct {
		name       string
		fastOption string
		wantFast   bool
	}{
		{name: "provider applies fast mode", fastOption: "fast", wantFast: true},
		{name: "provider does not advertise fast mode", fastOption: "", wantFast: false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("FAKEACP_FAST_OPTION", tt.fastOption)
			srv, _ := wakeTestServer(t)
			writeFastCapableBackend(t, srv)

			template := pipeline.Template{
				Version: 2, Title: "Release flow", OrchestratorRole: "impl",
				Inputs: []pipeline.ValueDecl{},
				Stages: []pipeline.Stage{{
					ID: "build", Title: "Build", Objective: "Build the release",
					Coordination: pipeline.CoordinationStanding,
					Inputs:       []pipeline.StageInput{}, Outputs: []pipeline.StageOutput{},
				}},
			}
			if record, err := srv.pipelineTemplates.Create("release-flow", template); err != nil || !record.Valid {
				t.Fatalf("create template = %+v err=%v", record, err)
			}

			detail, replay, err := srv.pipelineMgr.Start(context.Background(), pipeline.StartRequest{
				RequestID: "pipeline-handoff", TemplateID: "release-flow", DisplayName: "Ship it",
				Project: "tmpproj", Goal: "Ship the release", Inputs: map[string]string{},
				Orchestrator: pipeline.RuntimeAssignment{Backend: "claude", Model: "sonnet", Fast: true},
			})
			if err != nil || replay {
				t.Fatalf("Start = %+v replay=%v err=%v", detail, replay, err)
			}
			stages, err := srv.stateStore.ListPipelineStageTasks(detail.Run.RunID)
			if err != nil || len(stages) != 1 {
				t.Fatalf("stage tasks = %+v err=%v", stages, err)
			}
			stage := stages[0]
			stored, err := srv.stateStore.ReadTask(stage.TaskID)
			if err != nil {
				t.Fatalf("ReadTask before dispatch: %v", err)
			}
			if stored.State != state.TaskReady || !stored.Fast {
				t.Fatalf("frozen stage task = %+v, want ready fast=%v", stored, true)
			}

			srv.dispatchReadyTasks(context.Background())
			running := dispatchUntilTaskState(t, srv, stage.TaskID, state.TaskRunning)
			if running.AssignedAgentID == "" {
				t.Fatal("dispatched stage task has no assigned agent")
			}
			agent, err := srv.stateStore.ReadAgent(running.AssignedAgentID)
			if err != nil {
				t.Fatalf("ReadAgent: %v", err)
			}
			if agent.Fast != tt.wantFast || agent.Group != "Ship it" {
				t.Fatalf("applied agent fast=%v group=%q, want fast=%v group=%q", agent.Fast, agent.Group, tt.wantFast, "Ship it")
			}
			session, err := srv.stateStore.ReadSession(running.AssignedAgentID)
			if err != nil {
				t.Fatalf("ReadSession: %v", err)
			}
			if session.Fast != tt.wantFast {
				t.Fatalf("applied session fast=%v, want %v", session.Fast, tt.wantFast)
			}
		})
	}
}
