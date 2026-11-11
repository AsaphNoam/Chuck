package state

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Think Tank templates are project-independent snapshots of portable room
// setup (FS-21.R60–R61, TS-14.R30–R31). They never carry agent, project,
// workspace, session or room-history identity.

const (
	ThinkTankTemplateVersion     = 1
	ThinkTankTemplateMaxName     = 120
	ThinkTankTemplateMaxBytes    = 256 << 10
	ThinkTankTemplateListDefault = 100
	ThinkTankTemplateListMax     = 200
)

// ThinkTankLaunchChoice is the allowlisted portable part of a fresh chat
// launch. Role is the persona id; empty means No persona.
type ThinkTankLaunchChoice struct {
	Role    string `json:"role"`
	Backend string `json:"backend"`
	Model   string `json:"model"`
	Effort  string `json:"effort"`
	Fast    bool   `json:"fast"`
	Name    string `json:"name,omitempty"`
}

type ThinkTankTemplateParticipant struct {
	Launch       ThinkTankLaunchChoice `json:"launch"`
	Limit        int                   `json:"limit"`
	MayLeave     bool                  `json:"may_leave"`
	Instructions string                `json:"instructions,omitempty"`
}

type ThinkTankTemplateSetup struct {
	Participants      []ThinkTankTemplateParticipant `json:"participants"`
	Openings          bool                           `json:"openings"`
	Judge             *ThinkTankLaunchChoice         `json:"judge,omitempty"`
	JudgeInstructions string                         `json:"judge_instructions,omitempty"`
}

type ThinkTankTemplate struct {
	ID      string                 `json:"id"`
	Name    string                 `json:"name"`
	Version int                    `json:"version"`
	Setup   ThinkTankTemplateSetup `json:"setup"`
}

type ThinkTankTemplateSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ThinkTankTemplateDraftRow is one unstarted participant: an existing chat
// (lookup input only) or a fresh launch choice.
type ThinkTankTemplateDraftRow struct {
	AgentID      string                 `json:"agent_id,omitempty"`
	Launch       *ThinkTankLaunchChoice `json:"launch,omitempty"`
	Limit        int                    `json:"limit"`
	MayLeave     bool                   `json:"may_leave"`
	Instructions string                 `json:"instructions,omitempty"`
}

type ThinkTankTemplateDraft struct {
	Participants      []ThinkTankTemplateDraftRow `json:"participants"`
	Openings          bool                        `json:"openings"`
	Judge             *ThinkTankLaunchChoice      `json:"judge,omitempty"`
	JudgeInstructions string                      `json:"judge_instructions,omitempty"`
}

// ThinkTankTemplateSave names exactly one source: a committed room or a draft.
type ThinkTankTemplateSave struct {
	CommandID string                  `json:"-"`
	Name      string                  `json:"name"`
	RoomID    string                  `json:"room_id,omitempty"`
	Draft     *ThinkTankTemplateDraft `json:"draft,omitempty"`
}

func migrateThinkTankTemplates(tx *sql.Tx) error {
	_, err := tx.Exec(`CREATE TABLE think_tank_templates (
  template_id TEXT PRIMARY KEY,
  command_id  TEXT NOT NULL UNIQUE,
  name        TEXT NOT NULL,
  intent      TEXT NOT NULL,
  setup       TEXT NOT NULL CHECK(length(CAST(setup AS BLOB)) <= 262144),
  created_at  TEXT NOT NULL
)`)
	return err
}

// portableLaunch reads only the allowlisted fields of a stored fresh-launch
// request; project, group, driver and interface stay behind.
func portableLaunch(raw string) (ThinkTankLaunchChoice, error) {
	var c ThinkTankLaunchChoice
	if err := json.Unmarshal([]byte(raw), &c); err != nil {
		return ThinkTankLaunchChoice{}, err
	}
	c.Name = strings.TrimSpace(c.Name)
	return c, nil
}

func agentLaunchChoiceTx(tx *sql.Tx, agentID string) (ThinkTankLaunchChoice, error) {
	var c ThinkTankLaunchChoice
	var iface string
	err := tx.QueryRow(`SELECT name, role, backend, model, effort, fast, interface FROM agents WHERE agent_id = ?`, agentID).
		Scan(&c.Name, &c.Role, &c.Backend, &c.Model, &c.Effort, &c.Fast, &iface)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && iface != "chat") {
		return ThinkTankLaunchChoice{}, ErrNotFound
	}
	if err != nil {
		return ThinkTankLaunchChoice{}, fmt.Errorf("state: read agent launch choices: %w", err)
	}
	return c, nil
}

