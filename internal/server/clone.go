package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
	"github.com/AsaphNoam/Chuck/internal/strutil"
	"github.com/AsaphNoam/Chuck/internal/transcript"
)

// cloneResponse is the ordinary new-session envelope plus the fork lineage
// (TS-03.R43).
type cloneResponse struct {
	sessionResponse
	HistoryHandoff    string `json:"history_handoff"`
	ForkedFromAgentID string `json:"forked_from_agent_id"`
	ForkedFromSeq     int64  `json:"forked_from_seq"`
	// ComposeText is Edit's message text for the new agent's composer, unsent
	// (TS-03.R86); absent for Clone and Fork from here.
	ComposeText string `json:"compose_text,omitempty"`
}

// handleClone implements POST /api/sessions/{id}/clone: a new agent whose
// provider conversation is a native fork of the source's at its latest
// completed turn (FS-01.R36). The request carries no provider or session id;
// every setting and the native source id come from the stored source.
func (s *Server) handleClone(w http.ResponseWriter, r *http.Request) {
	var req cloneRequest
	if r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			writeAPIError(w, apiError(runtime.CodeValidation, "invalid clone body"))
			return
		}
	}
	if req.ThroughSeq < 0 || req.ComposeFromSeq < 0 || req.ComposeFromSeq != 0 && req.ThroughSeq == 0 {
		writeAPIError(w, apiError(runtime.CodeValidation, "compose_from_seq requires through_seq"))
		return
	}
	resp, ae := s.cloneAgent(r.Context(), r.PathValue("id"), req)
	if ae != nil {
		writeAPIError(w, ae)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (s *Server) cloneAgent(ctx context.Context, sourceID string, req cloneRequest) (cloneResponse, *runtime.APIError) {
	source, err := s.stateStore.ReadAgent(sourceID)
	if err != nil {
		return cloneResponse{}, apiError(runtime.CodeNotFound, "no such agent: "+sourceID)
	}
	// One exclusive lifecycle claim over the source: resume, switch, stop and
	// a second clone cannot move it while its conversation point is copied.
	if !s.claimLifecycle(sourceID) {
		return cloneResponse{}, apiError(runtime.CodeTransitionInProgress, "another lifecycle change is in progress for this agent")
	}
	defer s.releaseLifecycle(sourceID)

	snap, err := s.stateStore.ReadSession(sourceID)
	if err != nil && !errors.Is(err, state.ErrNotFound) {
		return cloneResponse{}, apiError(runtime.CodeInternal, "read source session")
	}
	busy := false
	if status, err := s.stateStore.ReadStatus(sourceID); err == nil {
		busy = state.CloneBusyState(status.State)
	}
	if busy && source.Interface == "chat" && !source.Archived {
		return cloneResponse{}, apiError(runtime.CodeAgentBusy, state.CloneReasonBusy)
	}
	if aff := state.CloneAvailability(source.Interface, source.Archived, snap.RuntimeCapabilities, snap.LastSessionID, busy); !aff.Available {
		return cloneResponse{}, cloneUnavailable(aff.Reason)
	}

	// The fork boundary is the named completed turn, or the source's latest; the
	// clone gets a durable copy of the visible transcript through it.
	prefix, boundary, err := transcript.CloneCompletedPrefix(s.configStore.Home(), sourceID, req.ThroughSeq)
	if err != nil {
		return cloneResponse{}, apiError(runtime.CodeInternal, "read source transcript")
	}
	if boundary.Seq == 0 {
		if req.ThroughSeq != 0 {
			return cloneResponse{}, cloneUnavailable(runtime.ForkReasonNotTurnBoundary)
		}
		return cloneResponse{}, cloneUnavailable(state.CloneReasonNoSession)
	}
	plan := &runtime.ForkPlan{
		SourceAgentID: sourceID, SourceSessionID: snap.LastSessionID, SourceSeq: boundary.Seq, Prefix: prefix,
		HistoryPrimer: snap.HistoryPrimer,
	}
	composeText := ""
	if req.ThroughSeq != 0 {
		// A chosen turn forks at its stored provider point (TS-04.R110).
		if aff := forkAffordance(boundary, snap.LastSessionID); !aff.Available {
			return cloneResponse{}, cloneUnavailable(aff.Reason)
		}
		plan.ForkPoint = boundary.ForkPoint
		if req.ComposeFromSeq != 0 {
			after, err := transcript.ReadFile(s.configStore.Home(), sourceID, transcript.ReadOptions{SinceSeq: boundary.Seq})
			if err != nil {
				return cloneResponse{}, apiError(runtime.CodeInternal, "read source transcript")
			}
			text, ok := composeTextAfter(after, boundary.Seq, req.ComposeFromSeq)
			if !ok {
				return cloneResponse{}, cloneUnavailable(runtime.ForkReasonInjectedMessage)
			}
			composeText = text
		}
	}

	resp, ae := s.launchClaimed(ctx, launchRequest{
		Name: cloneName(source.Name), Role: source.Role, Project: source.Project, Backend: source.Backend, Model: source.Model,
		Effort: source.Effort, Fast: source.Fast, Interface: source.Interface, Group: source.Group,
	}, launchOptions{ChuckAware: &snap.ChuckAware, Source: &snap, Fork: plan})
	if ae != nil {
		return cloneResponse{}, ae
	}
	return cloneResponse{sessionResponse: resp, HistoryHandoff: "native_fork", ForkedFromAgentID: sourceID,
		ForkedFromSeq: boundary.Seq, ComposeText: composeText}, nil
}

// cloneName names a clone after its source, shortening the source name so the
// result stays within the display-name limit; an unnamed source gets the
// curated suggestion (FS-01.R39).
func cloneName(source string) string {
	const suffix = " Copy"
	source = strings.TrimSpace(source)
	if source == "" {
		return ""
	}
	return strutil.ClipRunes(source, maxAgentNameRunes-utf8.RuneCountInString(suffix)) + suffix
}

func cloneUnavailable(reason string) *runtime.APIError {
	ae := apiError(runtime.CodeCloneUnavailable, reason)
	ae.Details = map[string]any{"reason": reason}
	return ae
}
