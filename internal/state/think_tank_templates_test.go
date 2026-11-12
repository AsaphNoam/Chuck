package state

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// FS-21.A43–A44, TS-14.R30–R31: a draft save resolves an existing chat to its
// authoritative launch choices; a room save captures current ceilings and
// retained launch intents. Neither copies identity, and exact replay returns
// the committed template after its sources are gone.
func TestThinkTankTemplateSaveFromDraftAndRoom(t *testing.T) {
	st, _ := newTestStore(t)
	if err := st.WriteAgent(Agent{AgentID: "a", Name: "Ari", Role: "critic", Project: "alpha", Backend: "claude",
		Model: "opus", Effort: "high", Fast: true, Interface: "chat"}); err != nil {
		t.Fatal(err)
	}
	fresh := &ThinkTankLaunchChoice{Role: "", Backend: "codex", Model: "gpt", Name: "Bea"}
	draft := ThinkTankTemplateSave{CommandID: "save-1", Name: "  Storage   team ", Draft: &ThinkTankTemplateDraft{
		Openings: true, Judge: &ThinkTankLaunchChoice{Backend: "claude"}, JudgeInstructions: "Rank them.",
		Participants: []ThinkTankTemplateDraftRow{
			{AgentID: "a", Limit: 2, MayLeave: true, Instructions: "Argue for SQLite."},
			{Launch: fresh, Limit: 3}}}}
	saved, err := st.SaveThinkTankTemplate(draft)
	if err != nil || saved.Name != "Storage team" {
		t.Fatalf("save = %+v %v", saved, err)
	}
	got, err := st.ReadThinkTankTemplate(saved.ID)
	if err != nil {
		t.Fatal(err)
	}
	ari := got.Setup.Participants[0]
	if ari.Launch != (ThinkTankLaunchChoice{Name: "Ari", Role: "critic", Backend: "claude", Model: "opus", Effort: "high", Fast: true}) ||
		ari.Limit != 2 || !ari.MayLeave || ari.Instructions != "Argue for SQLite." {
		t.Fatalf("existing-chat row = %+v", ari)
	}
	if got.Setup.Participants[1].Launch != *fresh || got.Setup.Participants[1].MayLeave || !got.Setup.Openings ||
		got.Setup.Judge == nil || got.Setup.JudgeInstructions != "Rank them." {
		t.Fatalf("setup = %+v", got.Setup)
	}
	assertTemplateRowOmits(t, st, saved.ID, `"agent_id"`)
	if err := st.DeleteAgent("a"); err != nil {
		t.Fatal(err)
	}
	if again, err := st.SaveThinkTankTemplate(draft); err != nil || again.ID != saved.ID {
		t.Fatalf("replay after source deletion = %+v %v", again, err)
	}
	draft.Name = "Other"
	if _, err := st.SaveThinkTankTemplate(draft); !errors.Is(err, ErrThinkTankConflict) {
		t.Fatalf("reused command err = %v", err)
	}
	draft.CommandID = "save-2"
	if _, err := st.SaveThinkTankTemplate(draft); !errors.Is(err, ErrThinkTankInvalid) || !strings.Contains(err.Error(), "participant 1") {
		t.Fatalf("unrecoverable existing chat err = %v", err)
	}

	room, err := st.CreateThinkTank(ThinkTankCreate{CommandID: "room", Goal: "Pick storage", OriginProject: "alpha",
		JudgeConfig: `{"role":"judge","project":"alpha","backend":"claude","interface":"chat"}`,
		Members: []ThinkTankMember{
			{AgentName: "Cy", Project: "alpha", Cap: 1, MayLeave: true, Instructions: "Be brief.",
				SetupConfig: `{"role":"critic","project":"alpha","backend":"codex","model":"gpt","name":"Cy","group":"g","interface":"chat"}`},
			{AgentName: "Di", Project: "beta", Cap: 2, SetupConfig: `{"project":"beta","backend":"claude"}`}}})
	if err != nil {
		t.Fatal(err)
	}
	cy := room.Members[0]
	// A live ceiling increase (FS-21.R49) is a durable cap change; set it
	// directly so this test stays independent of setup launches.
	if _, err := st.db.Exec(`UPDATE think_tank_members SET cap = 4 WHERE agent_id = ?`, cy.AgentID); err != nil {
		t.Fatal(err)
	}
	roomSave := ThinkTankTemplateSave{CommandID: "save-room", Name: "From room", RoomID: room.Room.RoomID}
	fromRoom, err := st.SaveThinkTankTemplate(roomSave)
	if err != nil {
		t.Fatal(err)
	}
	assertTemplateRowOmits(t, st, fromRoom.ID, room.Room.RoomID)
	if _, err := st.EndThinkTank(room.Room.RoomID); err != nil {
		t.Fatal(err)
	}
	if err := st.DeleteThinkTank(room.Room.RoomID); err != nil {
		t.Fatal(err)
	}
	if again, err := st.SaveThinkTankTemplate(roomSave); err != nil || again.ID != fromRoom.ID {
		t.Fatalf("room replay after deletion = %+v %v", again, err)
	}
	roomSave.Name = "Renamed"
	if _, err := st.SaveThinkTankTemplate(roomSave); !errors.Is(err, ErrThinkTankConflict) {
		t.Fatalf("reused room command err = %v", err)
	}
	got, err = st.ReadThinkTankTemplate(fromRoom.ID)
	if err != nil {
		t.Fatal(err)
	}
	p := got.Setup.Participants
	if len(p) != 2 || p[0].Limit != 4 || p[0].Instructions != "Be brief." ||
		p[0].Launch != (ThinkTankLaunchChoice{Role: "critic", Backend: "codex", Model: "gpt", Name: "Cy"}) ||
		got.Setup.Judge == nil || got.Setup.Judge.Role != "judge" {
		t.Fatalf("room template = %+v", got.Setup)
	}
}

