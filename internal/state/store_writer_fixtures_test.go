package state

// Writers below have no production caller (TS-01.R50) and exist only as
// fixtures for state-package tests.

// DeleteMessages removes the given messages.
func (s *Store) DeleteMessages(ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	placeholders, args := inClause(ids)
	_, err := s.db.Exec(`DELETE FROM messages WHERE message_id IN (`+placeholders+`)`, args...)
	return err
}

// MarkQuotaNoticeConfirmed acknowledges the durable notice key.
func (s *Store) MarkQuotaNoticeConfirmed(agentID, episode string, revision int64, token, key string) error {
	if key == "" || len([]rune(key)) > 128 {
		return ErrQuotaConflict
	}
	res, err := s.db.Exec(`UPDATE quota_intents SET revision=revision+1, notice_confirmed=1, updated_at=? WHERE agent_id=? AND episode_id=? AND revision=? AND state='claimed' AND attempt_token=? AND provider_started=1 AND notice_key=? AND notice_confirmed=0`, formatTime(timeNow()), agentID, episode, revision, token, key)
	return quotaCASResult(res, err)
}

// CancelQuotaSnapshot cancels one snapshot episode by CAS.
func (s *Store) CancelQuotaSnapshot(snapshot QuotaControlSnapshot) (bool, error) {
	return cancelQuotaSnapshotTx(s.db, snapshot)
}
