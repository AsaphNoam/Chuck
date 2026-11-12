package server

import (
	"encoding/json"

	"github.com/AsaphNoam/Chuck/internal/runtime"
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
func forkAffordance(ev runtime.Event, currentSession string) *runtime.ForkAffordance {
	if ev.Type != runtime.EvTurnEnd || ev.ActivityID != "" {
		return nil
	}
	var td struct {
		StopReason string `json:"stop_reason"`
	}
	_ = json.Unmarshal(ev.Data, &td)
	reason := ""
	switch fp := ev.ForkPoint; {
	case td.StopReason != "end_turn":
		reason = runtime.ForkReasonNotTurnBoundary
	case fp != nil && fp.Copied:
		reason = runtime.ForkReasonCopiedTurn
	case fp == nil:
		reason = runtime.ForkReasonNoForkPoint
	case fp.SessionID == "" || fp.SessionID != currentSession:
		reason = runtime.ForkReasonOtherSession
	case !runtime.ForkPointAdapterVerified(fp.AdapterName, fp.AdapterVersion):
		reason = runtime.ForkReasonUnverifiedAdapter
	default:
		return &runtime.ForkAffordance{Available: true}
	}
	return &runtime.ForkAffordance{Reason: reason}
}

// currentNativeSession is the agent's latest provider session id, "" when
// unknown.
func (s *Server) currentNativeSession(agentID string) string {
	snap, err := s.stateStore.ReadSession(agentID)
	if err != nil {
		return ""
	}
	return snap.LastSessionID
}

// decorateFork marks a root turn_end with its fork affordance for clients.
func (s *Server) decorateFork(ev *runtime.Event, currentSession string) {
	ev.Fork = forkAffordance(*ev, currentSession)
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
