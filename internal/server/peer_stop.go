package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/AsaphNoam/Chuck/internal/messaging"
	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
	"github.com/AsaphNoam/Chuck/internal/transcript"
)

// maxPeerStops bounds outstanding peer Stop requests server-wide; each
// requester may hold one (TS-04.R94, INV §16).
const maxPeerStops = 32

// peerExecution is the target execution a peer Stop approval is bound to. It
// is read from the durable running row, never from tool arguments.
type peerExecution struct {
	generation string
	pid        int
	startedAt  time.Time
}

// StopPeer implements messaging.PeerControl: an agent-requested Stop of one
// other agent that only a specific operator decision can authorize, executed
// through the ordinary Stop path (FS-06.R41–R44, TS-04.R94–R95).
func (s *Server) StopPeer(ctx context.Context, requester messaging.SessionIdentity, targetID, reason string) messaging.PeerStopResult {
	target, err := s.stateStore.ReadHistoryAgent(targetID)
	if errors.Is(err, state.ErrNotFound) {
		return messaging.PeerStopResult{Code: "agent_not_found", Message: "No Chuck agent has that agent_id."}
	}
	if err != nil {
		return peerStopUnchanged("", "Chuck could not read the target agent.")
	}
	from, err := s.stateStore.ReadAgent(requester.AgentID)
	if err != nil {
		return peerStopUnchanged("", "Chuck could not read your agent identity.")
	}
	exec, stopped, err := s.peerExecution(targetID)
	if err != nil {
		return peerStopUnchanged("", "The target's running state could not be verified, so nothing was stopped.")
	}
	if stopped {
		return messaging.PeerStopResult{Outcome: "already_stopped"}
	}
	if !s.admitPeerStop(requester.AgentID) {
		return messaging.PeerStopResult{Code: "peer_control_busy",
			Message: "Another peer Stop request is already waiting; finish it first."}
	}
	defer s.releasePeerStop(requester.AgentID)

	chat := s.registry.Chat()
	if chat == nil {
		return peerStopUnchanged("", "Peer Stop is unavailable.")
	}
	args, _ := json.Marshal(map[string]string{"agent_id": targetID, "reason": reason})
	hp, err := chat.RequestHostPermission(requester.AgentID, runtime.HostPermissionRequest{
		Generation: requester.Generation, Origin: runtime.PermissionOriginPeerStop,
		Name: "Stop " + clipPreview(target.Name, 80), Reason: reason, Args: args,
		PeerStop: &runtime.PeerStopSubject{
			Requester: runtime.PeerIdentity{AgentID: from.AgentID, Name: clipPreview(from.Name, 80), Project: from.Project},
			Target:    runtime.PeerIdentity{AgentID: target.AgentID, Name: clipPreview(target.Name, 80), Project: target.Project},
		},
	})
	if err != nil {
		return messaging.PeerStopResult{Code: "peer_stop_stale",
			Message: "Your session or turn is no longer current, so no Stop was requested."}
	}
	switch hp.Wait(ctx) {
	case runtime.HostApprove:
	case runtime.HostDeny:
		return messaging.PeerStopResult{Code: "peer_stop_denied", RequestID: hp.ID,
			Message: "The user denied this Stop. Nothing was stopped."}
	default:
		return messaging.PeerStopResult{Code: "peer_stop_cancelled", RequestID: hp.ID,
			Message: "The Stop request ended without a decision. Nothing was stopped."}
	}
	if err := hp.Admit(); err != nil {
		return messaging.PeerStopResult{Code: "peer_stop_stale", RequestID: hp.ID,
			Message: "Your turn ended before the approved Stop could start. Nothing was stopped."}
	}
	// The approved stop runs to completion even if the caller goes away.
	return s.executePeerStop(context.WithoutCancel(ctx), hp.ID, from, target, exec)
}