func TestThinkTankTemplateValidationAndPaging(t *testing.T) {
	st, _ := newTestStore(t)
	row := ThinkTankTemplateDraftRow{Launch: &ThinkTankLaunchChoice{Backend: "claude"}, Limit: 1}
	bad := []ThinkTankTemplateSave{
		{CommandID: "x", Name: " ", Draft: &ThinkTankTemplateDraft{Participants: []ThinkTankTemplateDraftRow{row, row}}},
		{CommandID: "x", Name: "n"},
		{CommandID: "x", Name: "n", RoomID: "tt_x", Draft: &ThinkTankTemplateDraft{}},
		{CommandID: "x", Name: "n", Draft: &ThinkTankTemplateDraft{Participants: []ThinkTankTemplateDraftRow{row}}},
		{CommandID: "x", Name: "n", Draft: &ThinkTankTemplateDraft{Participants: []ThinkTankTemplateDraftRow{row, row}, JudgeInstructions: "x"}},
		{CommandID: "x", Name: strings.Repeat("é", ThinkTankTemplateMaxName+1), Draft: &ThinkTankTemplateDraft{Participants: []ThinkTankTemplateDraftRow{row, row}}},
	}
	for i, req := range bad {
		if _, err := st.SaveThinkTankTemplate(req); !errors.Is(err, ErrThinkTankInvalid) {
			t.Fatalf("case %d: err = %v", i, err)
		}
	}
	if _, err := st.SaveThinkTankTemplate(ThinkTankTemplateSave{CommandID: "x", Name: "n", RoomID: "tt_missing"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing room err = %v", err)
	}
	for _, id := range []string{"1", "2", "3"} {
		if _, err := st.SaveThinkTankTemplate(ThinkTankTemplateSave{CommandID: id, Name: "T" + id,
			Draft: &ThinkTankTemplateDraft{Participants: []ThinkTankTemplateDraftRow{row, row}}}); err != nil {
			t.Fatal(err)
		}
	}
	var names []string
	cursor := ""
	for {
		page, next, err := st.ListThinkTankTemplates(cursor, 2)
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range page {
			names = append(names, s.Name)
		}
		if next == "" {
			break
		}
		cursor = next
	}
	if strings.Join(names, ",") != "T3,T2,T1" {
		t.Fatalf("paged names = %v", names)
	}
	if _, _, err := st.ListThinkTankTemplates("bogus", 2); !errors.Is(err, ErrThinkTankInvalid) {
		t.Fatalf("bad cursor err = %v", err)
	}
	if _, err := st.ReadThinkTankTemplate("ttt_missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing template err = %v", err)
	}
}

// assertTemplateRowOmits checks the stored row, not just the API shape: the
// replay identity is a fingerprint, so source ids never persist (TS-14.R30).
func assertTemplateRowOmits(t *testing.T, st *Store, id, source string) {
	t.Helper()
	var intent, setup string
	if err := st.db.QueryRow(`SELECT intent, setup FROM think_tank_templates WHERE template_id = ?`, id).Scan(&intent, &setup); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(intent, "sha256:") || strings.Contains(intent+setup, source) {
		t.Fatalf("stored template carries %q: intent=%s setup=%s", source, intent, setup)
	}
}

