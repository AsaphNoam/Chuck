package state

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"
)

const MaxQuotaOwnerBytes = 1024
const MaxQuotaDueBatch = 8

const (
	QuotaDisabled       = "disabled"
	QuotaUnknownReset   = "unknown_reset"
	QuotaScheduled      = "scheduled"
	QuotaDeferred       = "deferred"
	QuotaClaimed        = "claimed"
	QuotaContinued      = "continued"
	QuotaCancelled      = "cancelled"
	QuotaNeedsAttention = "needs_attention"
)

var (
	ErrQuotaConflict = errors.New("state: quota episode conflict")
	ErrQuotaOwner    = errors.New("state: invalid quota owner")
)

// QuotaOwner is a closed reference to work that already exists. Fields are
// mutually constrained by Kind; this descriptor never copies owner payload.
type QuotaOwner struct {
	Kind      string `json:"kind"`
	TaskID    string `json:"task_id,omitempty"`
	RunID     string `json:"run_id,omitempty"`
	RoomID    string `json:"room_id,omitempty"`
	AttemptID string `json:"attempt_id,omitempty"`
	Turn      string `json:"turn,omitempty"`
}

type QuotaObservation struct {
	AgentID          string
	EpisodeID        string
	SourceGeneration string
	ExecutingTurn    string
	Owner            QuotaOwner
	ObservedAt       time.Time
	ResetAt          *time.Time
	DueAt            *time.Time
	State            string // disabled, unknown_reset, or scheduled
}

