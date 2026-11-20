package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/AsaphNoam/Chuck/internal/attachment"
	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
	"github.com/AsaphNoam/Chuck/internal/transcript"
)

// editStartTimeout bounds candidate preparation and any restore, like an
// ordinary start (TS-01.R51).
const editStartTimeout = 3 * time.Minute

// editBeforeDecision lets tests hold a prepared candidate at the decision
// boundary; nil in production.
var editBeforeDecision func(ctx context.Context, agentID string)

// editAfterDecision lets tests hold a decided edit before it publishes the
// revision; nil in production.
var editAfterDecision func(agentID string)

// editBeforePublish lets tests fail a committed candidate's publication; nil
// in production.
var editBeforePublish func(agentID string) error

// editReasonUnrepaired is the committed edit's attention reason when neither
// the live completion nor its in-process recovery could finish the revision.
const editReasonUnrepaired = "The edit was saved but its history could not be finished. Restart Chuck to repair it."

// editReasonNotStarted is the committed edit's attention reason when its
// revised conversation could not start and the replacement was offered back.
const editReasonNotStarted = "The edit was saved, but the revised conversation could not start, so your replacement was not sent. Resume the agent to continue."

// Edit message-level refusal reasons (FS-01.R61).
const (
	editReasonInjected   = "This message was sent while the agent was working; edit the message that began that turn."
	editReasonUnfinished = "Wait for this turn to finish."
	editReasonLaterTurn  = "The agent took another turn after this message, so editing it would hide that turn."
	editReasonNoProof    = "This agent's adapter cannot prove a complete earlier turn; update to the supported package to edit later messages."
	editReasonNoMessage  = "This conversation has no sent message to edit."
	// editReasonSearchUnavailable is the committed edit's attention reason
	// when its search rebuild failed (TS-02.R70).
	editReasonSearchUnavailable = "The edit was applied, but search cannot find this conversation until `chuck reindex` repairs it."
	editReasonAttachment        = "An attachment on this message is no longer available, so it cannot be edited. Send a new message instead."
)

type editRequest struct {
	RequestID        string `json:"request_id"`
	MessageSeq       int64  `json:"message_seq"`
	ExpectedRevision int64  `json:"expected_revision"`
	Text             string `json:"text"`
}

// editResult is the receipt-backed operation result (TS-03.R90).
type editResult struct {
	AgentID   string `json:"agent_id"`
	RequestID string `json:"request_id"`
	Phase     string `json:"phase"`
	Revision  int64  `json:"revision"`
	Delivery  string `json:"delivery"`
	Active    string `json:"active"`
	Reason    string `json:"reason,omitempty"`
	Text      string `json:"text,omitempty"`
}