func (s *Server) executePeerStop(ctx context.Context, requestID string, from state.Agent, target state.HistoryAgent, exec peerExecution) messaging.PeerStopResult {
	release, ok := s.claimQuotaActions(quotaAgentKeys([]string{target.AgentID}))
	if !ok {
		return peerStopUnchanged(requestID, "Another lifecycle action on the target is in progress. Nothing was stopped.")
	}
	defer release()
	if !s.claimLifecycle(target.AgentID) {
		return peerStopUnchanged(requestID, "Another lifecycle action on the target is in progress. Nothing was stopped.")
	}
	defer s.releaseLifecycle(target.AgentID)

	now, stopped, err := s.peerExecution(target.AgentID)
	switch {
	case err != nil:
		return peerStopUnchanged(requestID, "The target's running state could not be verified, so nothing was stopped.")
	case stopped:
		return messaging.PeerStopResult{Outcome: "already_stopped", RequestID: requestID}
	case now.generation != exec.generation || now.pid != exec.pid || !now.startedAt.Equal(exec.startedAt):
		return messaging.PeerStopResult{Code: "peer_stop_stale", RequestID: requestID,
			Message: "The target restarted after the request; the approval does not cover the new run. Nothing was stopped."}
	}

	notice := func(phase, title, description string) runtime.NoticeData {
		return runtime.NoticeData{Severity: "info", Title: title, Description: description, Origin: "chuck",
			PeerStop: &runtime.PeerStopNotice{RequestID: requestID, RequesterID: from.AgentID, TargetID: target.AgentID, Phase: phase}}
	}
	approved := notice("approved", "The user approved a Stop requested by another agent",
		fmt.Sprintf("%s (%s, %s) asked to stop this agent and the user approved it once.",
			clipPreview(from.Name, 80), from.Project, from.AgentID))
	if err := s.appendPeerNotice(target.AgentID, exec.generation, approved); err != nil {
		s.log.Warn("peer stop approval notice", "target", target.AgentID, "err", err)
		return peerStopUnchanged(requestID, "Chuck could not record the approval on the target, so nothing was stopped.")
	}
	if ae := s.stopAgentClaimed(ctx, target.AgentID); ae != nil {
		s.log.Warn("peer stop", "target", target.AgentID, "err", ae.Message)
		return messaging.PeerStopResult{Code: "peer_stop_failed", RequestID: requestID, Effect: "unknown",
			Message: "Stop started but Chuck could not confirm its result. Check the target before asking again."}
	}
	done := notice("stopped", "Stopped "+clipPreview(target.Name, 80)+" after your approval",
		fmt.Sprintf("Stopped %s (%s, %s). Its history is kept and it can be woken later.",
			clipPreview(target.Name, 80), target.Project, target.AgentID))
	if _, err := s.registry.AppendPeerStopNoticeAndSync(from.AgentID, done); err != nil {
		s.log.Warn("peer stop completion notice", "requester", from.AgentID, "err", err)
		return messaging.PeerStopResult{Code: "peer_stop_failed", RequestID: requestID, Effect: "stopped",
			Message: "The target stopped, but Chuck could not record the completion. Do not ask again."}
	}
	return messaging.PeerStopResult{Outcome: "stopped", RequestID: requestID}
}

// peerExecution resolves the target's current execution identity. A missing
// running row with no owned runtime is stopped; an owned runtime without a
// matching row is conflicting state that refuses rather than kills.
func (s *Server) peerExecution(agentID string) (peerExecution, bool, error) {
	live := s.registry.Generation(agentID)
	row, err := s.stateStore.ReadRunning(agentID)
	if errors.Is(err, state.ErrNotFound) {
		if live != "" {
			return peerExecution{}, false, fmt.Errorf("owned runtime has no running row")
		}
		return peerExecution{}, true, nil
	}
	if err != nil {
		return peerExecution{}, false, err
	}
	if live != "" && live != row.Generation {
		return peerExecution{}, false, fmt.Errorf("owned generation differs from running row")
	}
	return peerExecution{generation: row.Generation, pid: row.PID, startedAt: row.StartedAt}, false, nil
}

// appendPeerNotice records a peer Stop notice on the target before any effect.
// A chat owner appends through its own writer. Otherwise no event writer is
// active — a terminal runtime appends nothing after its session record, and an
// orphan or stopped agent has none — and the caller holds the lifecycle claim.
func (s *Server) appendPeerNotice(agentID, generation string, data runtime.NoticeData) error {
	_, err := s.registry.AppendPeerStopNoticeAndSync(agentID, data)
	if err == nil || (!errors.Is(err, runtime.ErrNoHandle) && !errors.Is(err, runtime.ErrNotImplemented)) {
		return err
	}
	if !data.PeerStop.Valid() {
		return fmt.Errorf("invalid peer stop notice")
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	w, err := transcript.Open(s.configStore.Home(), agentID, nil)
	if err != nil {
		return err
	}
	defer w.Close()
	ev := runtime.Event{AgentID: agentID, Generation: generation, Seq: w.NextSeq(), Type: runtime.EvNotice,
		Data: raw, Ts: time.Now().UTC().Format(time.RFC3339)}
	if err := w.Append(ev); err != nil {
		return err
	}
	if err := w.Sync(); err != nil {
		return err
	}
	if s.indexer != nil {
		return s.indexer.OnEvent(agentID, ev)
	}
	return nil
}

func (s *Server) admitPeerStop(requesterID string) bool {
	s.peerStopMu.Lock()
	defer s.peerStopMu.Unlock()
	if s.peerStopRequesters[requesterID] || len(s.peerStopRequesters) >= maxPeerStops {
		return false
	}
	s.peerStopRequesters[requesterID] = true
	return true
}

func (s *Server) releasePeerStop(requesterID string) {
	s.peerStopMu.Lock()
	delete(s.peerStopRequesters, requesterID)
	s.peerStopMu.Unlock()
}

func peerStopUnchanged(requestID, message string) messaging.PeerStopResult {
	return messaging.PeerStopResult{Code: "peer_stop_failed", RequestID: requestID, Effect: "unchanged", Message: message}
}