func templateSetupFromDraftTx(tx *sql.Tx, d ThinkTankTemplateDraft) (ThinkTankTemplateSetup, error) {
	out := ThinkTankTemplateSetup{Participants: []ThinkTankTemplateParticipant{}, Openings: d.Openings,
		Judge: d.Judge, JudgeInstructions: d.JudgeInstructions}
	for i, r := range d.Participants {
		p := ThinkTankTemplateParticipant{Limit: r.Limit, MayLeave: r.MayLeave, Instructions: r.Instructions}
		switch {
		case r.AgentID != "" && r.Launch == nil:
			c, err := agentLaunchChoiceTx(tx, r.AgentID)
			if errors.Is(err, ErrNotFound) {
				return ThinkTankTemplateSetup{}, thinkTankInvalid("participant %d's launch choices cannot be recovered; replace or remove it", i+1)
			} else if err != nil {
				return ThinkTankTemplateSetup{}, err
			}
			p.Launch = c
		case r.AgentID == "" && r.Launch != nil:
			p.Launch = *r.Launch
		default:
			return ThinkTankTemplateSetup{}, thinkTankInvalid("each participant names an existing chat or new agent settings")
		}
		out.Participants = append(out.Participants, p)
	}
	return out, nil
}

func templateSetupFromRoomTx(tx *sql.Tx, roomID string) (ThinkTankTemplateSetup, error) {
	d, err := readThinkTankDetail(tx, roomID)
	if err != nil {
		return ThinkTankTemplateSetup{}, err
	}
	out := ThinkTankTemplateSetup{Participants: []ThinkTankTemplateParticipant{}, Openings: d.Room.Openings,
		JudgeInstructions: d.Room.JudgeInstructions}
	for _, m := range participantsOf(d.Members) {
		p := ThinkTankTemplateParticipant{Limit: m.Cap, MayLeave: m.MayLeave, Instructions: m.Instructions}
		if m.SetupConfig != "" {
			p.Launch, err = portableLaunch(m.SetupConfig)
		} else {
			p.Launch, err = agentLaunchChoiceTx(tx, m.AgentID)
		}
		if err != nil {
			return ThinkTankTemplateSetup{}, thinkTankInvalid("%s's launch choices cannot be recovered", m.AgentName)
		}
		out.Participants = append(out.Participants, p)
	}
	if d.Room.JudgeConfig != "" {
		j, err := portableLaunch(d.Room.JudgeConfig)
		if err != nil {
			return ThinkTankTemplateSetup{}, thinkTankInvalid("the judge's launch choices cannot be recovered")
		}
		out.Judge = &j
	}
	return out, nil
}

func validateThinkTankTemplateSetup(s ThinkTankTemplateSetup) error {
	if n := len(s.Participants); n < ThinkTankMinParticipants || n > ThinkTankMaxParticipants {
		return thinkTankInvalid("a think tank needs %d to %d participants", ThinkTankMinParticipants, ThinkTankMaxParticipants)
	}
	for _, p := range s.Participants {
		if p.Limit < 1 || p.Limit > ThinkTankMaxTurnLimit {
			return thinkTankInvalid("turn limit must be 1 to %d", ThinkTankMaxTurnLimit)
		}
		if utf8.RuneCountInString(p.Instructions) > ThinkTankMaxInstructRunes {
			return thinkTankInvalid("participant instructions exceed %d characters", ThinkTankMaxInstructRunes)
		}
	}
	if s.JudgeInstructions != "" && s.Judge == nil {
		return thinkTankInvalid("synthesis instructions need a judge")
	}
	if utf8.RuneCountInString(s.JudgeInstructions) > ThinkTankMaxGoalRunes {
		return thinkTankInvalid("synthesis instructions exceed %d characters", ThinkTankMaxGoalRunes)
	}
	return nil
}