func editPayloadHash(req editRequest) string {
	data, _ := json.Marshal([]any{"edit-last", req.MessageSeq, req.ExpectedRevision, req.Text})
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func editUnavailable(reason string) *runtime.APIError {
	ae := apiError(runtime.CodeEditUnavailable, reason)
	ae.Details = map[string]any{"reason": reason}
	return ae
}

// handleEditLast implements POST /api/sessions/{id}/edit-last: replace the
// latest sent message and regenerate its reply in the same conversation
// (FS-01.R61, TS-01.R51, TS-03.R90). Every refusal precedes acceptance; the
// transition then runs under the lifecycle claim and the idle-turn reservation.
func (s *Server) handleEditLast(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	r.Body = http.MaxBytesReader(w, r.Body, attachmentMessageLimit+4096)
	var req editRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, apiError(runtime.CodeValidation, "invalid edit body"))
		return
	}
	if !validRequestID(req.RequestID) || req.MessageSeq <= 0 || req.ExpectedRevision < 0 {
		writeAPIError(w, apiError(runtime.CodeValidation, "request_id, message_seq and expected_revision are required"))
		return
	}
	if strings.TrimSpace(req.Text) == "" || len(req.Text) > attachmentMessageLimit {
		writeAPIError(w, apiError(runtime.CodeValidation, "replacement text is required and must fit the message limit"))
		return
	}
	hash := editPayloadHash(req)
	if prior, err := s.stateStore.ReadEdit(id, req.RequestID); err == nil {
		if prior.PayloadHash != hash {
			writeAPIError(w, apiError(runtime.CodeRequestConflict, "request id already belongs to a different edit"))
			return
		}
		writeJSON(w, http.StatusAccepted, s.editResultFor(prior, false))
		return
	} else if !errors.Is(err, state.ErrNotFound) {
		writeAPIError(w, apiError(runtime.CodeInternal, "cannot read edit"))
		return
	}
	agent, err := s.stateStore.ReadAgent(id)
	if err != nil {
		writeAPIError(w, apiError(runtime.CodeNotFound, "no such agent: "+id))
		return
	}
	// The control is registered with the claim, under the same lock, so Stop
	// and Cancel reach the edit from its first instant (TS-01.R51).
	ctx, cancel := context.WithTimeout(context.Background(), editStartTimeout)
	ctl := &editControl{cancel: cancel, done: make(chan struct{})}
	s.lifecycleMu.Lock()
	if !s.claimLifecycleLocked(id) {
		s.lifecycleMu.Unlock()
		cancel()
		writeAPIError(w, apiError(runtime.CodeTransitionInProgress, "another lifecycle change is in progress for this agent"))
		return
	}
	s.editControls[id] = ctl
	s.lifecycleMu.Unlock()
	chat := s.registry.Chat()
	reserved := chat.ReserveEdit(id, req.RequestID)
	release := func() {
		cancel()
		if reserved {
			chat.ReleaseEdit(id, req.RequestID)
		}
		s.releaseLifecycle(id)
		s.lifecycleMu.Lock()
		if s.editControls[id] == ctl { // a later edit may already own the slot
			delete(s.editControls, id)
		}
		s.lifecycleMu.Unlock()
		close(ctl.done)
	}
	if !reserved {
		release()
		writeAPIError(w, editUnavailable(state.EditReasonEditing))
		return
	}
	plan, msg, snap, ae := s.validateEdit(agent, req)
	if ae != nil {
		release()
		writeAPIError(w, ae)
		return
	}
	data, _ := json.Marshal(msg)
	rec := state.EditTransition{AgentID: id, RequestID: req.RequestID, PayloadHash: hash, MessageJSON: string(data),
		MessageSeq: req.MessageSeq, ExpectedRevision: req.ExpectedRevision, OldSession: snap.LastSessionID}
	if err := s.stateStore.CreateEdit(rec); err != nil {
		release()
		switch {
		case errors.Is(err, state.ErrEditOpen):
			writeAPIError(w, editUnavailable(state.EditReasonEditing))
		case errors.Is(err, state.ErrEditExists):
			writeAPIError(w, apiError(runtime.CodeRequestConflict, "request id already belongs to a different edit"))
		default:
			writeAPIError(w, apiError(runtime.CodeInternal, "cannot accept edit"))
		}
		return
	}
	s.log.Info("edit accepted", "agent", id, "request_id", req.RequestID, "message_seq", req.MessageSeq)
	s.touchState(id)
	accepted, _ := s.stateStore.ReadEdit(id, req.RequestID)
	writeJSON(w, http.StatusAccepted, s.editResultFor(accepted, false))
	go func() {
		defer release()
		s.runEdit(ctx, ctl, agent, snap, accepted, plan, msg)
	}()
}

// editControl lets Stop and turn Cancel reach a running Edit, serialized
// against its commit decision (TS-01.R51).
type editControl struct {
	mu      sync.Mutex
	cancel  context.CancelFunc
	decided bool
	stop    bool
	done    chan struct{}
}

// editInterrupt is how a lifecycle action reaches a running Edit.
type editInterrupt int

