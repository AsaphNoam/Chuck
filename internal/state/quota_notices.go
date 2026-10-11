package state

// PendingQuotaNotices returns only provider-started claims whose stable notice
// receipt committed before transcript append. The batch bound keeps startup and
// dispatcher repair work finite.
func (s *Store) PendingQuotaNotices(limit int) ([]QuotaIntent, error) {
	if limit < 1 || limit > MaxQuotaDueBatch {
		return nil, ErrQuotaConflict
	}
	rows, err := s.db.Query(quotaSelect+` WHERE state='claimed' AND provider_started=1 AND notice_key<>'' AND (notice_confirmed=0 OR success_confirmed=1) ORDER BY updated_at, agent_id LIMIT ?`, limit)
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
