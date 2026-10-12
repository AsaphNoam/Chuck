package state

import (
	"database/sql"
	"errors"
	"time"
)

// AttachmentReceipt records acceptance before any provider-visible effect.
// MessageJSON contains descriptors only; immutable bytes live in conversation storage.
type AttachmentReceipt struct {
	AgentID, RequestID, PayloadHash, MessageJSON, Action, Delivery, Outcome string
}

func (s *Store) AttachmentReceipt(agentID, requestID string) (AttachmentReceipt, error) {
	var r AttachmentReceipt
	err := s.db.QueryRow(`SELECT agent_id,request_id,payload_hash,message_json,action,delivery,outcome FROM attachment_receipts WHERE agent_id=? AND request_id=?`, agentID, requestID).Scan(&r.AgentID, &r.RequestID, &r.PayloadHash, &r.MessageJSON, &r.Action, &r.Delivery, &r.Outcome)
	if errors.Is(err, sql.ErrNoRows) {
		return r, ErrNotFound
	}
	return r, err
}

func (s *Store) CreateAttachmentReceipt(r AttachmentReceipt) error {
	_, err := s.db.Exec(`INSERT INTO attachment_receipts(agent_id,request_id,payload_hash,message_json,action,delivery,updated_at) VALUES(?,?,?,?,?,'unknown',?)`, r.AgentID, r.RequestID, r.PayloadHash, r.MessageJSON, r.Action, formatTime(time.Now().UTC()))
	return err
}

func (s *Store) SetAttachmentDelivery(agentID, requestID, delivery string) error {
	_, err := s.db.Exec(`UPDATE attachment_receipts SET delivery=?,updated_at=? WHERE agent_id=? AND request_id=?`, delivery, formatTime(time.Now().UTC()), agentID, requestID)
	return err
}

func (s *Store) SetAttachmentOutcome(agentID, requestID, outcome string) error {
	_, err := s.db.Exec(`UPDATE attachment_receipts SET outcome=? WHERE agent_id=? AND request_id=?`, outcome, agentID, requestID)
	return err
}

func (s *Store) RemoveAttachmentReceipt(agentID, requestID string) error {
	_, err := s.db.Exec(`DELETE FROM attachment_receipts WHERE agent_id=? AND request_id=?`, agentID, requestID)
	return err
}

func (s *Store) AttachmentRecovery(agentID string) (AttachmentReceipt, error) {
	var id string
	err := s.db.QueryRow(`SELECT request_id FROM attachment_receipts WHERE agent_id=? AND delivery='released' AND dismissed=0 ORDER BY updated_at DESC,rowid DESC LIMIT 1`, agentID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return AttachmentReceipt{}, ErrNotFound
	}
	if err != nil {
		return AttachmentReceipt{}, err
	}
	return s.AttachmentReceipt(agentID, id)
}

func (s *Store) DismissAttachmentRecovery(agentID, requestID string) error {
	_, err := s.db.Exec(`UPDATE attachment_receipts SET dismissed=1 WHERE agent_id=? AND request_id=? AND delivery='released'`, agentID, requestID)
	return err
}

// A held request is never a durable runnable queue. Startup releases it for
// explicit recovery; submitted/unknown receipts remain non-retryable evidence.
func (s *Store) ReleaseRestartedAttachments() error {
	_, err := s.db.Exec(`UPDATE attachment_receipts SET delivery='released',updated_at=? WHERE delivery='held'`, formatTime(time.Now().UTC()))
	return err
}