const (
	// editCancel is turn Cancel: abort before the decision and restore the
	// source's running/stopped condition.
	editCancel editInterrupt = iota
	// editStop is Stop and group release: abort before the decision and
	// leave the agent stopped; after it, release the undispatched replacement.
	editStop
	// editStopBeforeDecision is an approved peer Stop: it aborts like Stop
	// before the decision, but the revised run it was not approved for is
	// left to finish its handoff.
	editStopBeforeDecision
)

// interrupt asks the edit to stop and reports whether the decision was still
// ahead.
func (c *editControl) interrupt(mode editInterrupt) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.decided {
		if mode == editStop {
			c.stop = true
		}
		return false
	}
	if mode != editCancel {
		c.stop = true
	}
	c.cancel()
	return true
}

// editRunning reports whether an Edit holds the agent's lifecycle claim.
func (s *Server) editRunning(id string) bool {
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	return s.editControls[id] != nil
}

// interruptEdit interrupts the agent's running Edit, if any, and waits for
// its worker to release the lifecycle claim. handled is false when no edit
// runs; beforeDecision reports whether the edit was aborted.
func (s *Server) interruptEdit(ctx context.Context, id string, mode editInterrupt) (handled, beforeDecision bool, ae *runtime.APIError) {
	s.lifecycleMu.Lock()
	ctl := s.editControls[id]
	s.lifecycleMu.Unlock()
	if ctl == nil {
		return false, false, nil
	}
	beforeDecision = ctl.interrupt(mode)
	s.log.Info("edit interrupted", "agent", id, "mode", int(mode), "before_decision", beforeDecision)
	select {
	case <-ctl.done:
		return true, beforeDecision, nil
	case <-ctx.Done():
		return true, beforeDecision, apiError(runtime.CodeTransitionInProgress, "the edit is still finishing; try again")
	}
}

// editPlan is the validated replacement: the superseded range and, for a
// later turn, the in-place fork ending at the kept turn.
type editPlan struct {
	from, through int64
	fork          *runtime.ForkPlan
}