// FS-21.R60, TS-14.R31: an existing-chat member that switched to Terminal
// after the discussion still contributes its recoverable launch choices to a
// room save; a draft that selects a terminal agent is still refused.
func TestThinkTankTemplateRoomSaveAfterMemberSwitch(t *testing.T) {
	st, _ := newTestStore(t)
	ari := Agent{AgentID: "a", Name: "Ari", Role: "critic", Project: "alpha", Backend: "claude",
		Model: "opus", Effort: "high", Interface: "chat"}
	if err := st.WriteAgent(ari); err != nil {
		t.Fatal(err)
	}
	room, err := st.CreateThinkTank(ThinkTankCreate{CommandID: "room", Goal: "Pick storage", OriginProject: "alpha",
		Members: []ThinkTankMember{{AgentID: "a", AgentName: "Ari", Project: "alpha", Cap: 2},
			{AgentName: "Bea", Project: "alpha", Cap: 1, SetupConfig: `{"backend":"codex"}`}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.EndThinkTank(room.Room.RoomID); err != nil {
		t.Fatal(err)
	}
	// The retained member's switch persists its new interface (server switch).
	ari.Interface = "terminal"
	if err := st.WriteAgent(ari); err != nil {
		t.Fatal(err)
	}
	saved, err := st.SaveThinkTankTemplate(ThinkTankTemplateSave{CommandID: "s", Name: "Ended", RoomID: room.Room.RoomID})
	if err != nil {
		t.Fatalf("room save after switch: %v", err)
	}
	got, err := st.ReadThinkTankTemplate(saved.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Setup.Participants[0].Launch != (ThinkTankLaunchChoice{Name: "Ari", Role: "critic", Backend: "claude", Model: "opus", Effort: "high"}) {
		t.Fatalf("switched member = %+v", got.Setup.Participants[0])
	}
	row := ThinkTankTemplateDraftRow{Launch: &ThinkTankLaunchChoice{Backend: "claude"}, Limit: 1}
	_, err = st.SaveThinkTankTemplate(ThinkTankTemplateSave{CommandID: "d", Name: "Draft",
		Draft: &ThinkTankTemplateDraft{Participants: []ThinkTankTemplateDraftRow{{AgentID: "a", Limit: 1}, row}}})
	if !errors.Is(err, ErrThinkTankInvalid) {
		t.Fatalf("draft with a terminal agent err = %v", err)
	}
}

// TS-14.R30: a row written before the version column and fingerprinted
// intent existed still decodes as v1, replays its command exactly and keeps
// no source id after migration.
func TestThinkTankTemplateLegacyRowMigratesAndDecodes(t *testing.T) {
	st, _ := newTestStore(t)
	req := ThinkTankTemplateSave{CommandID: "legacy", Name: "Old", RoomID: "tt_source_room"}
	legacyIntent, _ := json.Marshal(req)
	// Frozen v1 payload: the decoder must keep accepting this exact shape.
	const v1 = `{"participants":[{"launch":{"role":"critic","backend":"claude","model":"opus","effort":"high","fast":true,"name":"Ari"},"limit":2,"may_leave":true,"instructions":"Be brief."},{"launch":{"role":"","backend":"codex","model":"","effort":"","fast":false},"limit":1,"may_leave":false}],"openings":true,"judge":{"role":"judge","backend":"claude","model":"","effort":"","fast":false},"judge_instructions":"Rank them."}`
	tx, err := st.db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(`INSERT INTO think_tank_templates(template_id, command_id, name, intent, setup, created_at)
VALUES('ttt_legacy', 'legacy', 'Old', ?, ?, '2026-11-12T00:00:00Z')`, string(legacyIntent), v1); err != nil {
		t.Fatal(err)
	}
	if err := fingerprintLegacyTemplateIntents(tx); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	assertTemplateRowOmits(t, st, "ttt_legacy", "tt_source_room")
	if again, err := st.SaveThinkTankTemplate(req); err != nil || again.ID != "ttt_legacy" {
		t.Fatalf("legacy replay = %+v %v", again, err)
	}
	req.Name = "Other"
	if _, err := st.SaveThinkTankTemplate(req); !errors.Is(err, ErrThinkTankConflict) {
		t.Fatalf("legacy conflict err = %v", err)
	}
	got, err := st.ReadThinkTankTemplate("ttt_legacy")
	if err != nil {
		t.Fatal(err)
	}
	p := got.Setup.Participants
	if got.Version != 1 || len(p) != 2 || p[0].Launch.Name != "Ari" || !p[0].Launch.Fast || p[0].Instructions != "Be brief." ||
		!got.Setup.Openings || got.Setup.Judge == nil || got.Setup.JudgeInstructions != "Rank them." {
		t.Fatalf("v1 decode = %+v", got)
	}
	if _, err := st.db.Exec(`UPDATE think_tank_templates SET version = 99 WHERE template_id = 'ttt_legacy'`); err != nil {
		t.Fatal(err)
	}
	if _, err := st.ReadThinkTankTemplate("ttt_legacy"); err == nil {
		t.Fatal("unknown payload version decoded as current")
	}
}
