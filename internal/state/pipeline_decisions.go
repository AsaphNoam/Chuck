package state

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

// Goal-led decisions (TS-09.R66/R68). The run owner's judgment travels with its
// existing stage result: `next` names an existing step (a repeat or revisit
// included), `complete` claims the goal is met with evidence for every
// explicit check, and `needs_input` asks the person one concrete question.
const (
	DecisionNext       = "next"
	DecisionComplete   = "complete"
	DecisionNeedsInput = "needs_input"

	// MaxDecisionReasonRunes and MaxDecisionQuestionRunes bound the decision's
	// prose; required control fields are never clipped (TS-09.R71).
	MaxDecisionReasonRunes   = 2000
	MaxDecisionQuestionRunes = 4000
	// MaxPipelineSteps bounds the step menu and required-check references.
	MaxPipelineSteps = 32
	// PipelineRecoveryRounds is the default automatic repair/revisit allowance.
	PipelineRecoveryRounds = 3
)

// PipelineDecision is the owner's optional stage-result decision. Implicit
// marks the usual next step Chuck recorded for a success with no decision.
type PipelineDecision struct {
	Action         string            `json:"action"`
	StageID        string            `json:"stage_id,omitempty"`
	Reason         string            `json:"reason,omitempty"`
	Question       string            `json:"question,omitempty"`
	RequiredChecks map[string]string `json:"required_checks,omitempty"`
	Implicit       bool              `json:"implicit,omitempty"`
}

// DecisionError is a correctable refusal: the assignment stays actionable and
// the message tells the owner exactly what to change (TS-09.R68).
type DecisionError struct{ Message string }

func (e *DecisionError) Error() string { return "state: invalid pipeline decision: " + e.Message }

func decisionRefusal(format string, args ...any) error {
	return &DecisionError{Message: fmt.Sprintf(format, args...)}
}

// PipelineStep is one entry of a run's frozen plan as the owner sees it.
type PipelineStep struct {
	ID                    string `json:"id"`
	Title                 string `json:"title"`
	ObjectiveSummary      string `json:"objective_summary"`
	RequiredForCompletion bool   `json:"required_for_completion"`
	ApprovalAfterSuccess  bool   `json:"approval_after_success"`
}

// Required reports whether completion needs this step's evidence: an explicit
// marker or a configured human checkpoint (FS-14.R97).
func (p PipelineStep) Required() bool { return p.RequiredForCompletion || p.ApprovalAfterSuccess }

// PipelineSteps decodes the frozen template's plan. It reads only what routing
// needs, so the state layer does not depend on the template package.
func PipelineSteps(templateJSON string) ([]PipelineStep, error) {
	var snapshot struct {
		Stages []struct {
			ID                    string `json:"id"`
			Title                 string `json:"title"`
			Objective             string `json:"objective"`
			RequiredForCompletion bool   `json:"required_for_completion"`
			ApprovalAfterSuccess  bool   `json:"approval_after_success"`
		} `json:"stages"`
	}
	if err := json.Unmarshal([]byte(templateJSON), &snapshot); err != nil {
		return nil, fmt.Errorf("state: decode pipeline plan: %w", err)
	}
	steps := make([]PipelineStep, 0, len(snapshot.Stages))
	for _, st := range snapshot.Stages {
		if len(steps) == MaxPipelineSteps {
			break
		}
		steps = append(steps, PipelineStep{ID: st.ID, Title: st.Title, ObjectiveSummary: clipRunes(st.Objective, 240),
			RequiredForCompletion: st.RequiredForCompletion, ApprovalAfterSuccess: st.ApprovalAfterSuccess})
	}
	return steps, nil
}