// validateEdit checks the agent gate and the selected message against the
// active history under the reservation, before anything is accepted.
func (s *Server) validateEdit(agent state.Agent, req editRequest) (editPlan, runtime.Message, state.SessionSnapshot, *runtime.APIError) {
	var none editPlan
	id := agent.AgentID
	snap, err := s.stateStore.ReadSession(id)
	if err != nil && !errors.Is(err, state.ErrNotFound) {
		return none, runtime.Message{}, snap, apiError(runtime.CodeInternal, "read session")
	}
	update, err := s.stateMgr.Touch(id)
	view := update.AgentState
	if err != nil {
		return none, runtime.Message{}, snap, apiError(runtime.CodeInternal, "read agent state")
	}
	if gate := state.EditAvailability(view, snap.LastSessionID, false); !gate.Available {
		return none, runtime.Message{}, snap, editUnavailable(gate.Reason)
	}
	revision, err := s.stateStore.HistoryRevision(id)
	if err != nil {
		return none, runtime.Message{}, snap, apiError(runtime.CodeInternal, "read history revision")
	}
	events, err := transcript.ReadFile(s.configStore.Home(), id, transcript.ReadOptions{})
	if err != nil {
		return none, runtime.Message{}, snap, apiError(runtime.CodeInternal, "read conversation history")
	}
	var latest, previousEnd, endAfter *runtime.Event
	var tail int64
	endsAfter := 0 // root turn ends after the latest message
	for i := range events {
		ev := &events[i]
		if ev.Seq > tail {
			tail = ev.Seq
		}
		if ev.ActivityID != "" {
			continue
		}
		switch ev.Type {
		case runtime.EvUserPrompt:
			latest, endAfter, endsAfter = ev, nil, 0
		case runtime.EvTurnEnd:
			if latest == nil {
				previousEnd = ev
			} else {
				endsAfter++
				if endAfter == nil {
					endAfter = ev
				}
			}
		}
	}
	// The previous turn end is the last root turn_end before the latest message.
	previousEnd = nil
	beganTurn := true
	if latest != nil {
		for i := range events {
			ev := &events[i]
			if ev.ActivityID != "" || ev.Seq >= latest.Seq {
				continue
			}
			switch ev.Type {
			case runtime.EvTurnEnd:
				previousEnd, beganTurn = ev, true
			case runtime.EvUserPrompt:
				beganTurn = false
			}
		}
	}
	if latest == nil {
		return none, runtime.Message{}, snap, editUnavailable(editReasonNoMessage)
	}
	if latest.Seq != req.MessageSeq || revision != req.ExpectedRevision {
		ae := apiError(runtime.CodeEditConflict, "This conversation changed. Reload the latest messages to edit again.")
		ae.Details = map[string]any{"message_seq": latest.Seq, "history_revision": revision}
		return none, runtime.Message{}, snap, ae
	}
	if !beganTurn {
		return none, runtime.Message{}, snap, editUnavailable(editReasonInjected)
	}
	if endAfter == nil {
		return none, runtime.Message{}, snap, editUnavailable(editReasonUnfinished)
	}
	// The selected message must have started the latest turn: a later
	// application-initiated turn (mail, continuation) would be superseded.
	if endsAfter > 1 {
		return none, runtime.Message{}, snap, editUnavailable(editReasonLaterTurn)
	}
	var original runtime.UserPromptData
	if err := json.Unmarshal(latest.Data, &original); err != nil {
		return none, runtime.Message{}, snap, apiError(runtime.CodeInternal, "read the selected message")
	}
	plan := editPlan{from: latest.Seq, through: tail}
	if previousEnd != nil {
		if aff := forkAffordance(*previousEnd, s.forkSessionsFor(id, snap.LastSessionID)); !aff.Available {
			return none, runtime.Message{}, snap, editUnavailable(aff.Reason)
		}
		if !snap.RuntimeCapabilities.CompleteTurnFork {
			return none, runtime.Message{}, snap, editUnavailable(editReasonNoProof)
		}
		plan.fork = &runtime.ForkPlan{InPlace: true, SourceAgentID: id, SourceSessionID: forkSource(previousEnd.ForkPoint, snap.LastSessionID),
			SourceSeq: previousEnd.Seq, ForkPoint: previousEnd.ForkPoint, HistoryPrimer: snap.HistoryPrimer}
	}
	// The kept attachments must still be retained intact, under Send's own
	// bounds, before anything is accepted (TS-03.R90).
	store := attachment.New(s.configStore.Home())
	for _, d := range original.Attachments {
		f, current, err := store.Open(id, d.ID)
		if err != nil {
			return none, runtime.Message{}, snap, editUnavailable(editReasonAttachment)
		}
		f.Close()
		if current.SHA256 != d.SHA256 || current.Size != d.Size {
			return none, runtime.Message{}, snap, editUnavailable(editReasonAttachment)
		}
	}
	msg := runtime.Message{Text: req.Text, Attachments: original.Attachments, RequestID: req.RequestID}
	if len(msg.Attachments) > 0 {
		if encoded, err := json.Marshal(msg); err != nil || len(encoded) > 96<<10 {
			return none, runtime.Message{}, snap, apiError(runtime.CodeValidation, "replacement text and attachments exceed the message limit")
		}
	}
	return plan, msg, snap, nil
}

