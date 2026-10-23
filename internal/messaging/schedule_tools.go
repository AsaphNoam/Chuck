package messaging

import (
	"context"
	"errors"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/AsaphNoam/Chuck/internal/recurrence"
	"github.com/AsaphNoam/Chuck/internal/state"
)

// AgentScheduleRequest is one agent-created schedule after identity and target
// resolution: the task template create_task would send, plus its rule.
type AgentScheduleRequest struct {
	Template      AgentTaskRequest
	Rule          recurrence.Rule
	RunOnlyOnTime *bool
}

// ErrScheduleNotCreator refuses pause, resume or delete of a schedule the
// caller did not create. An unknown schedule gets the same answer, so a caller
// cannot probe for schedules it does not own (FS-16.R56, TS-10.R60).
var ErrScheduleNotCreator = errors.New("messaging: schedule not created by caller")

type createScheduleArgs struct {
	DisplayName string          `json:"display_name" jsonschema:"short name for each task the schedule creates"`
	Instruction string          `json:"instruction" jsonschema:"what the assigned agent should do at each occurrence"`
	To          string          `json:"to,omitempty" jsonschema:"existing agent to assign: role@project, name, or agent_id; omit to launch a new agent each time"`
	Role        string          `json:"role,omitempty" jsonschema:"role for a new agent when no target is named"`
	Backend     string          `json:"backend,omitempty" jsonschema:"optional backend for a new agent"`
	Model       string          `json:"model,omitempty" jsonschema:"optional model for a new agent"`
	Effort      string          `json:"effort,omitempty" jsonschema:"optional reasoning effort for a new agent; must be a level the chosen model declares"`
	Fast        bool            `json:"fast,omitempty" jsonschema:"request fast mode for a new agent; the chosen model must declare it"`
	Group       string          `json:"group,omitempty" jsonschema:"optional project dashboard group for a new agent"`
	Rule        recurrence.Rule `json:"rule" jsonschema:"start_date YYYY-MM-DD; times as HH:MM local; frequency daily, weekdays, weekly, every_n_days or every_n_weeks; weekdays mon..sun for weekly and every_n_weeks; interval N for every_n_*; end_kind never, count (end_count) or until (end_date)"`
	// RunOnlyOnTime is a pointer so an omitted value keeps the default of true.
	RunOnlyOnTime *bool                   `json:"run_only_on_time,omitempty" jsonschema:"default true: an occurrence missed while Chuck was off or asleep is skipped instead of run late"`
	Attachments   []createAttachmentInput `json:"attachments,omitempty" jsonschema:"context references to attach to every task the schedule creates"`
}

type scheduleIDArgs struct {
	ScheduleID string `json:"schedule_id" jsonschema:"the id of a schedule you created"`
}

// handleCreateSchedule creates a recurring start on the same target, selector
// and validation terms as create_task (FS-16.R56).
func (s *Server) handleCreateSchedule(_ context.Context, req *mcp.CallToolRequest, input createScheduleArgs) (*mcp.CallToolResult, any, error) {
	if _, ok := s.caller(req); !ok {
		return sessionUnknown()
	}
	control := s.taskControl()
	if control == nil {
		return errResult(map[string]any{"ok": false, "error": "internal", "message": "Task control plane is unavailable."})
	}
	template, refusal := s.agentTaskRequest(req, createTaskArgs{
		DisplayName: input.DisplayName, Instruction: input.Instruction, To: input.To, Role: input.Role,
		Backend: input.Backend, Model: input.Model, Effort: input.Effort, Fast: input.Fast, Group: input.Group,
		Attachments: input.Attachments,
	})
	if refusal != nil {
		return refusal()
	}
	sched, err := control.CreateAgentSchedule(AgentScheduleRequest{
		Template: template, Rule: input.Rule, RunOnlyOnTime: input.RunOnlyOnTime,
	})
	if err != nil {
		return scheduleToolError(err)
	}
	return scheduleResult(sched)
}

func (s *Server) handlePauseSchedule(_ context.Context, req *mcp.CallToolRequest, input scheduleIDArgs) (*mcp.CallToolResult, any, error) {
	return s.controlSchedule(req, input, TaskControl.PauseAgentSchedule)
}

func (s *Server) handleResumeSchedule(_ context.Context, req *mcp.CallToolRequest, input scheduleIDArgs) (*mcp.CallToolResult, any, error) {
	return s.controlSchedule(req, input, TaskControl.ResumeAgentSchedule)
}

func (s *Server) handleDeleteSchedule(_ context.Context, req *mcp.CallToolRequest, input scheduleIDArgs) (*mcp.CallToolResult, any, error) {
	identity, ok := s.caller(req)
	if !ok {
		return sessionUnknown()
	}
	control := s.taskControl()
	if control == nil {
		return errResult(map[string]any{"ok": false, "error": "internal", "message": "Task control plane is unavailable."})
	}
	if err := control.DeleteAgentSchedule(input.ScheduleID, identity.AgentID); err != nil {
		return scheduleToolError(err)
	}
	return jsonResult(map[string]any{"ok": true, "schedule_id": input.ScheduleID, "deleted": true})
}

func (s *Server) controlSchedule(req *mcp.CallToolRequest, input scheduleIDArgs,
	op func(TaskControl, string, string) (state.TaskSchedule, error)) (*mcp.CallToolResult, any, error) {
	identity, ok := s.caller(req)
	if !ok {
		return sessionUnknown()
	}
	control := s.taskControl()
	if control == nil {
		return errResult(map[string]any{"ok": false, "error": "internal", "message": "Task control plane is unavailable."})
	}
	sched, err := op(control, input.ScheduleID, identity.AgentID)
	if err != nil {
		return scheduleToolError(err)
	}
	return scheduleResult(sched)
}

func scheduleResult(sched state.TaskSchedule) (*mcp.CallToolResult, any, error) {
	result := map[string]any{
		"ok": true, "schedule_id": sched.ScheduleID, "state": sched.State, "rule": sched.RuleText,
	}
	if sched.NextDueAt != nil {
		result["next_occurrence"] = sched.NextDueAt
	}
	if sched.PauseReason != "" {
		result["pause_reason"] = sched.PauseReason
	}
	return jsonResult(result)
}

func scheduleToolError(err error) (*mcp.CallToolResult, any, error) {
	switch {
	case errors.Is(err, ErrScheduleNotCreator):
		return errResult(map[string]any{"ok": false, "error": "not_creator",
			"message": "No schedule you created has that id."})
	case errors.Is(err, state.ErrScheduleRule):
		return errResult(map[string]any{"ok": false, "error": "validation", "message": err.Error()})
	case errors.Is(err, state.ErrScheduleState):
		return errResult(map[string]any{"ok": false, "error": "invalid_state",
			"message": "The schedule cannot change from its current state."})
	}
	return taskToolError(err)
}