// SaveThinkTankTemplate normalizes a draft or a committed room into one
// portable snapshot and persists it in one transaction, with no launch or
// room mutation. Exact replay of CommandID returns the committed template
// even if its source has since changed; a different request conflicts.
func (s *Store) SaveThinkTankTemplate(req ThinkTankTemplateSave) (ThinkTankTemplateSummary, error) {
	req.Name = foldThinkTankWhitespace(req.Name)
	switch {
	case strings.TrimSpace(req.CommandID) == "":
		return ThinkTankTemplateSummary{}, thinkTankInvalid("command id is required")
	case req.Name == "":
		return ThinkTankTemplateSummary{}, thinkTankInvalid("a template needs a name")
	case utf8.RuneCountInString(req.Name) > ThinkTankTemplateMaxName:
		return ThinkTankTemplateSummary{}, thinkTankInvalid("template name exceeds %d characters", ThinkTankTemplateMaxName)
	case (req.RoomID == "") == (req.Draft == nil):
		return ThinkTankTemplateSummary{}, thinkTankInvalid("save names exactly one room or setup draft")
	}
	intentRaw, err := json.Marshal(req)
	if err != nil {
		return ThinkTankTemplateSummary{}, fmt.Errorf("state: encode template intent: %w", err)
	}
	intent := string(intentRaw)
	tx, err := s.db.Begin()
	if err != nil {
		return ThinkTankTemplateSummary{}, fmt.Errorf("state: begin save template: %w", err)
	}
	defer tx.Rollback()
	var existing ThinkTankTemplateSummary
	var savedIntent string
	err = tx.QueryRow(`SELECT template_id, name, intent FROM think_tank_templates WHERE command_id = ?`, req.CommandID).
		Scan(&existing.ID, &existing.Name, &savedIntent)
	if err == nil {
		if savedIntent != intent {
			return ThinkTankTemplateSummary{}, thinkTankConflict("command id reused for a different template")
		}
		return existing, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return ThinkTankTemplateSummary{}, fmt.Errorf("state: read template command: %w", err)
	}
	var setup ThinkTankTemplateSetup
	if req.Draft != nil {
		setup, err = templateSetupFromDraftTx(tx, *req.Draft)
	} else {
		setup, err = templateSetupFromRoomTx(tx, req.RoomID)
	}
	if err != nil {
		return ThinkTankTemplateSummary{}, err
	}
	if err := validateThinkTankTemplateSetup(setup); err != nil {
		return ThinkTankTemplateSummary{}, err
	}
	raw, err := json.Marshal(setup)
	if err != nil {
		return ThinkTankTemplateSummary{}, fmt.Errorf("state: encode template: %w", err)
	}
	if len(raw) > ThinkTankTemplateMaxBytes {
		return ThinkTankTemplateSummary{}, thinkTankInvalid("the setup is too large to save")
	}
	id, err := newThinkTankID("ttt_", 8)
	if err != nil {
		return ThinkTankTemplateSummary{}, err
	}
	if _, err := tx.Exec(`INSERT INTO think_tank_templates(template_id, command_id, name, intent, setup, created_at)
VALUES(?, ?, ?, ?, ?, ?)`, id, req.CommandID, req.Name, intent, string(raw), formatTime(timeNow())); err != nil {
		return ThinkTankTemplateSummary{}, fmt.Errorf("state: insert template: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return ThinkTankTemplateSummary{}, fmt.Errorf("state: commit template: %w", err)
	}
	return ThinkTankTemplateSummary{ID: id, Name: req.Name}, nil
}

// ListThinkTankTemplates pages summaries newest first. The continuation is
// an opaque row position; an unreadable cursor is invalid input.
func (s *Store) ListThinkTankTemplates(cursor string, limit int) ([]ThinkTankTemplateSummary, string, error) {
	if limit <= 0 || limit > ThinkTankTemplateListMax {
		limit = ThinkTankTemplateListDefault
	}
	before := int64(1<<63 - 1)
	if cursor != "" {
		n, err := strconv.ParseInt(cursor, 10, 64)
		if err != nil || n <= 0 {
			return nil, "", thinkTankInvalid("invalid template cursor")
		}
		before = n
	}
	rows, err := s.db.Query(`SELECT rowid, template_id, name FROM think_tank_templates WHERE rowid < ? ORDER BY rowid DESC LIMIT ?`,
		before, limit+1)
	if err != nil {
		return nil, "", fmt.Errorf("state: list templates: %w", err)
	}
	out := []ThinkTankTemplateSummary{}
	var last int64
	next := ""
	for rows.Next() {
		var t ThinkTankTemplateSummary
		var rowid int64
		if err := rows.Scan(&rowid, &t.ID, &t.Name); err != nil {
			rows.Close()
			return nil, "", fmt.Errorf("state: scan template: %w", err)
		}
		if len(out) == limit {
			next = strconv.FormatInt(last, 10)
			break
		}
		out, last = append(out, t), rowid
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, "", fmt.Errorf("state: iterate templates: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, "", fmt.Errorf("state: close templates: %w", err)
	}
	return out, next, nil
}

func (s *Store) ReadThinkTankTemplate(id string) (ThinkTankTemplate, error) {
	t := ThinkTankTemplate{ID: id, Version: ThinkTankTemplateVersion}
	var raw string
	err := s.db.QueryRow(`SELECT name, setup FROM think_tank_templates WHERE template_id = ?`, id).Scan(&t.Name, &raw)
	if errors.Is(err, sql.ErrNoRows) {
		return ThinkTankTemplate{}, ErrNotFound
	} else if err != nil {
		return ThinkTankTemplate{}, fmt.Errorf("state: read template: %w", err)
	}
	if err := json.Unmarshal([]byte(raw), &t.Setup); err != nil {
		return ThinkTankTemplate{}, fmt.Errorf("state: decode template %s: %w", id, err)
	}
	if t.Setup.Participants == nil {
		t.Setup.Participants = []ThinkTankTemplateParticipant{}
	}
	return t, nil
}