// runEdit prepares the replacement session, commits the revision and hands
// the replacement input to ordinary prompt admission exactly once.
func (s *Server) runEdit(ctx context.Context, ctl *editControl, agent state.Agent, snap state.SessionSnapshot, rec state.EditTransition, plan editPlan, msg runtime.Message) {
	id, requestID := agent.AgentID, rec.RequestID
	wasRunning := s.registry.Owns(id)
	abort := func(reason string) {
		ctl.mu.Lock()
		stopped := ctl.stop
		ctl.mu.Unlock()
		switch {
		case stopped:
			reason = "the agent was stopped before the edit was applied."
		case errors.Is(ctx.Err(), context.Canceled):
			reason = "the edit was cancelled before it was applied."
		}
		s.abortEdit(agent, snap, requestID, wasRunning && !stopped, reason)
	}
	candidateSession, err := s.prepareEditCandidate(ctx, agent, snap, plan)
	if err != nil {
		abort(err.Error())
		return
	}
	if editBeforeDecision != nil {
		editBeforeDecision(ctx, id)
	}
	// The irreversible decision, serialized against Stop and Cancel: from
	// here recovery completes the revision.
	ctl.mu.Lock()
	if ctx.Err() != nil {
		ctl.mu.Unlock()
		abort(ctx.Err().Error())
		return
	}
	revision, err := s.stateStore.DecideEdit(id, requestID, plan.from, plan.through, candidateSession)
	ctl.decided = err == nil
	ctl.mu.Unlock()
	if err != nil {
		abort("record the edit decision: " + err.Error())
		return
	}
	s.log.Info("edit committing", "agent", id, "request_id", requestID, "revision", revision)
	if editAfterDecision != nil {
		editAfterDecision(id)
	}
	if err := s.completeEditRevision(id, requestID); err != nil {
		s.log.Error("complete edit revision", "agent", id, "request_id", requestID, "err", err)
		// The candidate never published: stop it rather than leave it running
		// unseen, then complete the decision as a restart would (TS-02.R69).
		if _, staged := s.registry.StagedSession(id); staged {
			stopCtx, stopCancel := context.WithTimeout(context.Background(), editStartTimeout)
			if err := s.registry.Stop(stopCtx, id); err != nil && !errors.Is(err, runtime.ErrNoHandle) {
				s.log.Error("stop unpublished edit candidate", "agent", id, "err", err)
			}
			stopCancel()
			s.teardownAgentRegistration(id)
		}
		if rec, err := s.stateStore.ReadEdit(id, requestID); err == nil {
			s.recoverEdit(rec)
			if done, err := s.stateStore.ReadEdit(id, requestID); err == nil && done.Phase == state.EditCommitted && done.Reason == "" {
				_ = s.stateStore.SetEditReason(id, requestID, editReasonNotStarted)
			}
		}
		s.touchState(id)
		return
	}
	ctl.mu.Lock()
	stopped := ctl.stop
	ctl.mu.Unlock()
	if stopped {
		// Stop after the decision: the revised conversation stays, and the
		// undispatched replacement goes to explicit recovery.
		if rec, err := s.stateStore.ReadEdit(id, requestID); err == nil {
			s.releaseUndispatchedEdit(rec)
		}
		s.touchState(id)
		return
	}
	s.dispatchEdit(ctx, id, requestID, msg)
}

// prepareEditCandidate stops the current runtime and starts the replacement
// session under the same agent identity: an in-place fork ending at the kept
// turn, or the frozen initial context for the first message (TS-04.R113).
// The candidate is staged: nothing selects it until the edit commits, and it
// returns the candidate's native session for the decision.
func (s *Server) prepareEditCandidate(ctx context.Context, agent state.Agent, snap state.SessionSnapshot, plan editPlan) (string, error) {
	id := agent.AgentID
	if ae := s.acquireAgentStart(agent.Project, id); ae != nil {
		return "", errors.New(ae.Message)
	}
	defer s.releaseAgentStart(agent.Project, id)
	if err := s.registry.Stop(ctx, id); err != nil {
		if !errors.Is(err, runtime.ErrNoHandle) {
			return "", fmt.Errorf("stop the current runtime: %w", err)
		}
		if err := s.reapOrphanRuntime(id); err != nil {
			return "", fmt.Errorf("reap orphan runtime: %w", err)
		}
	}
	s.teardownAgentRegistration(id)
	ae, stopErr := s.startRuntimeClaimed(ctx, false, func() (runtime.LaunchSpec, *runtime.APIError) {
		return s.composeSwitchSpecFrom(ctx, agent, "", snap, nil)
	}, func(spec *runtime.LaunchSpec) *runtime.APIError {
		spec.Fork = plan.fork
		spec.EditStaged = true
		return nil
	})
	if stopErr != nil {
		return "", fmt.Errorf("stop the failed candidate: %w", stopErr)
	}
	if ae != nil {
		return "", errors.New(ae.Message)
	}
	session, ok := s.registry.StagedSession(id)
	if !ok {
		return "", errors.New("the prepared session is no longer running")
	}
	return session, nil
}