// resolveStageDecisionTx validates the owner's decision against the frozen plan
// and the run's accepted evidence inside the result transaction, and returns
// the decision to record. A success without a decision follows the usual next
// step; the plan's end needs an explicit `complete` (TS-09.R66/R68).
func resolveStageDecisionTx(tx *sql.Tx, runID, taskID, templateJSON string, stageIndex int, outcome string, d *PipelineDecision) (*PipelineDecision, error) {
	steps, err := PipelineSteps(templateJSON)
	if err != nil {
		return nil, err
	}
	if d == nil {
		if outcome != OutcomeSuccess {
			// Failure/blocked is evidence for owner recovery, not a decision.
			return nil, nil
		}
		if stageIndex+1 >= len(steps) {
			return nil, decisionRefusal("this is the last step of the plan: report pipeline_decision {\"action\":\"complete\"} with a reason and required_checks evidence, or choose a step with action \"next\"")
		}
		return &PipelineDecision{Action: DecisionNext, StageID: steps[stageIndex+1].ID, Reason: "usual next step", Implicit: true}, nil
	}
	decision := *d
	decision.Action = strings.TrimSpace(decision.Action)
	decision.StageID = strings.TrimSpace(decision.StageID)
	decision.Reason = strings.TrimSpace(decision.Reason)
	decision.Question = strings.TrimSpace(decision.Question)
	decision.Implicit = false
	if decision.Reason == "" {
		return nil, decisionRefusal("pipeline_decision.reason is required")
	}
	if utf8.RuneCountInString(decision.Reason) > MaxDecisionReasonRunes || utf8.RuneCountInString(decision.Question) > MaxDecisionQuestionRunes {
		return nil, decisionRefusal("pipeline_decision.reason must fit %d characters and question %d", MaxDecisionReasonRunes, MaxDecisionQuestionRunes)
	}
	byID := map[string]PipelineStep{}
	for _, st := range steps {
		byID[st.ID] = st
	}
	switch decision.Action {
	case DecisionNext:
		if _, ok := byID[decision.StageID]; !ok {
			return nil, decisionRefusal("pipeline_decision.stage_id %q is not a step of this run's plan; choose one listed by get_assigned_task", decision.StageID)
		}
		if decision.Question != "" || len(decision.RequiredChecks) > 0 {
			return nil, decisionRefusal("a \"next\" decision takes only stage_id and reason")
		}
	case DecisionNeedsInput:
		if decision.Question == "" {
			return nil, decisionRefusal("a \"needs_input\" decision requires the concrete question the person must answer")
		}
		if decision.StageID != "" || len(decision.RequiredChecks) > 0 {
			return nil, decisionRefusal("a \"needs_input\" decision takes only question and reason")
		}
	case DecisionComplete:
		if outcome != OutcomeSuccess {
			return nil, decisionRefusal("\"complete\" requires outcome success; report failure or blocked without completing, or choose further work")
		}
		if decision.StageID != "" || decision.Question != "" {
			return nil, decisionRefusal("a \"complete\" decision takes only reason and required_checks")
		}
		if decision.RequiredChecks == nil {
			decision.RequiredChecks = map[string]string{}
		}
		if len(decision.RequiredChecks) > MaxPipelineSteps {
			return nil, decisionRefusal("required_checks may name at most %d steps", MaxPipelineSteps)
		}
		var currentStage string
		if err := tx.QueryRow(`SELECT stage_id FROM pipeline_stage_tasks WHERE task_id = ?`, taskID).Scan(&currentStage); err != nil {
			return nil, fmt.Errorf("state: read current stage: %w", err)
		}
		for stageID, sourceTaskID := range decision.RequiredChecks {
			if _, ok := byID[stageID]; !ok {
				return nil, decisionRefusal("required_checks names %q, which is not a step of this run's plan", stageID)
			}
			if err := checkEvidenceTx(tx, runID, taskID, currentStage, stageID, sourceTaskID); err != nil {
				return nil, err
			}
		}
		for _, st := range steps {
			if st.Required() && decision.RequiredChecks[st.ID] == "" {
				return nil, decisionRefusal("step %q is a required check: required_checks must name its accepted successful task (this run's %q result) before the run can complete", st.ID, st.ID)
			}
		}
	default:
		return nil, decisionRefusal("pipeline_decision.action must be next, complete or needs_input")
	}
	return &decision, nil
}

// checkEvidenceTx accepts a successful accepted task of this run for that step,
// or the result being committed now when the current visit is that step.
// Chuck checks authority and presence; the owner attests that the evidence
// still applies (TS-09.R68).
func checkEvidenceTx(tx *sql.Tx, runID, currentTaskID, currentStage, stageID, sourceTaskID string) error {
	if sourceTaskID == currentTaskID {
		if currentStage != stageID {
			return decisionRefusal("required_checks.%s names the current task, which is step %q", stageID, currentStage)
		}
		return nil
	}
	var outcome string
	err := tx.QueryRow(`SELECT COALESCE(t.outcome, '') FROM pipeline_stage_tasks p JOIN tasks t ON t.task_id = p.task_id
WHERE p.run_id = ? AND p.stage_id = ? AND p.task_id = ? AND t.state = ?`, runID, stageID, sourceTaskID, TaskFinished).Scan(&outcome)
	if errors.Is(err, sql.ErrNoRows) {
		return decisionRefusal("required_checks.%s must name a finished %q task of this run; %q is not one", stageID, stageID, sourceTaskID)
	}
	if err != nil {
		return fmt.Errorf("state: read required check evidence: %w", err)
	}
	if outcome != OutcomeSuccess {
		return decisionRefusal("required_checks.%s names %q, whose outcome is %s, not success", stageID, sourceTaskID, outcome)
	}
	return nil
}

// PipelineAssignment is the code-owned pipeline context get_assigned_task
// returns beside the historical instruction text (TS-09.R71).
type PipelineAssignment struct {
	RunID          string            `json:"run_id"`
	VisitSequence  int               `json:"visit_sequence"`
	CurrentStageID string            `json:"current_stage_id"`
	Steps          []PipelineStep    `json:"steps"`
	RequiredChecks map[string]string `json:"required_checks"`
	Recovery       PipelineRecovery  `json:"recovery"`
}

