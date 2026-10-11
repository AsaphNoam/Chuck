package state

import "errors"

// ResolveQuota clears active quota presentation only for a successful turn in
// the source generation or the exact admitted continuation tuple. The latest
// bounded receipt remains stored until conversation deletion.
func (s *Store) ResolveQuota(agentID, generation, turn string) error {
	q, err := s.GetQuota(agentID)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if (q.SourceGeneration != generation || q.ExecutingTurn != turn) &&
		(q.ContinuedGeneration != generation || q.ContinuedTurn != turn) &&
		(q.ManualGeneration != generation || q.ManualTurn != turn) {
		return nil
	}
	_, err = s.db.Exec(`UPDATE quota_intents SET state='continued', revision=revision+1, due_at=NULL, updated_at=?
WHERE agent_id=? AND episode_id=? AND revision=?`, formatTime(timeNow()), agentID, q.EpisodeID, q.Revision)
	return err
}
