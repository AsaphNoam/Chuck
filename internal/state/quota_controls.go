package state

import (
	"database/sql"
	"errors"
	"fmt"
)

// QuotaControlSnapshot identifies the one source episode visible before a
// manual owner mutation. Applying it later cannot cancel a replacement episode.
type QuotaControlSnapshot struct {
	AgentID, EpisodeID string
	Revision           int64
	AttemptToken       string
	SourceGeneration   string
	ExecutingTurn      string
}

func quotaControlSnapshot(q QuotaIntent) QuotaControlSnapshot {
	return QuotaControlSnapshot{AgentID: q.AgentID, EpisodeID: q.EpisodeID, Revision: q.Revision,
		AttemptToken: q.AttemptToken, SourceGeneration: q.SourceGeneration, ExecutingTurn: q.ExecutingTurn}
}

func (s *Store) SnapshotAgentQuota(agentID string) (*QuotaControlSnapshot, error) {
	q, err := s.GetQuota(agentID)
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if q.ProviderStarted || !quotaControlCancellable(q.State) {
		return nil, nil
	}
	snap := quotaControlSnapshot(q)
	return &snap, nil
}

// SnapshotOwnerQuotas captures active episodes that reference the owner before
// a task, pipeline run, or room is changed.
func (s *Store) SnapshotOwnerQuotas(owner QuotaOwner) ([]QuotaControlSnapshot, error) {
	if _, err := validateQuotaOwner(owner); err != nil || owner.Kind == "ordinary" {
		return nil, ErrQuotaOwner
	}
	column, value := "", ""
	switch owner.Kind {
	case "task":
		column, value = "task_id", owner.TaskID
	case "pipeline":
		column, value = "run_id", owner.RunID
	case "room":
		column, value = "room_id", owner.RoomID
	}
	return s.snapshotQuotaOwnerID(owner.Kind, column, value)
}

func (s *Store) SnapshotQuotaOwnerID(kind, field, value string) ([]QuotaControlSnapshot, error) {
	if kind != "task" && kind != "pipeline" && kind != "room" {
		return nil, ErrQuotaOwner
	}
	if field != "task_id" && field != "run_id" && field != "room_id" {
		return nil, ErrQuotaOwner
	}
	return s.snapshotQuotaOwnerID(kind, field, value)
}

func (s *Store) snapshotQuotaOwnerID(kind, column, value string) ([]QuotaControlSnapshot, error) {
	query := ` WHERE json_extract(owner_json, '$.kind')=? AND json_extract(owner_json, '$.` + column + `')=?`
	args := []any{kind, value}
	if kind == "pipeline" && column == "run_id" {
		query = ` WHERE (json_extract(owner_json,'$.kind')='pipeline' AND json_extract(owner_json,'$.run_id')=?)
OR (json_extract(owner_json,'$.task_id') IN (
  SELECT task_id FROM task_lineage WHERE pipeline_run_id=?
  UNION SELECT task_id FROM pipeline_stage_tasks WHERE run_id=?))
OR (json_extract(owner_json,'$.kind')='room' AND json_extract(owner_json,'$.room_id') IN
  (SELECT room_id FROM think_tanks WHERE pipeline_run_id=? AND pipeline_run_id!=''))`
		args = []any{value, value, value, value}
	}
	rows, err := s.db.Query(quotaSelect+query, args...)
	if err != nil {
		return nil, fmt.Errorf("state: snapshot owner quotas: %w", err)
	}
	defer rows.Close()
	var snapshots []QuotaControlSnapshot
	for rows.Next() {
		q, err := scanQuota(rows)
		if err != nil {
			return nil, err
		}
		if !q.ProviderStarted && quotaControlCancellable(q.State) {
			snapshots = append(snapshots, quotaControlSnapshot(q))
		}
	}
	return snapshots, rows.Err()
}

func quotaControlCancellable(state string) bool {
	switch state {
	case QuotaDisabled, QuotaUnknownReset, QuotaScheduled, QuotaDeferred, QuotaClaimed, QuotaNeedsAttention:
		return true
	default:
		return false
	}
}

// CancelQuotaSnapshot is best-effort CAS: a changed or already-started source
// episode is left alone because it may belong to work accepted after capture.
func (s *Store) CancelQuotaSnapshot(snapshot QuotaControlSnapshot) (bool, error) {
	return cancelQuotaSnapshotTx(s.db, snapshot)
}

type quotaControlDB interface {
	Exec(string, ...any) (sql.Result, error)
	QueryRow(string, ...any) *sql.Row
}

func cancelQuotaSnapshotTx(db quotaControlDB, snapshot QuotaControlSnapshot) (bool, error) {
	var revision int64
	err := db.QueryRow(`SELECT revision FROM quota_intents WHERE agent_id=? AND episode_id=? AND source_generation=? AND executing_turn=? AND attempt_token=? AND provider_started=0 AND state IN (?,?,?,?,?,?)`,
		snapshot.AgentID, snapshot.EpisodeID, snapshot.SourceGeneration, snapshot.ExecutingTurn, snapshot.AttemptToken,
		QuotaDisabled, QuotaUnknownReset, QuotaScheduled, QuotaDeferred, QuotaClaimed, QuotaNeedsAttention).Scan(&revision)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	res, err := db.Exec(`UPDATE quota_intents SET state='cancelled', revision=revision+1, due_at=NULL, updated_at=?
	WHERE agent_id=? AND episode_id=? AND revision=? AND source_generation=? AND executing_turn=? AND attempt_token=? AND provider_started=0 AND state IN (?,?,?,?,?,?)`,
		formatTime(timeNow()), snapshot.AgentID, snapshot.EpisodeID, revision, snapshot.SourceGeneration, snapshot.ExecutingTurn,
		snapshot.AttemptToken, QuotaDisabled, QuotaUnknownReset, QuotaScheduled, QuotaDeferred, QuotaClaimed, QuotaNeedsAttention)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
}

func (s *Store) CancelQuotaSnapshots(snapshots []QuotaControlSnapshot) ([]string, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var changed []string
	for _, snapshot := range snapshots {
		ok, err := cancelQuotaSnapshotTx(tx, snapshot)
		if err != nil {
			return nil, err
		}
		if ok {
			changed = append(changed, snapshot.AgentID)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return changed, nil
}