// PipelineRecovery is the run's automatic repair/revisit allowance.
type PipelineRecovery struct {
	Used        int    `json:"used"`
	Limit       int    `json:"limit"`
	PauseReason string `json:"pause_reason,omitempty"`
}

// PipelineAssignmentContext projects the plan, the latest accepted evidence
// for each required step and the recovery allowance for a stage task. Other
// tasks return ErrNotFound.
func (s *Store) PipelineAssignmentContext(taskID string) (PipelineAssignment, error) {
	var a PipelineAssignment
	var templateJSON, attention string
	err := s.db.QueryRow(`SELECT p.run_id, p.visit_sequence, p.stage_id, r.template_snapshot_json, r.recovery_rounds_used, r.attention_reason
FROM pipeline_stage_tasks p JOIN pipeline_runs r ON r.run_id = p.run_id WHERE p.task_id = ? AND p.execution_kind = ?`, taskID, StageExecutionAgent).
		Scan(&a.RunID, &a.VisitSequence, &a.CurrentStageID, &templateJSON, &a.Recovery.Used, &attention)
	if errors.Is(err, sql.ErrNoRows) {
		return PipelineAssignment{}, ErrNotFound
	}
	if err != nil {
		return PipelineAssignment{}, fmt.Errorf("state: read pipeline assignment: %w", err)
	}
	a.Recovery.Limit = PipelineRecoveryRounds
	if attention == PauseRecoveryExhausted {
		a.Recovery.PauseReason = attention
	}
	if a.Steps, err = PipelineSteps(templateJSON); err != nil {
		return PipelineAssignment{}, err
	}
	if a.RequiredChecks, err = s.requiredEvidence(a.RunID, a.Steps); err != nil {
		return PipelineAssignment{}, err
	}
	return a, nil
}

// RequiredCheckEvidence maps every required step of a run to its latest
// accepted successful task, empty when none exists yet.
func (s *Store) RequiredCheckEvidence(runID string) (map[string]string, error) {
	var templateJSON string
	if err := s.db.QueryRow(`SELECT template_snapshot_json FROM pipeline_runs WHERE run_id = ?`, runID).Scan(&templateJSON); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	steps, err := PipelineSteps(templateJSON)
	if err != nil {
		return nil, err
	}
	return s.requiredEvidence(runID, steps)
}

func (s *Store) requiredEvidence(runID string, steps []PipelineStep) (map[string]string, error) {
	out := map[string]string{}
	for _, st := range steps {
		if !st.Required() {
			continue
		}
		var source string
		err := s.db.QueryRow(`SELECT p.task_id FROM pipeline_stage_tasks p JOIN tasks t ON t.task_id = p.task_id
WHERE p.run_id = ? AND p.stage_id = ? AND t.state = ? AND t.outcome = ? ORDER BY p.visit_sequence DESC LIMIT 1`, runID, st.ID, TaskFinished, OutcomeSuccess).Scan(&source)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("state: read required check evidence: %w", err)
		}
		out[st.ID] = source
	}
	return out, nil
}

// PauseRecoveryExhausted is the attention reason of a run whose automatic
// recovery allowance is spent; it is recoverable, never a final failure.
const PauseRecoveryExhausted = "recovery_exhausted"

// ApproveStageVisit records the person's approval of one accepted visit. A
// question answer or ordinary chat never calls this (FS-14.R97/R99).
func (s *Store) ApproveStageVisit(taskID string) error {
	_, err := s.db.Exec(`UPDATE pipeline_stage_tasks SET approved_at = ? WHERE task_id = ? AND approved_at = ''`, formatTime(timeNow()), taskID)
	return err
}

// StageVisitApproved reports whether a visit's approval was recorded.
func (s *Store) StageVisitApproved(taskID string) (bool, error) {
	var at string
	if err := s.db.QueryRow(`SELECT approved_at FROM pipeline_stage_tasks WHERE task_id = ?`, taskID).Scan(&at); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, ErrNotFound
		}
		return false, err
	}
	return at != "", nil
}

// StageDecision reads the decision recorded with a stage result, nil if none.
func (s *Store) StageDecision(taskID string) (*PipelineDecision, error) {
	var raw string
	if err := s.db.QueryRow(`SELECT decision_json FROM pipeline_stage_tasks WHERE task_id = ?`, taskID).Scan(&raw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if raw == "" {
		return nil, nil
	}
	var d PipelineDecision
	if err := json.Unmarshal([]byte(raw), &d); err != nil {
		return nil, fmt.Errorf("state: decode stage decision: %w", err)
	}
	return &d, nil
}