type QuotaIntent struct {
	AgentID             string     `json:"agent_id"`
	EpisodeID           string     `json:"episode_id"`
	Revision            int64      `json:"revision"`
	SourceGeneration    string     `json:"source_generation"`
	ExecutingTurn       string     `json:"executing_turn"`
	Owner               QuotaOwner `json:"owner"`
	ObservedAt          time.Time  `json:"observed_at"`
	ResetAt             *time.Time `json:"reset_at,omitempty"`
	DueAt               *time.Time `json:"due_at,omitempty"`
	State               string     `json:"state"`
	ReasonCode          string     `json:"reason_code,omitempty"`
	AttemptToken        string     `json:"attempt_token,omitempty"`
	ProviderStarted     bool       `json:"provider_started"`
	ContinuedGeneration string     `json:"continued_generation,omitempty"`
	ContinuedTurn       string     `json:"continued_turn,omitempty"`
	NoticeKey           string     `json:"notice_key,omitempty"`
	NoticeConfirmed     bool       `json:"notice_confirmed"`
	SuccessConfirmed    bool       `json:"success_confirmed"`
	ManualGeneration    string     `json:"manual_generation,omitempty"`
	ManualTurn          string     `json:"manual_turn,omitempty"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

func validateQuotaOwner(owner QuotaOwner) ([]byte, error) {
	blankExcept := func(allow ...string) bool {
		allowed := map[string]bool{}
		for _, field := range allow {
			allowed[field] = true
		}
		values := map[string]string{"task_id": owner.TaskID, "run_id": owner.RunID, "room_id": owner.RoomID, "attempt_id": owner.AttemptID, "turn": owner.Turn}
		for name, value := range values {
			if !allowed[name] && value != "" {
				return false
			}
		}
		return true
	}
	valid := false
	switch owner.Kind {
	case "ordinary":
		valid = blankExcept()
	case "task":
		valid = owner.TaskID != "" && blankExcept("task_id")
	case "pipeline":
		valid = owner.TaskID != "" && owner.RunID != "" && blankExcept("task_id", "run_id")
	case "room":
		valid = owner.RoomID != "" && owner.AttemptID != "" && blankExcept("room_id", "attempt_id", "turn") &&
			(owner.Turn == ThinkTankTurnOpening || owner.Turn == ThinkTankTurnDiscussion || owner.Turn == ThinkTankTurnClosing || owner.Turn == ThinkTankTurnJudge)
	}
	if !valid {
		return nil, ErrQuotaOwner
	}
	b, err := json.Marshal(owner)
	if err != nil || len(b) > MaxQuotaOwnerBytes {
		return nil, ErrQuotaOwner
	}
	return b, nil
}

func validateQuotaOwnerTx(tx *sql.Tx, owner QuotaOwner) error {
	var found int
	switch owner.Kind {
	case "ordinary":
		return nil
	case "task":
		if err := tx.QueryRow(`SELECT 1 FROM tasks WHERE task_id=?`, owner.TaskID).Scan(&found); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrQuotaOwner
			}
			return fmt.Errorf("state: validate quota task owner: %w", err)
		}
	case "pipeline":
		if err := tx.QueryRow(`SELECT 1 FROM pipeline_stage_tasks WHERE task_id=? AND run_id=?`, owner.TaskID, owner.RunID).Scan(&found); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrQuotaOwner
			}
			return fmt.Errorf("state: validate quota pipeline owner: %w", err)
		}
	case "room":
		if err := tx.QueryRow(`SELECT 1 FROM think_tank_attempts WHERE room_id=? AND attempt_id=? AND turn=?`, owner.RoomID, owner.AttemptID, owner.Turn).Scan(&found); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrQuotaOwner
			}
			return fmt.Errorf("state: validate quota room owner: %w", err)
		}
	default:
		return ErrQuotaOwner
	}
	return nil
}

// ObserveQuota inserts a new episode or atomically replaces the current one.
// Re-observing the same episode is idempotent and preserves all progress.
func (s *Store) ObserveQuota(in QuotaObservation) (QuotaIntent, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return QuotaIntent{}, fmt.Errorf("state: begin quota observation: %w", err)
	}
	defer tx.Rollback()
	q, err := observeQuotaTx(tx, in)
	if err != nil {
		return QuotaIntent{}, err
	}
	if err := tx.Commit(); err != nil {
		return QuotaIntent{}, err
	}
	return q, nil
}

// observeQuotaTx validates and writes one episode within the caller's existing
// transaction so an owner settlement and its recovery obligation commit together.
func observeQuotaTx(tx *sql.Tx, in QuotaObservation) (QuotaIntent, error) {
	if in.AgentID == "" || in.EpisodeID == "" || in.SourceGeneration == "" || in.ExecutingTurn == "" {
		return QuotaIntent{}, ErrQuotaConflict
	}
	owner, err := validateQuotaOwner(in.Owner)
	if err != nil {
		return QuotaIntent{}, err
	}
	if in.State != QuotaDisabled && in.State != QuotaUnknownReset && in.State != QuotaScheduled {
		return QuotaIntent{}, ErrQuotaConflict
	}
	if in.State == QuotaScheduled && (in.ResetAt == nil || in.DueAt == nil) {
		return QuotaIntent{}, ErrQuotaConflict
	}
	if in.State == QuotaScheduled && (!in.ResetAt.After(in.ObservedAt) || in.DueAt.Before(*in.ResetAt)) {
		return QuotaIntent{}, ErrQuotaConflict
	}
	if in.State != QuotaScheduled && in.DueAt != nil {
		return QuotaIntent{}, ErrQuotaConflict
	}
	if in.ObservedAt.IsZero() {
		return QuotaIntent{}, ErrQuotaConflict
	}
	if err := validateQuotaOwnerTx(tx, in.Owner); err != nil {
		return QuotaIntent{}, err
	}
	var oldEpisode string
	var revision int64
	err = tx.QueryRow(`SELECT episode_id, revision FROM quota_intents WHERE agent_id = ?`, in.AgentID).Scan(&oldEpisode, &revision)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return QuotaIntent{}, fmt.Errorf("state: read current quota episode: %w", err)
	}
	if err == nil && oldEpisode == in.EpisodeID {
		// Later trustworthy evidence may fill an unknown reset for this exact
		// interrupted turn. Cancellation/disable/attempt receipts are never
		// resurrected by a duplicate observation.
		if in.State == QuotaScheduled {
			if _, err := tx.Exec(`UPDATE quota_intents SET state='scheduled', revision=revision+1, reset_at=?, due_at=?, updated_at=?
WHERE agent_id=? AND episode_id=? AND revision=? AND state='unknown_reset' AND source_generation=? AND executing_turn=?`,
				formatOptionalTime(in.ResetAt), formatOptionalTime(in.DueAt), formatTime(timeNow()), in.AgentID, in.EpisodeID, revision, in.SourceGeneration, in.ExecutingTurn); err != nil {
				return QuotaIntent{}, err
			}
		}
		return scanQuota(tx.QueryRow(quotaSelect+` WHERE agent_id = ?`, in.AgentID))
	}
	if err == nil {
		revision++
	} else {
		revision = 1
	}
	now := timeNow()
	_, err = tx.Exec(`INSERT INTO quota_intents(agent_id, episode_id, revision, source_generation, executing_turn, owner_json, observed_at, reset_at, due_at, state, reason_code, attempt_token, provider_started, continued_generation, continued_turn, notice_key, notice_confirmed, success_confirmed, manual_generation, manual_turn, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, '', '', 0, '', '', '', 0, 0, '', '', ?)
ON CONFLICT(agent_id) DO UPDATE SET episode_id=excluded.episode_id, revision=excluded.revision,
source_generation=excluded.source_generation, executing_turn=excluded.executing_turn, owner_json=excluded.owner_json,
observed_at=excluded.observed_at, reset_at=excluded.reset_at, due_at=excluded.due_at, state=excluded.state,
reason_code='', attempt_token='', provider_started=0, continued_generation='', continued_turn='', notice_key='', notice_confirmed=0, success_confirmed=0, manual_generation='', manual_turn='', updated_at=excluded.updated_at`,
		in.AgentID, in.EpisodeID, revision, in.SourceGeneration, in.ExecutingTurn, string(owner), formatTime(in.ObservedAt), formatOptionalTime(in.ResetAt), formatOptionalTime(in.DueAt), in.State, formatTime(now))
	if err != nil {
		return QuotaIntent{}, fmt.Errorf("state: write quota observation: %w", err)
	}
	return scanQuota(tx.QueryRow(quotaSelect+` WHERE agent_id = ?`, in.AgentID))
}

func (s *Store) GetQuota(agentID string) (QuotaIntent, error) {
	q, err := scanQuota(s.db.QueryRow(quotaSelect+` WHERE agent_id = ?`, agentID))
	if errors.Is(err, sql.ErrNoRows) {
		return QuotaIntent{}, ErrNotFound
	}
	return q, err
}

const quotaSelect = `SELECT agent_id, episode_id, revision, source_generation, executing_turn, owner_json, observed_at, reset_at, due_at, state, reason_code, attempt_token, provider_started, continued_generation, continued_turn, notice_key, notice_confirmed, success_confirmed, manual_generation, manual_turn, updated_at FROM quota_intents`

func scanQuota(row interface{ Scan(...any) error }) (QuotaIntent, error) {
	var q QuotaIntent
	var owner, observed, updated string
	var resetNull, dueNull sql.NullString
	err := row.Scan(&q.AgentID, &q.EpisodeID, &q.Revision, &q.SourceGeneration, &q.ExecutingTurn, &owner, &observed, &resetNull, &dueNull, &q.State, &q.ReasonCode, &q.AttemptToken, &q.ProviderStarted, &q.ContinuedGeneration, &q.ContinuedTurn, &q.NoticeKey, &q.NoticeConfirmed, &q.SuccessConfirmed, &q.ManualGeneration, &q.ManualTurn, &updated)
	if err != nil {
		return QuotaIntent{}, err
	}
	dec := json.NewDecoder(strings.NewReader(owner))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&q.Owner); err != nil {
		return QuotaIntent{}, fmt.Errorf("state: decode quota owner: %w", err)
	}
	if err := dec.Decode(new(any)); !errors.Is(err, io.EOF) {
		return QuotaIntent{}, fmt.Errorf("state: decode quota owner: trailing data")
	}
	if _, err := validateQuotaOwner(q.Owner); err != nil {
		return QuotaIntent{}, err
	}
	var errTime error
	if q.ObservedAt, errTime = parseTime(observed); errTime != nil {
		return QuotaIntent{}, wrapTimeErr("quota observed_at", errTime)
	}
	if q.UpdatedAt, errTime = parseTime(updated); errTime != nil {
		return QuotaIntent{}, wrapTimeErr("quota updated_at", errTime)
	}
	if q.ResetAt, errTime = parseOptionalTime(resetNull); errTime != nil {
		return QuotaIntent{}, wrapTimeErr("quota reset_at", errTime)
	}
	if q.DueAt, errTime = parseOptionalTime(dueNull); errTime != nil {
		return QuotaIntent{}, wrapTimeErr("quota due_at", errTime)
	}
	return q, nil
}

func (s *Store) DueQuotaIntents(now time.Time, limit int) ([]QuotaIntent, error) {
	if limit < 1 || limit > MaxQuotaDueBatch {
		return nil, ErrQuotaConflict
	}
	rows, err := s.db.Query(quotaSelect+` WHERE state IN ('scheduled','deferred') AND due_at IS NOT NULL AND due_at <= ? ORDER BY due_at, agent_id LIMIT ?`, formatTime(now), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]QuotaIntent, 0, limit)
	for rows.Next() {
		q, err := scanQuota(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, q)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func quotaCASResult(res sql.Result, err error) error {
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrQuotaConflict
	}
	return nil
}

// ClaimQuota wins admission by episode, revision and unique attempt token.
func (s *Store) ClaimQuota(agentID, episode string, revision int64, token string) error {
	if token == "" || utf8.RuneCountInString(token) > 128 {
		return ErrQuotaConflict
	}
	now := formatTime(timeNow())
	res, err := s.db.Exec(`UPDATE quota_intents SET state='claimed', revision=revision+1, attempt_token=?, provider_started=0, continued_generation='', continued_turn='', notice_key='', notice_confirmed=0, success_confirmed=0, manual_generation='', manual_turn='', reason_code='', updated_at=? WHERE agent_id=? AND episode_id=? AND revision=? AND state IN ('scheduled','deferred') AND due_at IS NOT NULL AND due_at<=?`, token, now, agentID, episode, revision, now)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrQuotaConflict
	}
	return nil
}

func (s *Store) DeferQuota(agentID, episode string, revision int64, token string, due time.Time, reason string) error {
	if due.IsZero() || !validQuotaReason(reason) {
		return ErrQuotaConflict
	}
	res, err := s.db.Exec(`UPDATE quota_intents SET state='deferred', revision=revision+1, due_at=?, reason_code=?, provider_started=0, updated_at=? WHERE agent_id=? AND episode_id=? AND revision=? AND state='claimed' AND attempt_token=? AND provider_started=0`, formatTime(due), reason, formatTime(timeNow()), agentID, episode, revision, token)
	return quotaCASResult(res, err)
}

func (s *Store) MarkQuotaProviderStarted(agentID, episode string, revision int64, token, generation, turn string) error {
	return markQuotaProviderStartedTx(s.db, agentID, episode, revision, token, generation, turn)
}

func markQuotaProviderStartedTx(db execer, agentID, episode string, revision int64, token, generation, turn string) error {
	if generation == "" || turn == "" {
		return ErrQuotaConflict
	}
	res, err := db.Exec(`UPDATE quota_intents SET revision=revision+1, provider_started=1, continued_generation=?, continued_turn=?, updated_at=? WHERE agent_id=? AND episode_id=? AND revision=? AND state='claimed' AND attempt_token=?`, generation, turn, formatTime(timeNow()), agentID, episode, revision, token)
	return quotaCASResult(res, err)
}

func (s *Store) MarkQuotaNotice(agentID, episode string, revision int64, token, key string) error {
	if key == "" || utf8.RuneCountInString(key) > 128 {
		return ErrQuotaConflict
	}
	res, err := s.db.Exec(`UPDATE quota_intents SET revision=revision+1, notice_key=?, notice_confirmed=0, updated_at=? WHERE agent_id=? AND episode_id=? AND revision=? AND state='claimed' AND attempt_token=? AND provider_started=1`, key, formatTime(timeNow()), agentID, episode, revision, token)
	return quotaCASResult(res, err)
}

// MarkQuotaNoticeConfirmed acknowledges the fsynced transcript event for the
// exact durable notice key.
func (s *Store) MarkQuotaNoticeConfirmed(agentID, episode string, revision int64, token, key string) error {
	if key == "" || utf8.RuneCountInString(key) > 128 {
		return ErrQuotaConflict
	}
	res, err := s.db.Exec(`UPDATE quota_intents SET revision=revision+1, notice_confirmed=1, updated_at=? WHERE agent_id=? AND episode_id=? AND revision=? AND state='claimed' AND attempt_token=? AND provider_started=1 AND notice_key=? AND notice_confirmed=0`, formatTime(timeNow()), agentID, episode, revision, token, key)
	return quotaCASResult(res, err)
}

// ConfirmQuotaSuccess records successful terminal completion for the exact
// continuation tuple so receipt repair cannot mistake progress for success.
func (s *Store) BindQuotaManualTurn(agentID, generation, turn string) error {
	if agentID == "" || generation == "" || turn == "" {
		return ErrQuotaConflict
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	q, err := scanQuota(tx.QueryRow(quotaSelect+` WHERE agent_id=?`, agentID))
	if errors.Is(err, sql.ErrNoRows) || err == nil && q.State == QuotaContinued {
		return nil
	}
	if err != nil {
		return err
	}
	if q.Owner.Kind == "task" {
		var assignedGeneration string
		err := tx.QueryRow(`SELECT assigned_generation FROM tasks WHERE task_id=? AND assigned_agent_id=? AND state IN (?,?) AND pending_release=0 AND pending_yield=0`+taskRunOpenClause,
			q.Owner.TaskID, agentID, TaskRunning, TaskWaiting).Scan(&assignedGeneration)
		if errors.Is(err, sql.ErrNoRows) || err == nil && assignedGeneration != q.SourceGeneration && assignedGeneration != q.ContinuedGeneration && assignedGeneration != q.ManualGeneration {
			return ErrQuotaOwner
		}
		if err != nil {
			return err
		}
		res, err := tx.Exec(`UPDATE tasks SET assigned_generation=?, execution_turn=?, revision=revision+1, updated_at=? WHERE task_id=? AND assigned_agent_id=? AND assigned_generation=? AND state IN (?,?) AND pending_release=0 AND pending_yield=0`+taskRunOpenClause,
			generation, turn, formatTime(timeNow()), q.Owner.TaskID, agentID, assignedGeneration, TaskRunning, TaskWaiting)
		if err != nil {
			return err
		}
		if n, err := res.RowsAffected(); err != nil || n != 1 {
			if err != nil {
				return err
			}
			return ErrQuotaOwner
		}
	}
	res, err := tx.Exec(`UPDATE quota_intents SET revision=revision+1, manual_generation=?, manual_turn=?, updated_at=? WHERE agent_id=? AND episode_id=? AND revision=? AND state<>'continued'`, generation, turn, formatTime(timeNow()), agentID, q.EpisodeID, q.Revision)
	if err := quotaCASResult(res, err); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) HoldQuotaManualTurn(agentID, generation, turn string) error {
	if agentID == "" || generation == "" || turn == "" {
		return ErrQuotaConflict
	}
	_, err := s.db.Exec(`UPDATE quota_intents SET state='needs_attention', revision=revision+1, reason_code='non_quota_failure', due_at=NULL, updated_at=? WHERE agent_id=? AND manual_generation=? AND manual_turn=? AND state<>'continued'`, formatTime(timeNow()), agentID, generation, turn)
	return err
}

func (s *Store) ConfirmQuotaSuccess(agentID, episode string, revision int64, token, generation, turn string) error {
	if generation == "" || turn == "" {
		return ErrQuotaConflict
	}
	res, err := s.db.Exec(`UPDATE quota_intents SET revision=revision+1, success_confirmed=1, updated_at=? WHERE agent_id=? AND episode_id=? AND revision=? AND state='claimed' AND attempt_token=? AND provider_started=1 AND continued_generation=? AND continued_turn=?`, formatTime(timeNow()), agentID, episode, revision, token, generation, turn)
	return quotaCASResult(res, err)
}

func (s *Store) CompleteQuota(agentID, episode string, revision int64, token, generation, turn string) error {
	if generation == "" || turn == "" {
		return ErrQuotaConflict
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	q, err := scanQuota(tx.QueryRow(quotaSelect+` WHERE agent_id=?`, agentID))
	if err != nil {
		return err
	}
	res, err := tx.Exec(`UPDATE quota_intents SET state='continued', revision=revision+1, due_at=NULL, updated_at=? WHERE agent_id=? AND episode_id=? AND revision=? AND state='claimed' AND attempt_token=? AND provider_started=1 AND continued_generation=? AND continued_turn=? AND notice_key<>'' AND notice_confirmed=1 AND success_confirmed=1`, formatTime(timeNow()), agentID, episode, revision, token, generation, turn)
	if err := quotaCASResult(res, err); err != nil {
		return err
	}
	if q.Owner.Kind == "room" {
		if err := releaseQuotaRoomHoldTx(tx, q.Owner.RoomID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) HoldQuota(agentID, episode string, revision int64, token, reason string) error {
	if !validQuotaReason(reason) {
		return ErrQuotaConflict
	}
	res, err := s.db.Exec(`UPDATE quota_intents SET state='needs_attention', revision=revision+1, reason_code=?, due_at=NULL, updated_at=? WHERE agent_id=? AND episode_id=? AND revision=? AND state='claimed' AND attempt_token=?`, reason, formatTime(timeNow()), agentID, episode, revision, token)
	return quotaCASResult(res, err)
}

func (s *Store) CancelQuota(agentID, episode string, revision int64, token string) error {
	res, err := s.db.Exec(`UPDATE quota_intents SET state='cancelled', revision=revision+1, due_at=NULL, updated_at=? WHERE agent_id=? AND episode_id=? AND revision=? AND state IN ('scheduled','deferred','claimed') AND attempt_token=?`, formatTime(timeNow()), agentID, episode, revision, token)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrQuotaConflict
	}
	return nil
}

// CancelPendingQuotas invalidates all unstarted intents when policy is disabled.
// Already-cancelled identities are included so saving false again can repair a
// failed projection publication without changing their revisions.
func (s *Store) CancelPendingQuotas() ([]string, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE quota_intents SET state='cancelled', revision=revision+1, due_at=NULL, updated_at=? WHERE state IN ('scheduled','deferred','claimed') AND provider_started=0`, formatTime(timeNow())); err != nil {
		return nil, err
	}
	rows, err := tx.Query(`SELECT agent_id FROM quota_intents WHERE state='cancelled' AND provider_started=0 ORDER BY agent_id`)
	if err != nil {
		return nil, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return ids, nil
}

