package state

import (
	"database/sql"
	"time"
	"unicode/utf8"
)

// Isolated quota checks are receipts on the single recovery row
// (TS-02.R50, TS-10.R46–R47). A check receipt never satisfies an original-work
// provider_started or continuation receipt.

const (
	QuotaCheckReady       = "ready"
	QuotaCheckQuota       = "quota"
	QuotaCheckUnsupported = "unsupported"
	QuotaCheckFailed      = "failed"
	QuotaCheckTimeout     = "timeout"
)

// QuotaCheckResult is the bounded outcome of one check. Cleaned reports that
// the exact probe session/process was disposed. Intervention marks a resetless
// quota result naming a credit, plan or spend-cap requirement.
type QuotaCheckResult struct {
	Outcome      string
	ResetAt      *time.Time
	ResetSource  string
	Intervention bool
	Cleaned      bool
}

// StartQuotaCheck charges the chain's allowance once, before the provider
// process exists. A pending cleanup reference must resolve first.
func (s *Store) StartQuotaCheck(agentID, episode string, revision int64, token, backend string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := startQuotaCheckTx(tx, agentID, episode, revision, token, backend); err != nil {
		return err
	}
	return tx.Commit()
}

// StartEligibleQuotaCheck rechecks owner/capacity and charges in one transaction,
// without borrowing or reserving a runtime on behalf of an isolated probe.
func (s *Store) StartEligibleQuotaCheck(q QuotaIntent, backend string, budget int) (bool, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	eligible, err := quotaWorkEligibleTx(tx, q, budget)
	if err != nil || !eligible {
		return false, err
	}
	if err := startQuotaCheckTx(tx, q.AgentID, q.EpisodeID, q.Revision, q.AttemptToken, backend); err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

func startQuotaCheckTx(tx *sql.Tx, agentID, episode string, revision int64, token, backend string) error {
	if backend != "claude-acp" && backend != "codex-acp" {
		return ErrQuotaConflict
	}
	res, err := tx.Exec(`UPDATE quota_intents SET revision=revision+1, check_started=1, checks_started=checks_started+1, check_backend=?, updated_at=?
WHERE agent_id=? AND episode_id=? AND revision=? AND state='claimed' AND attempt_token=? AND next_action='check'
AND check_started=0 AND provider_started=0 AND checks_started<? AND check_session=''`,
		backend, formatTime(timeNow()), agentID, episode, revision, token, MaxQuotaChecks)
	return quotaCASResult(res, err)
}

// RecordQuotaCheckSession persists the exact probe session before any probe
// prompt so cleanup can be retried without loading or prompting it.
func (s *Store) RecordQuotaCheckSession(agentID, episode string, revision int64, token, session string) error {
	if session == "" || len(session) > 256 || !utf8.ValidString(session) {
		return ErrQuotaConflict
	}
	res, err := s.db.Exec(`UPDATE quota_intents SET revision=revision+1, check_session=?, updated_at=?
WHERE agent_id=? AND episode_id=? AND revision=? AND state='claimed' AND attempt_token=? AND check_started=1 AND check_session=''`,
		session, formatTime(timeNow()), agentID, episode, revision, token)
	return quotaCASResult(res, err)
}

// ResolveQuotaCheckCleanup drops an exact cleanup reference after its session
// was disposed, independent of whatever the intent has become since.
func (s *Store) ResolveQuotaCheckCleanup(agentID, session string) error {
	if session == "" {
		return nil
	}
	_, err := s.db.Exec(`UPDATE quota_intents SET revision=revision+1, check_session='', check_backend='', updated_at=? WHERE agent_id=? AND check_session=?`,
		formatTime(timeNow()), agentID, session)
	return err
}

// FinishQuotaCheck applies a check outcome to the exact started claim. Only a
// ready check authorizes one current continuation; it never completes work.
func (s *Store) FinishQuotaCheck(agentID, episode string, revision int64, token string, r QuotaCheckResult) error {
	now := timeNow()
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	q, err := scanQuota(tx.QueryRow(quotaSelect+` WHERE agent_id=? AND episode_id=? AND revision=? AND state='claimed' AND attempt_token=? AND check_started=1`,
		agentID, episode, revision, token))
	if err != nil {
		return ErrQuotaConflict
	}
	state, action, reason := QuotaNeedsAttention, QuotaActionCheck, ""
	var resetAt, due *time.Time
	ready, source := false, ""
	switch {
	case !r.Cleaned:
		reason = "check_cleanup_failed"
	case r.Outcome == QuotaCheckReady:
		state, action, ready, due = QuotaScheduled, QuotaActionContinue, true, &now
	case r.Outcome == QuotaCheckQuota && r.ResetAt != nil && r.ResetAt.After(now) && (r.ResetSource == "structured" || r.ResetSource == "text"):
		d := r.ResetAt.Add(QuotaResetBuffer)
		state, action, resetAt, due, source = QuotaScheduled, QuotaActionContinue, r.ResetAt, &d, r.ResetSource
	case r.Outcome == QuotaCheckQuota && r.Intervention:
		// Explicit credit/plan/spend-cap errors stay manual, exactly like an
		// interruption first observed with that text (FS-01.R49).
		state = QuotaUnknownReset
		action = QuotaActionContinue
	case r.Outcome == QuotaCheckQuota && q.ChecksStarted >= MaxQuotaChecks:
		reason = "check_exhausted"
	case r.Outcome == QuotaCheckQuota:
		d := now.Add(QuotaCheckInterval)
		state, due = QuotaScheduled, &d
	case r.Outcome == QuotaCheckUnsupported:
		reason = "check_unsupported"
	case r.Outcome == QuotaCheckTimeout:
		reason = "check_timeout"
	default:
		reason = "check_failed"
	}
	res, err := tx.Exec(`UPDATE quota_intents SET state=?, next_action=?, reason_code=?, reset_at=COALESCE(?, reset_at), due_at=?, reset_source=CASE WHEN ?<>'' THEN ? ELSE reset_source END,
check_ready=?, check_started=0, attempt_token='', revision=revision+1, updated_at=?
WHERE agent_id=? AND episode_id=? AND revision=? AND state='claimed' AND attempt_token=? AND check_started=1`,
		state, action, reason, formatOptionalTime(resetAt), formatOptionalTime(due), source, source, ready, formatTime(now),
		agentID, episode, revision, token)
	if err := quotaCASResult(res, err); err != nil {
		return err
	}
	return tx.Commit()
}

// PendingQuotaCheckCleanups lists exact probe references awaiting cleanup. The
// caller skips any whose check is still running in this process.
func (s *Store) PendingQuotaCheckCleanups(limit int) ([]QuotaIntent, error) {
	return s.PendingQuotaCheckCleanupsAfter(limit, "")
}

// PendingQuotaCheckCleanupsAfter pages a stable key so failed, busy and cooling
// references cannot monopolize every bounded cleanup sweep.
func (s *Store) PendingQuotaCheckCleanupsAfter(limit int, afterAgentID string) ([]QuotaIntent, error) {
	if limit < 1 || limit > MaxQuotaDueBatch {
		return nil, ErrQuotaConflict
	}
	rows, err := s.db.Query(quotaSelect+` WHERE check_session<>'' AND agent_id>? ORDER BY agent_id LIMIT ?`, afterAgentID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []QuotaIntent
	for rows.Next() {
		q, err := scanQuota(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, q)
	}
	return items, rows.Err()
}
