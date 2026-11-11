package server

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
)

// Think Tank setup templates (FS-21.R60–R61, TS-14.R30–R31). Local
// dashboard routes only; no remote, CLI, agent-side or SSE surface.

type thinkTankTemplateSetupRequest struct {
	Participants      []thinkTankParticipantRequest `json:"participants"`
	Openings          bool                          `json:"openings"`
	Judge             *launchRequest                `json:"judge,omitempty"`
	JudgeInstructions string                        `json:"judge_instructions,omitempty"`
}

type thinkTankTemplateSaveRequest struct {
	CommandID string                         `json:"command_id"`
	Name      string                         `json:"name"`
	RoomID    string                         `json:"room_id,omitempty"`
	Setup     *thinkTankTemplateSetupRequest `json:"setup,omitempty"`
}

// templateLaunchChoice keeps only the portable fields of a launch draft.
func templateLaunchChoice(l *launchRequest) *state.ThinkTankLaunchChoice {
	if l == nil {
		return nil
	}
	return &state.ThinkTankLaunchChoice{Role: l.Role, Backend: l.Backend, Model: l.Model, Effort: l.Effort,
		Fast: l.Fast, Name: strings.TrimSpace(l.Name)}
}

func (s *Server) handleSaveThinkTankTemplate(w http.ResponseWriter, r *http.Request) {
	var req thinkTankTemplateSaveRequest
	if !decodeThinkTankBody(w, r, &req) {
		return
	}
	save := state.ThinkTankTemplateSave{CommandID: req.CommandID, Name: req.Name, RoomID: req.RoomID}
	if req.Setup != nil {
		draft := &state.ThinkTankTemplateDraft{Participants: []state.ThinkTankTemplateDraftRow{}, Openings: req.Setup.Openings,
			Judge: templateLaunchChoice(req.Setup.Judge), JudgeInstructions: strings.TrimSpace(req.Setup.JudgeInstructions)}
		for _, p := range req.Setup.Participants {
			draft.Participants = append(draft.Participants, state.ThinkTankTemplateDraftRow{AgentID: p.AgentID,
				Launch: templateLaunchChoice(p.New), Limit: p.Limit, MayLeave: p.MayLeave == nil || *p.MayLeave,
				Instructions: strings.TrimSpace(p.Instructions)})
		}
		save.Draft = draft
	}
	summary, err := s.stateStore.SaveThinkTankTemplate(save)
	if err != nil {
		s.writeThinkTankError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, summary)
}

func (s *Server) handleThinkTankTemplates(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	items, next, err := s.stateStore.ListThinkTankTemplates(q.Get("cursor"), limit)
	if err != nil {
		s.writeThinkTankError(w, err)
		return
	}
	out := map[string]any{"version": state.ThinkTankTemplateVersion, "templates": items}
	if next != "" {
		out["next_cursor"] = next
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleThinkTankTemplate(w http.ResponseWriter, r *http.Request) {
	t, err := s.stateStore.ReadThinkTankTemplate(r.PathValue("id"))
	if errors.Is(err, state.ErrNotFound) {
		writeAPIError(w, apiError(runtime.CodeNotFound, "no such template"))
		return
	} else if err != nil {
		s.writeThinkTankError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}