// ClearQuota cancels an intent only when it still names the exact execution
// tuple that received successful manual input.
func (s *Store) ClearQuota(agentID, generation, turn string) (bool, error) {
	res, err := s.db.Exec(`UPDATE quota_intents SET state='cancelled', revision=revision+1, due_at=NULL, updated_at=? WHERE agent_id=? AND state IN ('scheduled','deferred','claimed') AND ((source_generation=? AND executing_turn=?) OR (continued_generation=? AND continued_turn=?))`, formatTime(timeNow()), agentID, generation, turn, generation, turn)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
}

// RecoverQuotaClaims repairs process-owned claims after a restart. A claim with
// no recorded provider effect is readmittable; uncertain delivery needs review.
func (s *Store) RecoverQuotaClaims() error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := formatTime(timeNow())
	if _, err := tx.Exec(`UPDATE quota_intents SET state='scheduled', revision=revision+1, attempt_token='', reason_code='', updated_at=? WHERE state='claimed' AND provider_started=0`, now); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE quota_intents SET state='needs_attention', revision=revision+1, due_at=NULL, reason_code='uncertain_delivery', updated_at=? WHERE state='claimed' AND provider_started=1 AND (notice_key='' OR notice_confirmed=1 AND success_confirmed=0)`, now); err != nil {
		return err
	}
	return tx.Commit()
}

func validQuotaReason(reason string) bool {
	switch reason {
	case "busy", "capacity", "store_failure", "owner_unavailable", "project_ineligible", "archived", "provider_start_failed", "uncertain_delivery", "non_quota_failure", "invalid_owner":
		return true
	}
	return false
}
