package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/AsaphNoam/Chuck/internal/messaging"
	"github.com/AsaphNoam/Chuck/internal/recurrence"
	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
)

// scheduleRequest is the HTTP create/edit body: create_task's template fields
// plus the rule (FS-16.R53, TS-10.R60). Revision guards an edit.
type scheduleRequest struct {
	Project     string `json:"project"`
	DisplayName string `json:"display_name"`
	Instruction string `json:"instruction"`

	TargetKind    string `json:"target_kind"`
	TargetAgentID string `json:"target_agent_id,omitempty"`
	Role          string `json:"role,omitempty"`
	Backend       string `json:"backend,omitempty"`
	Model         string `json:"model,omitempty"`
	Effort        string `json:"effort,omitempty"`
	Fast          bool   `json:"fast"`

	Rule          recurrence.Rule           `json:"rule"`
	RunOnlyOnTime *bool                     `json:"run_only_on_time,omitempty"`
	Attachments   []createAttachmentRequest `json:"attachments,omitempty"`
	Revision      int                       `json:"revision,omitempty"`
}

// composeSchedule validates the template exactly as a task create would, so a
// schedule can never hold a target a task could not (FS-16.R56–R57).
func (s *Server) composeSchedule(req scheduleRequest) (state.TaskSchedule, *runtime.APIError) {
	task, ae := s.composeTask(createTaskRequest{
		Project: req.Project, DisplayName: req.DisplayName, Instruction: req.Instruction,
		TargetKind: req.TargetKind, TargetAgentID: req.TargetAgentID, Role: req.Role,
		Backend: req.Backend, Model: req.Model, Effort: req.Effort, Fast: req.Fast,
		Attachments: req.Attachments,
	})
	if ae != nil {
		return state.TaskSchedule{}, ae
	}
	return scheduleFromTemplate(task, req.Rule, req.RunOnlyOnTime, taskAttachments(req.Attachments)), nil
}

func scheduleFromTemplate(task state.Task, rule recurrence.Rule, runOnlyOnTime *bool, attachments []state.TaskAttachment) state.TaskSchedule {
	return state.TaskSchedule{
		Project: task.Project, DisplayName: task.DisplayName, Instruction: task.Instruction,
		TargetKind: task.TargetKind, TargetAgentID: task.TargetAgentID, Role: task.Role,
		Backend: task.Backend, Model: task.Model, Effort: task.Effort, Fast: task.Fast, Group: task.Group,
		Rule: rule, OnTimeOnly: runOnlyOnTime == nil || *runOnlyOnTime,
		CreatedByKind: task.CreatedByKind, CreatedByAgentID: task.CreatedByAgentID,
		CreatedByGeneration: task.CreatedByGeneration, Attachments: attachments,
	}
}

// handleSchedules implements GET /api/schedules?project=.
func (s *Server) handleSchedules(w http.ResponseWriter, r *http.Request) {
	project := r.URL.Query().Get("project")
	if project == "" {
		writeAPIError(w, apiError(runtime.CodeValidation, "project is required"))
		return
	}
	schedules, err := s.stateStore.ListSchedules(project)
	if err != nil {
		s.writeScheduleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"schedules": schedules})
}

// handleCreateSchedule implements POST /api/schedules. A person is the
// creator of record, as for a task.
func (s *Server) handleCreateSchedule(w http.ResponseWriter, r *http.Request) {
	var req scheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, apiError(runtime.CodeValidation, "invalid JSON body"))
		return
	}
	sched, ae := s.composeSchedule(req)
	if ae != nil {
		writeAPIError(w, ae)
		return
	}
	created, err := s.createSchedule(sched, "")
	if err != nil {
		s.writeScheduleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) createSchedule(sched state.TaskSchedule, creatorAgentID string) (state.TaskSchedule, error) {
	id, err := state.NewScheduleID()
	if err != nil {
		return state.TaskSchedule{}, err
	}
	sched.ScheduleID = id
	created, err := s.stateStore.CreateSchedule(sched, creatorAgentID, time.Local)
	if err != nil {
		return state.TaskSchedule{}, err
	}
	s.publishScheduleUpdate(created)
	return created, nil
}

// handleScheduleDetail implements GET /api/schedules/{id}.
func (s *Server) handleScheduleDetail(w http.ResponseWriter, r *http.Request) {
	sched, err := s.stateStore.ReadSchedule(r.PathValue("id"))
	if err != nil {
		s.writeScheduleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sched)
}

// handleEditSchedule implements PUT /api/schedules/{id}: the template and rule
// are replaced under the observed revision; the project and creator are kept.
func (s *Server) handleEditSchedule(w http.ResponseWriter, r *http.Request) {
	var req scheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, apiError(runtime.CodeValidation, "invalid JSON body"))
		return
	}
	current, err := s.stateStore.ReadSchedule(r.PathValue("id"))
	if err != nil {
		s.writeScheduleError(w, err)
		return
	}
	req.Project = current.Project
	edit, ae := s.composeSchedule(req)
	if ae != nil {
		writeAPIError(w, ae)
		return
	}
	edit.ScheduleID = current.ScheduleID
	// The HTTP template has no group field (as for a task); a launch target
	// keeps the group an agent chose rather than losing it on edit.
	if edit.TargetKind == state.TargetLaunch {
		edit.Group = current.Group
	}
	// An agent-created schedule keeps its creator's attachment authority.
	updated, err := s.stateStore.UpdateSchedule(edit, req.Revision, current.CreatedByAgentID, time.Local)
	if err != nil {
		s.writeScheduleError(w, err)
		return
	}
	s.publishScheduleUpdate(updated)
	writeJSON(w, http.StatusOK, updated)
}

