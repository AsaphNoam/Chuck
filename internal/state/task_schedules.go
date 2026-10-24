package state

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/AsaphNoam/Chuck/internal/recurrence"
)

// Schedule states and last-occurrence dispositions (FS-16.R53–R57).
const (
	ScheduleActive   = "active"
	SchedulePaused   = "paused"
	ScheduleFinished = "finished"

	OccurrenceCreated = "created"
	OccurrenceMissed  = "missed"
	OccurrencePaused  = "paused"
)

// scheduleEnumerationCap bounds how many elapsed occurrences one due pass or
// resume enumerates. It only paces work: a schedule left behind catches up on
// later passes (TS-10.R57, R60).
const scheduleEnumerationCap = 512

// ErrScheduleConflict is returned when a schedule changed under the caller.
var ErrScheduleConflict = errors.New("state: schedule revision conflict")

// ErrScheduleRule refuses a malformed rule or one with no future occurrence.
var ErrScheduleRule = errors.New("state: invalid schedule rule")

// ErrScheduleState refuses pause/resume from a state that does not allow it.
var ErrScheduleState = errors.New("state: schedule cannot change from its current state")

// TaskSchedule is a durable project-scoped template that creates one ordinary
// task at each occurrence of its rule (FS-16.R53, TS-10.R57).
type TaskSchedule struct {
	ScheduleID  string `json:"schedule_id"`
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
	Group         string `json:"group,omitempty"`

	Rule       recurrence.Rule `json:"rule"`
	RuleText   string          `json:"rule_text"`
	OnTimeOnly bool            `json:"run_only_on_time"`

	State       string `json:"state"`
	PauseReason string `json:"pause_reason,omitempty"`

	CreatedByKind       string `json:"created_by_kind"`
	CreatedByAgentID    string `json:"created_by_agent_id,omitempty"`
	CreatedByGeneration string `json:"created_by_generation,omitempty"`

	NextDueAt *time.Time `json:"next_due_at,omitempty"`
	// NextDueWall names NextDueAt's occurrence in recurrence.WallLayout.
	NextDueWall        string     `json:"-"`
	OccurrencesElapsed int        `json:"occurrences_elapsed"`
	LastOccurrenceAt   *time.Time `json:"last_occurrence_at,omitempty"`
	LastDisposition    string     `json:"last_disposition,omitempty"`
	LastTaskID         string     `json:"last_task_id,omitempty"`

	Revision  int       `json:"revision"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Attachments []TaskAttachment `json:"attachments"`
}

// NewScheduleID mints a durable opaque schedule id.
func NewScheduleID() (string, error) {
	var bytes [8]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", fmt.Errorf("state: mint schedule id: %w", err)
	}
	return "sc_" + hex.EncodeToString(bytes[:]), nil
}

// CreateSchedule normalizes the rule, computes the first occurrence after now
// in loc, and writes the schedule and its template attachments together. An
// agent creator may attach only what it can already read (FS-16.R56).
func (s *Store) CreateSchedule(sched TaskSchedule, creatorAgentID string, loc *time.Location) (TaskSchedule, error) {
	rule, err := sched.Rule.Normalize()
	if err != nil {
		return TaskSchedule{}, fmt.Errorf("%w: %v", ErrScheduleRule, err)
	}
	now := timeNow()
	next, wall, ok := firstOccurrence(rule, now, 0, loc)
	if !ok {
		return TaskSchedule{}, fmt.Errorf("%w: the rule has no future occurrence", ErrScheduleRule)
	}
	sched.Rule, sched.State, sched.NextDueAt = rule, ScheduleActive, &next
	sched.Revision, sched.CreatedAt, sched.UpdatedAt = 1, now, now
	ruleJSON, err := json.Marshal(rule)
	if err != nil {
		return TaskSchedule{}, fmt.Errorf("state: encode schedule rule: %w", err)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return TaskSchedule{}, fmt.Errorf("state: begin create schedule: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`
INSERT INTO task_schedules(schedule_id, project, display_name, instruction, target_kind,
  target_agent_id, role, backend, model, effort, fast, group_name, rule_json, on_time_only, state,
  created_by_kind, created_by_agent_id, created_by_generation, next_due_at, next_due_wall, revision,
  created_at, updated_at)
VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sched.ScheduleID, sched.Project, sched.DisplayName, sched.Instruction, sched.TargetKind,
		sched.TargetAgentID, sched.Role, sched.Backend, sched.Model, sched.Effort, sched.Fast, sched.Group,
		string(ruleJSON), sched.OnTimeOnly, sched.State, sched.CreatedByKind, sched.CreatedByAgentID,
		sched.CreatedByGeneration, formatTime(next), wall, sched.Revision, formatTime(now), formatTime(now)); err != nil {
		return TaskSchedule{}, fmt.Errorf("state: insert schedule: %w", err)
	}
	if err := insertScheduleAttachments(tx, sched.ScheduleID, sched.Attachments, creatorAgentID); err != nil {
		return TaskSchedule{}, err
	}
	if err := tx.Commit(); err != nil {
		return TaskSchedule{}, fmt.Errorf("state: commit create schedule: %w", err)
	}
	return s.ReadSchedule(sched.ScheduleID)
}

