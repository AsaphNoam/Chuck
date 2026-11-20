package server

import (
	"encoding/json"

	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
)

// cloneRequest is Clone's optional turn boundary (TS-03.R86). An empty body is
// today's Clone at the latest completed turn.
type cloneRequest struct {
	ThroughSeq     int64 `json:"through_seq"`
	ComposeFromSeq int64 `json:"compose_from_seq"`
}

// forkAffordance is the turn-specific half of Edit/Fork from here eligibility,
// computed from the stored fork point (TS-03.R86, TS-04.R110). Clone's own
// refusals (busy, terminal, no native session) stay on the agent's clone
// affordance. Nil for any event that is not a root turn_end.
func forkAffordance(ev runtime.Event, sessions forkSessions) *runtime.ForkAffordance {
	if ev.Type != runtime.EvTurnEnd || ev.ActivityID != "" {
		return nil
	}
	outcome, _, _ := runtime.DecodeTurnEnd(ev)
	reason := ""
	switch fp := ev.ForkPoint; {
	case outcome != runtime.TurnCompleted:
		reason = runtime.ForkReasonNotTurnBoundary
	case fp != nil && fp.Copied:
		reason = runtime.ForkReasonCopiedTurn
	case fp == nil:
		reason = runtime.ForkReasonNoForkPoint
	case !sessions.holds(fp.SessionID, ev.Seq):
		reason = runtime.ForkReasonOtherSession
	case !runtime.ForkPointAdapterVerified(fp.AdapterName, fp.AdapterVersion):
		reason = runtime.ForkReasonUnverifiedAdapter
	default:
		return &runtime.ForkAffordance{Available: true}
	}
	return &runtime.ForkAffordance{Reason: reason}
}

// forkSessions are the native sessions a recorded fork point may name: the
// current one, and each older session a committed Edit continued for the
// turns it kept (TS-04.R113). Forking such a point forks its own session;
// the fork's replay fingerprint still proves the turn.
type forkSessions struct {
	current string
	lineage []state.EditLineageStep // newest first
}

// holds reports whether a fork point in session at seq is in the current
// conversation's provider context.
func (f forkSessions) holds(session string, seq int64) bool {
	if session == "" {
		return false
	}
	chain := f.current
	for _, step := range f.lineage {
		if session == chain {
			return true
		}
		if step.CandidateSession == chain && seq < step.KeptBefore {
			chain = step.OldSession
		}
	}
	return session == chain
}

// currentForkSessions reads the agent's fork sessions; empty when unknown.
func (s *Server) currentForkSessions(agentID string) forkSessions {
	snap, err := s.stateStore.ReadSession(agentID)
	if err != nil {
		return forkSessions{}
	}
	return s.forkSessionsFor(agentID, snap.LastSessionID)
}

func (s *Server) forkSessionsFor(agentID, current string) forkSessions {
	lineage, err := s.stateStore.EditLineage(agentID)
	if err != nil {
		s.log.Warn("read edit lineage", "agent", agentID, "err", err)
	}
	return forkSessions{current: current, lineage: lineage}
}

// forkSource is the session a fork at point starts from: the point's own
// session, which forkSessions.holds already related to the current one.
func forkSource(point *runtime.ForkPoint, current string) string {
	if point != nil && point.SessionID != "" {
		return point.SessionID
	}
	return current
}

// decorateTurnEnd marks a root turn_end with its fork affordance and decoded
// outcome for clients, on live events and transcript reads alike (TS-01.R42).
func (s *Server) decorateTurnEnd(ev *runtime.Event, sessions forkSessions) {
	ev.Fork = forkAffordance(*ev, sessions)
	ev.Outcome = ""
	if ev.Type == runtime.EvTurnEnd && ev.ActivityID == "" {
		outcome, _, _ := runtime.DecodeTurnEnd(*ev)
		ev.Outcome = outcome.Label()
	}
}

// publishHistory sends one committed event to live clients. A live root
// turn_end carries the same fork affordance and decoded outcome as a transcript
// read (TS-03.R86, TS-01.R42).
func (s *Server) publishHistory(ev runtime.Event) {
	if ev.Type == runtime.EvTurnEnd && ev.ActivityID == "" {
		s.decorateTurnEnd(&ev, s.currentForkSessions(ev.AgentID))
	}
	s.eventBus.PublishRuntimeEvent(ev)
}

// composeText returns the person's message at seq when it is the first root
// prompt after the boundary, ok false otherwise.
func composeTextAfter(prefixEvents []runtime.Event, boundary, seq int64) (string, bool) {
	for _, ev := range prefixEvents {
		if ev.Seq <= boundary || ev.ActivityID != "" || ev.Type != runtime.EvUserPrompt {
			continue
		}
		if ev.Seq != seq {
			return "", false
		}
		var p runtime.UserPromptData
		if json.Unmarshal(ev.Data, &p) != nil {
			return "", false
		}
		return p.Text, true
	}
	return "", false
}
