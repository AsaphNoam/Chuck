package state

import (
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
	fromRoom, err := st.SaveThinkTankTemplate(ThinkTankTemplateSave{CommandID: "save-room", Name: "From room", RoomID: room.Room.RoomID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.EndThinkTank(room.Room.RoomID); err != nil {
		t.Fatal(err)
	}
	if err := st.DeleteThinkTank(room.Room.RoomID); err != nil {
		t.Fatal(err)
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