// handlePauseSchedule implements POST /api/schedules/{id}/pause.
func (s *Server) handlePauseSchedule(w http.ResponseWriter, r *http.Request) {
	sched, err := s.pauseSchedule(r.PathValue("id"))
	if err != nil {
		s.writeScheduleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sched)
}

// handleResumeSchedule implements POST /api/schedules/{id}/resume.
func (s *Server) handleResumeSchedule(w http.ResponseWriter, r *http.Request) {
	sched, err := s.resumeSchedule(r.PathValue("id"))
	if err != nil {
		s.writeScheduleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sched)
}

// handleDeleteSchedule implements DELETE /api/schedules/{id}.
func (s *Server) handleDeleteSchedule(w http.ResponseWriter, r *http.Request) {
	sched, err := s.stateStore.ReadSchedule(r.PathValue("id"))
	if err != nil {
		s.writeScheduleError(w, err)
		return
	}
	if err := s.deleteSchedule(sched); err != nil {
		s.writeScheduleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) pauseSchedule(id string) (state.TaskSchedule, error) {
	sched, err := s.stateStore.PauseSchedule(id, "")
	if err != nil {
		return state.TaskSchedule{}, err
	}
	s.publishScheduleUpdate(sched)
	return sched, nil
}

// errScheduleTarget refuses a resume whose target still cannot run (R57).
var errScheduleTarget = errors.New("schedule target cannot run")

func (s *Server) resumeSchedule(id string) (state.TaskSchedule, error) {
	current, err := s.stateStore.ReadSchedule(id)
	if err != nil {
		return state.TaskSchedule{}, err
	}
	if reason := s.scheduleUnavailable(current); reason != "" {
		return state.TaskSchedule{}, errors.Join(errScheduleTarget, errors.New(reason))
	}
	sched, err := s.stateStore.ResumeSchedule(id, time.Local)
	if err != nil {
		return state.TaskSchedule{}, err
	}
	s.publishScheduleUpdate(sched)
	return sched, nil
}

func (s *Server) deleteSchedule(sched state.TaskSchedule) error {
	if err := s.stateStore.DeleteSchedule(sched.ScheduleID); err != nil {
		return err
	}
	s.eventBus.Publish("schedule_update", nil, map[string]any{
		"schedule_id": sched.ScheduleID, "project": sched.Project, "deleted": true,
	})
	return nil
}

func (s *Server) writeScheduleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, state.ErrNotFound):
		writeAPIError(w, apiError(runtime.CodeNotFound, "no such schedule"))
	case errors.Is(err, state.ErrScheduleRule):
		ae := apiError(runtime.CodeValidation, err.Error())
		ae.Details = map[string]any{"code": "invalid_rule"}
		writeAPIError(w, ae)
	case errors.Is(err, state.ErrScheduleConflict):
		writeAPIError(w, apiError(runtime.CodeConflict, "the schedule changed while this request was in flight"))
	case errors.Is(err, state.ErrScheduleState):
		ae := apiError(runtime.CodeConflict, "the schedule cannot change from its current state")
		ae.Details = map[string]any{"code": "invalid_state"}
		writeAPIError(w, ae)
	case errors.Is(err, errScheduleTarget):
		ae := apiError(runtime.CodeValidation, err.Error())
		ae.Details = map[string]any{"code": "target_ineligible"}
		writeAPIError(w, ae)
	default:
		s.writeTaskError(w, err)
	}
}

// CreateAgentSchedule creates a schedule for a token-bound agent on exactly the
// terms create_task uses (FS-16.R56, TS-10.R60).
func (s *Server) CreateAgentSchedule(req messaging.AgentScheduleRequest) (state.TaskSchedule, error) {
	task, err := s.composeAgentTask(req.Template)
	if err != nil {
		return state.TaskSchedule{}, err
	}
	return s.createSchedule(scheduleFromTemplate(task, req.Rule, req.RunOnlyOnTime, req.Template.Attachments),
		req.Template.CreatorAgentID)
}

// agentOwnedSchedule answers an unknown schedule and another creator's the
// same way (TS-10.R60, TS-05.R14).
func (s *Server) agentOwnedSchedule(id, creatorAgentID string) (state.TaskSchedule, error) {
	sched, err := s.stateStore.ReadSchedule(id)
	if errors.Is(err, state.ErrNotFound) ||
		(err == nil && (sched.CreatedByKind != "agent" || sched.CreatedByAgentID != creatorAgentID)) {
		return state.TaskSchedule{}, messaging.ErrScheduleNotCreator
	}
	return sched, err
}

func (s *Server) PauseAgentSchedule(id, creatorAgentID string) (state.TaskSchedule, error) {
	if _, err := s.agentOwnedSchedule(id, creatorAgentID); err != nil {
		return state.TaskSchedule{}, err
	}
	return s.pauseSchedule(id)
}

func (s *Server) ResumeAgentSchedule(id, creatorAgentID string) (state.TaskSchedule, error) {
	if _, err := s.agentOwnedSchedule(id, creatorAgentID); err != nil {
		return state.TaskSchedule{}, err
	}
	sched, err := s.resumeSchedule(id)
	if errors.Is(err, errScheduleTarget) {
		return state.TaskSchedule{}, &messaging.ToolError{Code: "target_ineligible", Message: err.Error()}
	}
	return sched, err
}

func (s *Server) DeleteAgentSchedule(id, creatorAgentID string) error {
	sched, err := s.agentOwnedSchedule(id, creatorAgentID)
	if err != nil {
		return err
	}
	return s.deleteSchedule(sched)
}