// abortEdit stops a staged candidate after a failure, Stop or Cancel before
// the decision, restarting the original session when it was running and
// should be restored. The staged candidate published nothing, so the
// original selection and history need no rewrite.
func (s *Server) abortEdit(agent state.Agent, snap state.SessionSnapshot, requestID string, restart bool, reason string) {
	ctx, cancel := context.WithTimeout(context.Background(), editStartTimeout)
	defer cancel()
	id := agent.AgentID
	s.log.Warn("edit aborted", "agent", id, "request_id", requestID, "reason", clipDetail(reason))
	if s.registry.Owns(id) {
		if err := s.registry.Stop(ctx, id); err != nil && !errors.Is(err, runtime.ErrNoHandle) {
			s.log.Error("stop edit candidate", "agent", id, "err", err)
		}
		s.teardownAgentRegistration(id)
	}
	// A failed candidate start can have written its applied agent identity.
	if err := s.stateStore.RestoreSwitchSource(agent, snap); err != nil {
		s.log.Error("restore edit source", "agent", id, "err", err)
	}
	if restart {
		ae, stopErr := s.startRuntimeClaimed(ctx, false, func() (runtime.LaunchSpec, *runtime.APIError) {
			return s.composeSwitchSpecFrom(ctx, agent, snap.LastSessionID, snap, nil)
		}, nil)
		if ae != nil || stopErr != nil {
			s.log.Error("restart edit source", "agent", id, "api_err", ae, "err", stopErr)
			reason += " The original conversation is kept; Resume the agent to continue."
		}
	}
	if err := s.stateStore.AbortEdit(id, requestID, "Nothing was changed: "+reason); err != nil {
		s.log.Error("abort edit", "agent", id, "request_id", requestID, "err", err)
	}
	s.touchState(id)
}