func insertScheduleAttachments(tx *sql.Tx, scheduleID string, attachments []TaskAttachment, creatorAgentID string) error {
	for _, attachment := range attachments {
		if err := checkTaskAttachment(tx, attachment.ContextRefID, creatorAgentID); err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT INTO task_schedule_attachments(schedule_id, context_ref_id, label, description) VALUES(?, ?, ?, ?)`,
			scheduleID, attachment.ContextRefID, attachment.Label, attachment.Description); err != nil {
			return fmt.Errorf("state: attach schedule context: %w", err)
		}
	}
	return nil
}

// firstOccurrence is the next occurrence strictly after `after`, or none when
// the rule's date end has passed or its count is already used up.
func firstOccurrence(rule recurrence.Rule, after time.Time, elapsed int, loc *time.Location) (time.Time, string, bool) {
	if rule.EndKind == recurrence.EndCount && elapsed >= rule.EndCount {
		return time.Time{}, "", false
	}
	return rule.NextWall(after, loc)
}

// UpdateSchedule replaces the template and rule under the observed revision.
// Only future occurrences change: the elapsed count is kept and the next
// occurrence is recomputed from now. A finished schedule whose new rule has a
// future occurrence becomes active again (FS-16.R56, TS-10.R59).
func (s *Store) UpdateSchedule(edit TaskSchedule, revision int, creatorAgentID string, loc *time.Location) (TaskSchedule, error) {
	rule, err := edit.Rule.Normalize()
	if err != nil {
		return TaskSchedule{}, fmt.Errorf("%w: %v", ErrScheduleRule, err)
	}
	current, err := s.ReadSchedule(edit.ScheduleID)
	if err != nil {
		return TaskSchedule{}, err
	}
	now := timeNow()
	next, wall, ok := firstOccurrence(rule, now, current.OccurrencesElapsed, loc)
	nextState := current.State
	if !ok {
		nextState = ScheduleFinished
	} else if nextState == ScheduleFinished {
		nextState = ScheduleActive
	}
	var nextDue any
	if ok {
		nextDue = formatTime(next)
	}
	ruleJSON, err := json.Marshal(rule)
	if err != nil {
		return TaskSchedule{}, fmt.Errorf("state: encode schedule rule: %w", err)
	}
	tx, err := s.db.Begin()
	if err != nil {
		return TaskSchedule{}, fmt.Errorf("state: begin edit schedule: %w", err)
	}
	defer tx.Rollback()
	res, err := tx.Exec(`
UPDATE task_schedules SET display_name = ?, instruction = ?, target_kind = ?, target_agent_id = ?,
  role = ?, backend = ?, model = ?, effort = ?, fast = ?, group_name = ?, rule_json = ?, on_time_only = ?,
  state = ?, next_due_at = ?, next_due_wall = ?, revision = revision + 1, updated_at = ?
WHERE schedule_id = ? AND revision = ?`,
		edit.DisplayName, edit.Instruction, edit.TargetKind, edit.TargetAgentID, edit.Role, edit.Backend,
		edit.Model, edit.Effort, edit.Fast, edit.Group, string(ruleJSON), edit.OnTimeOnly, nextState, nextDue,
		wall, formatTime(now), edit.ScheduleID, revision)
	if err != nil {
		return TaskSchedule{}, fmt.Errorf("state: edit schedule: %w", err)
	}
	if n, err := res.RowsAffected(); err != nil {
		return TaskSchedule{}, fmt.Errorf("state: edit schedule rows: %w", err)
	} else if n == 0 {
		return TaskSchedule{}, ErrScheduleConflict
	}
	if _, err := tx.Exec(`DELETE FROM task_schedule_attachments WHERE schedule_id = ?`, edit.ScheduleID); err != nil {
		return TaskSchedule{}, fmt.Errorf("state: clear schedule attachments: %w", err)
	}
	if err := insertScheduleAttachments(tx, edit.ScheduleID, edit.Attachments, creatorAgentID); err != nil {
		return TaskSchedule{}, err
	}
	if err := tx.Commit(); err != nil {
		return TaskSchedule{}, fmt.Errorf("state: commit edit schedule: %w", err)
	}
	return s.ReadSchedule(edit.ScheduleID)
}

// PauseSchedule stops an active schedule from creating occurrences. The
// occurrences it passes over are counted as paused when it resumes.
func (s *Store) PauseSchedule(scheduleID, reason string) (TaskSchedule, error) {
	res, err := s.db.Exec(`
UPDATE task_schedules SET state = ?, pause_reason = ?, revision = revision + 1, updated_at = ?
WHERE schedule_id = ? AND state = ?`, SchedulePaused, reason, formatTime(timeNow()), scheduleID, ScheduleActive)
	if err != nil {
		return TaskSchedule{}, fmt.Errorf("state: pause schedule: %w", err)
	}
	return s.afterScheduleTransition(scheduleID, res)
}

// ResumeSchedule reactivates a paused schedule without replaying anything:
// every occurrence that came due while paused is counted as paused, and the
// next occurrence is the first one after now (FS-16.R56, TS-10.R59). The
// caller revalidates the target first.
func (s *Store) ResumeSchedule(scheduleID string, loc *time.Location) (TaskSchedule, error) {
	sched, err := s.ReadSchedule(scheduleID)
	if err != nil {
		return TaskSchedule{}, err
	}
	if sched.State != SchedulePaused {
		return TaskSchedule{}, ErrScheduleState
	}
	now := timeNow()
	elapsed, last := sched.OccurrencesElapsed, sched.LastOccurrenceAt
	disposition := sched.LastDisposition
	cursor, _ := sched.pendingDue(loc)
	// Count what the pause passed over; the count also ends a count rule.
	for passed := 0; cursor != nil && !cursor.After(now) && passed < 100*scheduleEnumerationCap; passed++ {
		if sched.Rule.EndKind == recurrence.EndCount && elapsed >= sched.Rule.EndCount {
			break
		}
		at := *cursor
		elapsed, last, disposition = elapsed+1, &at, OccurrencePaused
		next, ok := sched.Rule.Next(at, loc)
		if !ok {
			cursor = nil
			break
		}
		cursor = &next
	}
	state, nextDue, nextWall := ScheduleActive, any(nil), ""
	if next, wall, ok := firstOccurrence(sched.Rule, now, elapsed, loc); ok {
		nextDue, nextWall = formatTime(next), wall
	} else {
		state = ScheduleFinished
	}
	res, err := s.db.Exec(`
UPDATE task_schedules SET state = ?, pause_reason = '', next_due_at = ?, next_due_wall = ?,
  occurrences_elapsed = ?, last_occurrence_at = ?, last_disposition = ?, revision = revision + 1, updated_at = ?
WHERE schedule_id = ? AND revision = ? AND state = ?`,
		state, nextDue, nextWall, elapsed, formatOptionalTime(last), disposition, formatTime(now),
		scheduleID, sched.Revision, SchedulePaused)
	if err != nil {
		return TaskSchedule{}, fmt.Errorf("state: resume schedule: %w", err)
	}
	return s.afterScheduleTransition(scheduleID, res)
}

func (s *Store) afterScheduleTransition(scheduleID string, res sql.Result) (TaskSchedule, error) {
	n, err := res.RowsAffected()
	if err != nil {
		return TaskSchedule{}, fmt.Errorf("state: schedule transition rows: %w", err)
	}
	if n == 0 {
		if _, err := s.ReadSchedule(scheduleID); err != nil {
			return TaskSchedule{}, err
		}
		return TaskSchedule{}, ErrScheduleState
	}
	return s.ReadSchedule(scheduleID)
}

// DeleteSchedule removes the schedule and its template attachments. Every task
// it created stays an ordinary task (FS-16.R56, TS-10.R58).
func (s *Store) DeleteSchedule(scheduleID string) error {
	res, err := s.db.Exec(`DELETE FROM task_schedules WHERE schedule_id = ?`, scheduleID)
	if err != nil {
		return fmt.Errorf("state: delete schedule: %w", err)
	}
	if n, err := res.RowsAffected(); err != nil {
		return fmt.Errorf("state: delete schedule rows: %w", err)
	} else if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ScheduleOutcome is what one due pass did to one schedule: the schedule after
// the pass and the occurrence task it created, if any.
type ScheduleOutcome struct {
	Schedule TaskSchedule
	Task     *Task
}

// ProcessDueSchedules handles at most limit active schedules whose next
// occurrence has come, each in its own transaction (TS-10.R57). unavailable
// returns why the schedule's target cannot run now, or "" when it can; a
// non-empty reason pauses the schedule instead of creating a task (R59).
func (s *Store) ProcessDueSchedules(limit int, loc *time.Location, unavailable func(TaskSchedule) string) ([]ScheduleOutcome, error) {
	if s.scheduleZone.Load() != loc {
		if err := s.rebaseSchedules(loc); err != nil {
			return nil, err
		}
		s.scheduleZone.Store(loc)
	}
	rows, err := s.db.Query(`
SELECT schedule_id FROM task_schedules WHERE state = ? AND next_due_at <= ?
ORDER BY next_due_at, schedule_id LIMIT ?`, ScheduleActive, formatTime(timeNow()), limit)
	if err != nil {
		return nil, fmt.Errorf("state: list due schedules: %w", err)
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, fmt.Errorf("state: scan due schedule: %w", err)
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, fmt.Errorf("state: iterate due schedules: %w", err)
	}
	// Each schedule is independently failable: one bad row is reported and
	// the rest still run (INV §7).
	var outcomes []ScheduleOutcome
	var firstErr error
	for _, id := range ids {
		outcome, handled, err := s.processDueSchedule(id, loc, unavailable)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("schedule %s: %w", id, err)
			}
			continue
		}
		if handled {
			outcomes = append(outcomes, outcome)
		}
	}
	return outcomes, firstErr
}

// pendingDue is the unhandled occurrence's instant in loc and its name. A
// row written before names were stored is named from its instant in loc.
func (sched TaskSchedule) pendingDue(loc *time.Location) (*time.Time, string) {
	if sched.NextDueAt == nil {
		return nil, ""
	}
	wall := sched.NextDueWall
	if wall == "" {
		wall = sched.NextDueAt.In(loc).Format(recurrence.WallLayout)
	}
	at, err := recurrence.WallAt(wall, loc)
	if err != nil {
		return sched.NextDueAt, wall
	}
	return &at, wall
}

// rebaseSchedules moves each pending occurrence to its instant in loc, so a
// restart in another zone neither runs it at the old zone's time nor replays
// a handled one (FS-16.R54/R58, TS-10.R56).
func (s *Store) rebaseSchedules(loc *time.Location) error {
	rows, err := s.db.Query(`SELECT schedule_id, next_due_at, next_due_wall FROM task_schedules
WHERE state IN (?, ?) AND next_due_at IS NOT NULL`, ScheduleActive, SchedulePaused)
	if err != nil {
		return fmt.Errorf("state: list schedules to rebase: %w", err)
	}
	type move struct {
		id   string
		at   time.Time
		wall string
	}
	var moves []move
	for rows.Next() {
		var sched TaskSchedule
		var nextDue string
		if err := rows.Scan(&sched.ScheduleID, &nextDue, &sched.NextDueWall); err != nil {
			rows.Close()
			return fmt.Errorf("state: scan schedule to rebase: %w", err)
		}
		stored, err := parseTime(nextDue)
		if err != nil {
			// One bad row is left for its own pass to report (INV §7).
			continue
		}
		sched.NextDueAt = &stored
		if at, wall := sched.pendingDue(loc); !at.Equal(stored) || wall != sched.NextDueWall {
			moves = append(moves, move{sched.ScheduleID, *at, wall})
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return fmt.Errorf("state: iterate schedules to rebase: %w", err)
	}
	for _, m := range moves {
		if _, err := s.db.Exec(`UPDATE task_schedules SET next_due_at = ?, next_due_wall = ? WHERE schedule_id = ?`,
			formatTime(m.at), m.wall, m.id); err != nil {
			return fmt.Errorf("state: rebase schedule %s: %w", m.id, err)
		}
	}
	return nil
}

func (s *Store) processDueSchedule(id string, loc *time.Location, unavailable func(TaskSchedule) string) (ScheduleOutcome, bool, error) {
	sched, err := s.ReadSchedule(id)
	if err != nil || sched.State != ScheduleActive || sched.NextDueAt == nil {
		return ScheduleOutcome{}, false, err
	}
	now := timeNow()
	// Enumerate every elapsed occurrence up to now, bounded by the rule's end
	// and the per-pass cap.
	var due []time.Time
	start, wall := sched.pendingDue(loc)
	cursor, more := *start, true
	for more && !cursor.After(now) && len(due) < scheduleEnumerationCap {
		if sched.Rule.EndKind == recurrence.EndCount && sched.OccurrencesElapsed+len(due) >= sched.Rule.EndCount {
			more = false
			break
		}
		due = append(due, cursor)
		cursor, wall, more = sched.Rule.NextWall(cursor, loc)
	}
	if len(due) == 0 {
		return ScheduleOutcome{}, false, nil
	}
	elapsed := sched.OccurrencesElapsed + len(due)
	if more && sched.Rule.EndKind == recurrence.EndCount && elapsed >= sched.Rule.EndCount {
		more = false
	}
	latest := due[len(due)-1]
	late := now.Sub(latest) > OnTimeWindow
	// A pass the cap cut short has not reached the most recent occurrence yet,
	// so only a later pass may decide which one creates the task.
	caughtUp := !more || cursor.After(now)
	create := caughtUp && !(late && sched.OnTimeOnly)
	disposition, state, reason := OccurrenceMissed, ScheduleActive, ""
	if create {
		if reason = unavailable(sched); reason != "" {
			create, disposition, state = false, OccurrencePaused, SchedulePaused
		} else {
			disposition = OccurrenceCreated
		}
	}
	if !more && state == ScheduleActive {
		state = ScheduleFinished
	}
	var nextDue any
	nextWall := ""
	if more {
		nextDue, nextWall = formatTime(cursor), wall
	}

	// The store has one connection, so the id is minted before the
	// transaction holds it.
	taskID := ""
	if create {
		if taskID, err = s.NewTaskID(); err != nil {
			return ScheduleOutcome{}, false, err
		}
	}
	tx, err := s.db.Begin()
	if err != nil {
		return ScheduleOutcome{}, false, fmt.Errorf("state: begin due schedule: %w", err)
	}
	defer tx.Rollback()
	var task *Task
	lastTaskID := sched.LastTaskID
	if create {
		created, err := insertOccurrenceTask(tx, taskID, sched, latest, late, now)
		if err != nil {
			return ScheduleOutcome{}, false, err
		}
		task, lastTaskID = &created, created.TaskID
	}
	// Task creation and schedule advancement commit together under the
	// observed revision, so no occurrence is created twice (FS-16.R58, INV §15).
	res, err := tx.Exec(`
UPDATE task_schedules SET state = ?, pause_reason = ?, next_due_at = ?, next_due_wall = ?,
  occurrences_elapsed = ?, last_occurrence_at = ?, last_disposition = ?, last_task_id = ?,
  revision = revision + 1, updated_at = ?
WHERE schedule_id = ? AND revision = ? AND state = ?`,
		state, reason, nextDue, nextWall, elapsed, formatTime(latest), disposition, lastTaskID, formatTime(now),
		id, sched.Revision, ScheduleActive)
	if err != nil {
		return ScheduleOutcome{}, false, fmt.Errorf("state: advance schedule: %w", err)
	}
	if n, err := res.RowsAffected(); err != nil {
		return ScheduleOutcome{}, false, fmt.Errorf("state: advance schedule rows: %w", err)
	} else if n == 0 {
		return ScheduleOutcome{}, false, nil
	}
	if err := tx.Commit(); err != nil {
		return ScheduleOutcome{}, false, fmt.Errorf("state: commit due schedule: %w", err)
	}
	after, err := s.ReadSchedule(id)
	if err != nil {
		return ScheduleOutcome{}, false, err
	}
	if task != nil {
		read, err := s.ReadTask(task.TaskID)
		if err != nil {
			return ScheduleOutcome{}, false, err
		}
		task = &read
	}
	return ScheduleOutcome{Schedule: after, Task: task}, true, nil
}

// insertOccurrenceTask writes the schedule's ordinary task for one occurrence:
// no arms, so ready at once, with the schedule's creator as its creator of
// record (FS-16.R53, TS-10.R58).
func insertOccurrenceTask(tx *sql.Tx, id string, sched TaskSchedule, occurrence time.Time, late bool, now time.Time) (Task, error) {
	task := Task{
		TaskID: id, Project: sched.Project, DisplayName: sched.DisplayName, Instruction: sched.Instruction,
		TargetKind: sched.TargetKind, TargetAgentID: sched.TargetAgentID, Role: sched.Role,
		Backend: sched.Backend, Model: sched.Model, Effort: sched.Effort, Fast: sched.Fast, Group: sched.Group,
		State: TaskReady, CreatedByKind: sched.CreatedByKind, CreatedByAgentID: sched.CreatedByAgentID,
		CreatedByGeneration: sched.CreatedByGeneration, ScheduleID: sched.ScheduleID,
		ScheduledFor: &occurrence, LateStart: late, Revision: 1, ReadyAt: &now, CreatedAt: now, UpdatedAt: now,
	}
	if err := insertTaskRowTx(tx, task); err != nil {
		return Task{}, err
	}
	for _, attachment := range sched.Attachments {
		if _, err := tx.Exec(`INSERT INTO task_attachments(task_id, context_ref_id, label, description, created_at) VALUES(?, ?, ?, ?, ?)`,
			id, attachment.ContextRefID, attachment.Label, attachment.Description, formatTime(now)); err != nil {
			return Task{}, fmt.Errorf("state: attach occurrence context: %w", err)
		}
	}
	return task, nil
}

const scheduleSelect = `
SELECT schedule_id, project, display_name, instruction, target_kind, target_agent_id, role, backend,
  model, effort, fast, group_name, rule_json, on_time_only, state, pause_reason, created_by_kind,
  created_by_agent_id, created_by_generation, next_due_at, next_due_wall, occurrences_elapsed, last_occurrence_at,
  last_disposition, last_task_id, revision, created_at, updated_at
FROM task_schedules`

// ReadSchedule returns one schedule with its template attachments.
func (s *Store) ReadSchedule(scheduleID string) (TaskSchedule, error) {
	sched, err := scanSchedule(s.db.QueryRow(scheduleSelect+` WHERE schedule_id = ?`, scheduleID))
	if errors.Is(err, sql.ErrNoRows) {
		return TaskSchedule{}, ErrNotFound
	}
	if err != nil {
		return TaskSchedule{}, err
	}
	return s.withScheduleAttachments(sched)
}

// ListSchedules returns a project's schedules oldest first.
func (s *Store) ListSchedules(project string) ([]TaskSchedule, error) {
	rows, err := s.db.Query(scheduleSelect+` WHERE project = ? ORDER BY created_at, schedule_id`, project)
	if err != nil {
		return nil, fmt.Errorf("state: list schedules: %w", err)
	}
	defer rows.Close()
	out := []TaskSchedule{}
	for rows.Next() {
		sched, err := scanSchedule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, sched)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("state: iterate schedules: %w", err)
	}
	for i := range out {
		if out[i], err = s.withScheduleAttachments(out[i]); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (s *Store) withScheduleAttachments(sched TaskSchedule) (TaskSchedule, error) {
	rows, err := s.db.Query(`SELECT context_ref_id, label, description FROM task_schedule_attachments
WHERE schedule_id = ? ORDER BY context_ref_id`, sched.ScheduleID)
	if err != nil {
		return TaskSchedule{}, fmt.Errorf("state: list schedule attachments: %w", err)
	}
	defer rows.Close()
	sched.Attachments = []TaskAttachment{}
	for rows.Next() {
		var a TaskAttachment
		if err := rows.Scan(&a.ContextRefID, &a.Label, &a.Description); err != nil {
			return TaskSchedule{}, fmt.Errorf("state: scan schedule attachment: %w", err)
		}
		sched.Attachments = append(sched.Attachments, a)
	}
	if err := rows.Err(); err != nil {
		return TaskSchedule{}, fmt.Errorf("state: iterate schedule attachments: %w", err)
	}
	return sched, nil
}

func scanSchedule(row rowScanner) (TaskSchedule, error) {
	var sched TaskSchedule
	var ruleJSON, createdAt, updatedAt string
	var nextDue, lastAt sql.NullString
	if err := row.Scan(&sched.ScheduleID, &sched.Project, &sched.DisplayName, &sched.Instruction,
		&sched.TargetKind, &sched.TargetAgentID, &sched.Role, &sched.Backend, &sched.Model, &sched.Effort,
		&sched.Fast, &sched.Group, &ruleJSON, &sched.OnTimeOnly, &sched.State, &sched.PauseReason,
		&sched.CreatedByKind, &sched.CreatedByAgentID, &sched.CreatedByGeneration, &nextDue,
		&sched.NextDueWall, &sched.OccurrencesElapsed, &lastAt, &sched.LastDisposition, &sched.LastTaskID, &sched.Revision,
		&createdAt, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return TaskSchedule{}, err
		}
		return TaskSchedule{}, fmt.Errorf("state: scan schedule: %w", err)
	}
	if err := json.Unmarshal([]byte(ruleJSON), &sched.Rule); err != nil {
		return TaskSchedule{}, fmt.Errorf("state: decode schedule rule: %w", err)
	}
	sched.RuleText = sched.Rule.Describe()
	var err error
	if sched.NextDueAt, err = parseOptionalTime(nextDue); err != nil {
		return TaskSchedule{}, wrapTimeErr("schedule next_due_at", err)
	}
	if sched.LastOccurrenceAt, err = parseOptionalTime(lastAt); err != nil {
		return TaskSchedule{}, wrapTimeErr("schedule last_occurrence_at", err)
	}
	if sched.CreatedAt, err = parseTime(createdAt); err != nil {
		return TaskSchedule{}, wrapTimeErr("schedule created_at", err)
	}
	if sched.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return TaskSchedule{}, wrapTimeErr("schedule updated_at", err)
	}
	return sched, nil
}
