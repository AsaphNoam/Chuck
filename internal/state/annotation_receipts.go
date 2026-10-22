package state

import (
	"database/sql"
	"fmt"
	"time"
)

type AnnotationReceipt struct {
	SourceAgentID string
	CommandID     string
	PayloadHash   string
	TargetKind    string
	TargetAgentID string
	SourceSeq     int64
	EffectID      string
	State         string
	UpdatedAt     time.Time
}

func (s *Store) AnnotationReceipt(sourceAgentID, commandID string) (AnnotationReceipt, error) {
	var r AnnotationReceipt
	var updated string
	err := s.db.QueryRow(`SELECT source_agent_id, command_id, payload_hash, target_kind, target_agent_id,
source_seq, effect_id, state, updated_at FROM annotation_receipts WHERE source_agent_id=? AND command_id=?`,
		sourceAgentID, commandID).Scan(&r.SourceAgentID, &r.CommandID, &r.PayloadHash, &r.TargetKind,
		&r.TargetAgentID, &r.SourceSeq, &r.EffectID, &r.State, &updated)
	if err != nil {
		if err == sql.ErrNoRows {
			return AnnotationReceipt{}, ErrNotFound
		}
		return AnnotationReceipt{}, fmt.Errorf("state: read annotation receipt: %w", err)
	}
	r.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	return r, nil
}

func (s *Store) CreateAnnotationReceipt(r AnnotationReceipt) error {
	if r.UpdatedAt.IsZero() {
		r.UpdatedAt = time.Now().UTC()
	}
	_, err := s.db.Exec(`INSERT INTO annotation_receipts(source_agent_id, command_id, payload_hash, target_kind,
target_agent_id, source_seq, effect_id, state, updated_at) VALUES(?,?,?,?,?,?,?,?,?)`,
		r.SourceAgentID, r.CommandID, r.PayloadHash, r.TargetKind, r.TargetAgentID,
		r.SourceSeq, r.EffectID, r.State, formatTime(r.UpdatedAt))
	if err != nil {
		return fmt.Errorf("state: create annotation receipt: %w", err)
	}
	return nil
}

func (s *Store) SetAnnotationSourceSeq(sourceAgentID, commandID string, seq int64) error {
	_, err := s.db.Exec(`UPDATE annotation_receipts SET source_seq=?, state='recorded', updated_at=?
WHERE source_agent_id=? AND command_id=? AND source_seq=0`, seq, formatTime(time.Now().UTC()), sourceAgentID, commandID)
	return err
}

func (s *Store) SetAnnotationEffect(sourceAgentID, commandID, effectID, state string) error {
	_, err := s.db.Exec(`UPDATE annotation_receipts SET effect_id=?, state=?, updated_at=?
WHERE source_agent_id=? AND command_id=?`, effectID, state, formatTime(time.Now().UTC()), sourceAgentID, commandID)
	return err
}

// ClaimAnnotationEffect makes an external self-prompt explicitly uncertain
// before the provider call. A crash after this update therefore cannot replay
// a prompt whose native effect may already have happened.
func (s *Store) ClaimAnnotationEffect(sourceAgentID, commandID string) error {
	_, err := s.db.Exec(`UPDATE annotation_receipts SET state='unknown', updated_at=?
WHERE source_agent_id=? AND command_id=? AND state='recorded'`, formatTime(time.Now().UTC()), sourceAgentID, commandID)
	return err
}

// CommitAnnotationMail atomically creates the peer mail and records its effect.
// A retry after a lost HTTP response therefore observes the same receipt and
// never inserts a second message.
func (s *Store) CommitAnnotationMail(sourceAgentID, commandID string, mail Message) (AnnotationReceipt, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return AnnotationReceipt{}, fmt.Errorf("state: begin annotation mail: %w", err)
	}
	defer tx.Rollback()
	var r AnnotationReceipt
	var updated string
	if err := tx.QueryRow(`SELECT source_agent_id, command_id, payload_hash, target_kind, target_agent_id,
source_seq, effect_id, state, updated_at FROM annotation_receipts WHERE source_agent_id=? AND command_id=?`,
		sourceAgentID, commandID).Scan(&r.SourceAgentID, &r.CommandID, &r.PayloadHash, &r.TargetKind,
		&r.TargetAgentID, &r.SourceSeq, &r.EffectID, &r.State, &updated); err != nil {
		return AnnotationReceipt{}, err
	}
	r.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	if r.EffectID != "" || r.State == "accepted" {
		return r, nil
	}
	mail.Wake, mail.DeliveredVia = true, DeliveryPending
	effectID, err := insertMessageTx(tx, mail)
	if err != nil {
		return AnnotationReceipt{}, err
	}
	if err := EnsurePendingMailActivationTx(tx, mail.ToAgent); err != nil {
		return AnnotationReceipt{}, err
	}
	if _, err := tx.Exec(`UPDATE annotation_receipts SET effect_id=?, state='accepted', updated_at=?
WHERE source_agent_id=? AND command_id=?`, effectID, formatTime(time.Now().UTC()), sourceAgentID, commandID); err != nil {
		return AnnotationReceipt{}, err
	}
	if err := tx.Commit(); err != nil {
		return AnnotationReceipt{}, err
	}
	r.EffectID, r.State = effectID, "accepted"
	return r, nil
}