// completeEditRevision appends the synced revision boundary once (detected by
// request id on recovery) and finishes the committed phase.
func (s *Server) completeEditRevision(id, requestID string) error {
	rec, err := s.stateStore.ReadEdit(id, requestID)
	if err != nil {
		return err
	}
	if rec.Phase == state.EditCommitting {
		present, err := revisionRecorded(s.configStore.Home(), id, requestID)
		if err != nil {
			return err
		}
		if !present {
			data, _ := json.Marshal(runtime.ConversationRevisionData{RequestID: requestID, Revision: rec.Revision,
				ReplacesFromSeq: rec.ReplacesFrom, ReplacesThroughSeq: rec.ReplacesThrough})
			if _, err := s.registry.AppendHistory(id, runtime.CommitRequest{Event: runtime.Event{
				Type: runtime.EvConversationRevision, Data: data, Ts: time.Now().UTC().Format(time.RFC3339),
			}}); err != nil {
				return err
			}
		}
		// With no live candidate (a restart, or a stopped unpublished one)
		// PublishStaged cannot journal the selected session; record it from
		// the snapshot instead, as a switch rollback does. A retry may repeat
		// this record, which only restates the same selection.
		if _, live := s.registry.StagedSession(id); !live && rec.CandidateSession != "" {
			agent, err := s.stateStore.ReadAgent(id)
			if err != nil {
				return err
			}
			snap, err := s.stateStore.ReadSession(id)
			if err != nil {
				return err
			}
			snap.LastSessionID = rec.CandidateSession
			if err := s.appendRestoredSessionMeta(agent, snap); err != nil {
				return fmt.Errorf("record the edited session: %w", err)
			}
		}
		if err := s.stateStore.FinishEdit(id, requestID); err != nil {
			return err
		}
	}
	// The committed candidate becomes visible only now (TS-01.R51); after a
	// restart no candidate runs and the snapshot already selects it.
	if editBeforePublish != nil {
		if err := editBeforePublish(id); err != nil {
			return err
		}
	}
	if err := s.registry.PublishStaged(id); err != nil && !errors.Is(err, runtime.ErrNoHandle) {
		return fmt.Errorf("publish the edited session: %w", err)
	}
	// Search, previews and rollups rebuild from the active history. A failed
	// rebuild withdraws the conversation's search text rather than leaving the
	// superseded text searchable as current (TS-02.R70).
	if err := s.indexer.ReindexAgent(s.configStore.Home(), id); err != nil {
		s.log.Error("reindex edited conversation", "agent", id, "request_id", requestID, "err", err)
		if dropErr := s.indexer.WithdrawSearch(id); dropErr != nil {
			s.log.Error("withdraw edited conversation search", "agent", id, "err", dropErr)
		}
		_ = s.stateStore.SetEditReason(id, requestID, editReasonSearchUnavailable)
	}
	s.touchState(id)
	return nil
}

func revisionRecorded(home, id, requestID string) (bool, error) {
	found := false
	err := transcript.ForEachFile(home, id, transcript.ReadOptions{Raw: true}, func(ev runtime.Event) error {
		if ev.Type != runtime.EvConversationRevision {
			return nil
		}
		var d runtime.ConversationRevisionData
		if json.Unmarshal(ev.Data, &d) == nil && d.RequestID == requestID {
			found = true
		}
		return nil
	})
	return found, err
}

// dispatchEdit hands the replacement to ordinary person-turn admission under
// the edit's request id. Dispatch and the prepared receipt are one write
// before admission, so recovery never resends and never loses it; afterwards
// the ordinary receipt decides delivery (TS-02.R69).
func (s *Server) dispatchEdit(ctx context.Context, id, requestID string, msg runtime.Message) {
	if err := s.stateStore.DispatchEdit(editReceipt(id, requestID, msg), "prepared"); err != nil {
		// The edit stays committed and undispatched; restart recovery releases it.
		s.log.Error("dispatch edit", "agent", id, "request_id", requestID, "err", err)
		return
	}
	held, err := s.registry.SendMessageOrHold(ctx, id, msg)
	if err != nil {
		s.log.Warn("edit replacement not delivered", "agent", id, "request_id", requestID, "err", err)
		if rec, readErr := s.stateStore.AttachmentReceipt(id, requestID); readErr == nil && rec.Delivery == "prepared" {
			_ = s.stateStore.SetAttachmentDelivery(id, requestID, "released")
		}
	}
	s.log.Info("edit dispatched", "agent", id, "request_id", requestID, "held", held)
	s.touchState(id)
}

// handleEditStatus implements GET /api/sessions/{id}/edit-last/{request_id}.
func (s *Server) handleEditStatus(w http.ResponseWriter, r *http.Request) {
	rec, err := s.stateStore.ReadEdit(r.PathValue("id"), r.PathValue("request_id"))
	if errors.Is(err, state.ErrNotFound) {
		writeAPIError(w, apiError(runtime.CodeNotFound, "no such edit"))
		return
	}
	if err != nil {
		writeAPIError(w, apiError(runtime.CodeInternal, "cannot read edit"))
		return
	}
	writeJSON(w, http.StatusOK, s.editResultFor(rec, true))
}

