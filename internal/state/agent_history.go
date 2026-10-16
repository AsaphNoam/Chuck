package state

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// HistoryAgent is one durable identity in the bounded history directory
// (FS-06.R40, TS-04.R93). It carries identity and lifecycle metadata only:
// never transcript content, credentials, or mail eligibility.
type HistoryAgent struct {
	AgentID   string `json:"agent_id"`
	Name      string `json:"name"`
	Role      string `json:"role"`
	Project   string `json:"project"`
	Interface string `json:"interface"`
	State     string `json:"state"`
	Running   bool   `json:"running"`
	Archived  bool   `json:"archived"`
	CreatedAt string `json:"created_at"`
}

// HistoryFilter selects one page of AgentHistory. AfterCreatedAt/AfterID are
// the previous page's last keyset position; both empty start from the newest.
type HistoryFilter struct {
	Project        string
	State          string
	ExcludeAgentID string
	Limit          int
	AfterCreatedAt string
	AfterID        string
}

// AgentHistory pages every durable agent identity — running, stopped, and
// archived, across projects — newest first by creation time then agent id.
func (s *Store) AgentHistory(f HistoryFilter) ([]HistoryAgent, error) {
	var where []string
	var args []any
	if f.Project != "" {
		where = append(where, "a.project = ?")
		args = append(args, f.Project)
	}
	if f.State != "" {
		where = append(where, "COALESCE(st.state, 'unknown') = ?")
		args = append(args, f.State)
	}
	if f.ExcludeAgentID != "" {
		where = append(where, "a.agent_id <> ?")
		args = append(args, f.ExcludeAgentID)
	}
	if f.AfterCreatedAt != "" {
		where = append(where, "(a.created_at < ? OR (a.created_at = ? AND a.agent_id < ?))")
		args = append(args, f.AfterCreatedAt, f.AfterCreatedAt, f.AfterID)
	}
	query := historySelect
	if len(where) > 0 {
		query += "\nWHERE " + strings.Join(where, " AND ")
	}
	query += "\nORDER BY a.created_at DESC, a.agent_id DESC\nLIMIT ?"
	args = append(args, f.Limit)
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("state: list agent history: %w", err)
	}
	defer rows.Close()
	out := []HistoryAgent{}
	for rows.Next() {
		h, err := scanHistoryAgent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("state: iterate agent history: %w", err)
	}
	return out, nil
}

// ReadHistoryAgent returns one durable identity's history metadata, or
// ErrNotFound when no such agent exists.
func (s *Store) ReadHistoryAgent(agentID string) (HistoryAgent, error) {
	h, err := scanHistoryAgent(s.db.QueryRow(historySelect+"\nWHERE a.agent_id = ?", agentID))
	if errors.Is(err, sql.ErrNoRows) {
		return HistoryAgent{}, ErrNotFound
	}
	return h, err
}

// historySelect is the one projection both history reads share (INV §2).
const historySelect = `
SELECT a.agent_id, a.name, a.role, a.project, a.interface, COALESCE(st.state, 'unknown'),
       r.agent_id IS NOT NULL, a.archived, a.created_at
FROM agents a
LEFT JOIN running r ON r.agent_id = a.agent_id
LEFT JOIN status st ON st.agent_id = a.agent_id`

func scanHistoryAgent(row interface{ Scan(...any) error }) (HistoryAgent, error) {
	var h HistoryAgent
	err := row.Scan(&h.AgentID, &h.Name, &h.Role, &h.Project, &h.Interface, &h.State,
		&h.Running, &h.Archived, &h.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return HistoryAgent{}, err
	}
	if err != nil {
		return HistoryAgent{}, fmt.Errorf("state: scan agent history: %w", err)
	}
	return h, nil
}