func (s *Server) editResultFor(rec state.EditTransition, withText bool) editResult {
	out := editResult{AgentID: rec.AgentID, RequestID: rec.RequestID, Phase: rec.Phase, Revision: rec.Revision,
		Delivery: "pending", Active: "original", Reason: rec.Reason}
	if rec.Phase == state.EditCommitting || rec.Phase == state.EditCommitted {
		out.Active = "revised"
	}
	if rec.Dispatched {
		out.Delivery = "unknown"
		if receipt, err := s.stateStore.AttachmentReceipt(rec.AgentID, rec.RequestID); err == nil {
			out.Delivery = receiptDelivery(receipt.Delivery)
		}
	}
	if withText {
		var msg runtime.Message
		if json.Unmarshal([]byte(rec.MessageJSON), &msg) == nil {
			out.Text = msg.Text
		}
	}
	return out
}

// recoverEdits examines every open edit before ordinary work resumes after a
// restart (TS-02.R69): preparing aborts to the original selection, committing
// completes its recorded decision, and an undispatched replacement is released
// to the existing explicit recovery path rather than sent automatically.
func (s *Server) recoverEdits() {
	open, err := s.stateStore.OpenEdits()
	if err != nil {
		s.log.Error("read open edits", "err", err)
		return
	}
	for _, rec := range open {
		s.recoverEdit(rec)
	}
}

// recoverEdit settles one open edit with no running worker.
func (s *Server) recoverEdit(rec state.EditTransition) {
	id := rec.AgentID
	switch rec.Phase {
	case state.EditPreparing:
		if snap, err := s.stateStore.ReadSession(id); err == nil && rec.OldSession != "" && snap.LastSessionID != rec.OldSession {
			snap.LastSessionID = rec.OldSession
			if agent, err := s.stateStore.ReadAgent(id); err == nil {
				if err := s.stateStore.RestoreSwitchSource(agent, snap); err != nil {
					s.log.Error("restore interrupted edit source", "agent", id, "err", err)
				}
			}
		}
		if err := s.stateStore.AbortEdit(id, rec.RequestID, "Nothing was changed: Chuck restarted while preparing the edit."); err != nil {
			s.log.Error("abort interrupted edit", "agent", id, "err", err)
		}
	case state.EditCommitting:
		if err := s.completeEditRevision(id, rec.RequestID); err != nil {
			s.log.Error("complete interrupted edit", "agent", id, "err", err)
			_ = s.stateStore.SetEditReason(id, rec.RequestID, editReasonUnrepaired)
			return
		}
		s.releaseUndispatchedEdit(rec)
	case state.EditCommitted:
		s.releaseUndispatchedEdit(rec)
	}
}

func (s *Server) releaseUndispatchedEdit(rec state.EditTransition) {
	var msg runtime.Message
	if err := json.Unmarshal([]byte(rec.MessageJSON), &msg); err != nil {
		s.log.Error("decode undispatched edit", "agent", rec.AgentID, "err", err)
		return
	}
	if err := s.stateStore.DispatchEdit(editReceipt(rec.AgentID, rec.RequestID, msg), "released"); err != nil {
		s.log.Error("release undispatched edit", "agent", rec.AgentID, "err", err)
	}
}

func editReceipt(id, requestID string, msg runtime.Message) state.AttachmentReceipt {
	data, _ := json.Marshal(msg)
	return state.AttachmentReceipt{AgentID: id, RequestID: requestID,
		PayloadHash: messagePayloadHash("prompt", msg), MessageJSON: string(data), Action: "prompt"}
}

// touchState republishes the agent's snapshot so every client sees the
// edit's phase and gate.
func (s *Server) touchState(id string) {
	if _, err := s.stateMgr.Touch(id); err != nil {
		s.log.Debug("state touch failed", "agent", id, "err", err)
	}
}
