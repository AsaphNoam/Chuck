package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/AsaphNoam/Chuck/internal/attachment"
	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
	"github.com/AsaphNoam/Chuck/internal/transcript"
)

const (
	maxAnnotationCount      = 20
	maxAnnotationChars      = 2000
	maxAnnotationMailBytes  = 8000
	annotationClippedMarker = "\n… [excerpt clipped]"
)

// handlePrompt implements POST /api/sessions/{id}/prompt (techspec §7.3).
func (s *Server) handlePrompt(w http.ResponseWriter, r *http.Request) {
	s.handleMessageAction(w, r, "prompt")
}

// promptDelivery names what happened to a submitted prompt so a client renders
// the pending state from the answer rather than inferring it from agent status
// (TS-03.R38).
func promptDelivery(held bool) string {
	if held {
		return "held"
	}
	return "sent"
}

// handleGetPrompt rehydrates the runtime-owned hold after a browser reload.
// Empty text means no message is currently held; the value remains live state
// and never enters the transcript or database (FS-03.R48/R49, TS-08.R56).
func (s *Server) handleGetPrompt(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	msg, afterSeq, heldID, err := s.registry.HeldSnapshot(id)
	if err != nil {
		if !errors.Is(err, runtime.ErrNoHandle) {
			writeAPIError(w, sessionOpError(err))
			return
		}
		if _, e := s.stateStore.ReadAgent(id); e != nil {
			writeAPIError(w, sessionOpError(err))
			return
		}
		if _, e := s.stateStore.AttachmentRecovery(id); errors.Is(e, state.ErrNotFound) {
			writeAPIError(w, sessionOpError(err))
			return
		} else if e != nil {
			writeAPIError(w, apiError(runtime.CodeInternal, "cannot read message recovery"))
			return
		}
	}
	msg.Attachments = append([]attachment.Descriptor{}, msg.Attachments...)
	body := map[string]any{"agent_id": id, "text": msg.Text, "after_seq": afterSeq, "held_id": heldID, "attachments": msg.Attachments, "request_id": msg.RequestID}
	if recovery, e := s.stateStore.AttachmentRecovery(id); e == nil {
		var recovered runtime.Message
		if e = json.Unmarshal([]byte(recovery.MessageJSON), &recovered); e != nil {
			writeAPIError(w, apiError(runtime.CodeInternal, "cannot read recovered message"))
			return
		}
		body["recovery"] = map[string]any{
			"text": recovered.Text, "request_id": recovered.RequestID,
			"attachments": append([]attachment.Descriptor{}, recovered.Attachments...),
			"delivery":    recovery.Delivery,
		}
	} else if !errors.Is(e, state.ErrNotFound) {
		writeAPIError(w, apiError(runtime.CodeInternal, "cannot read message recovery"))
		return
	}
	writeJSON(w, http.StatusOK, body)
}

// handleWithdrawPrompt implements DELETE /api/sessions/{id}/prompt: the one
// operation the hold adds, spelled as a resource (TS-03.R38). Withdrawing when
// nothing is held returns the same body, so a double withdraw is not an error.
func (s *Server) handleWithdrawPrompt(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if requestID := r.URL.Query().Get("recovery_request_id"); requestID != "" {
		if err := s.stateStore.DismissAttachmentRecovery(id, requestID); err != nil {
			writeAPIError(w, apiError(runtime.CodeInternal, "cannot dismiss recovered message"))
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"accepted": true, "agent_id": id, "delivery": "dismissed"})
		return
	}
	removed, withdrawn, err := s.registry.WithdrawHeldExpected(id, r.URL.Query().Get("held_id"))
	if err != nil {
		if errors.Is(err, runtime.ErrHeldChanged) {
			writeAPIError(w, apiError(runtime.CodeHeldChanged, "held message changed"))
			return
		}
		writeAPIError(w, sessionOpError(err))
		return
	}
	body := map[string]any{"accepted": true, "agent_id": id, "delivery": "withdrawn", "withdrawn": withdrawn}
	if withdrawn {
		body["message"] = map[string]any{"held_id": removed.HeldID, "text": removed.Text, "attachments": removed.Attachments, "request_id": removed.RequestID}
	}
	writeJSON(w, http.StatusOK, body)
}

// handleSteer implements POST /api/sessions/{id}/steer (TS-03.R39, FS-03.R50).
// An empty text steers the held follow-up, which is the only way to send one.
func (s *Server) handleSteer(w http.ResponseWriter, r *http.Request) {
	s.handleMessageAction(w, r, "steer")
}

// steerError maps a steer failure to an APIError. An adapter that does not
// advertise steering and an empty steer with nothing held are both conflicts
// against the live session's state; an adapter refusal carries its own reason
// upstream rather than being downgraded to a queue (TS-03.R39, INV §8).
func steerError(err error) *runtime.APIError {
	switch {
	case errors.Is(err, runtime.ErrNoHandle):
		return apiError(runtime.CodeNotFound, "agent not started")
	case errors.Is(err, runtime.ErrSteeringUnsupported):
		return apiError(runtime.CodeConflict, "this agent's adapter does not support steering")
	case errors.Is(err, runtime.ErrNothingHeld):
		return apiError(runtime.CodeConflict, "no message is held for this agent")
	case errors.Is(err, runtime.ErrHeldChanged):
		return apiError(runtime.CodeHeldChanged, "held message changed; reload before steering")
	case errors.Is(err, runtime.ErrHistoryUnavailable), errors.Is(err, runtime.ErrTurnInFlight):
		// The same recoverable refusals Send reports: nothing reached the agent.
		return sessionOpError(err)
	default:
		return apiError(runtime.CodeInternal, err.Error())
	}
}

// handleValidateAnnotations checks a new-task batch before the UI launches its
// recipient, using the same content and mail-size checks as delivery.
func (s *Server) handleValidateAnnotations(w http.ResponseWriter, r *http.Request) {
	var body runtime.AnnotationData
	r.Body = http.MaxBytesReader(w, r.Body, remoteBodyLimit)
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAPIError(w, apiError(runtime.CodeValidation, "invalid JSON body"))
		return
	}
	if err := validateAnnotations(&body); err != nil {
		writeAPIError(w, apiError(runtime.CodeValidation, err.Error()))
		return
	}
	if _, err := annotationMailBody(body); err != nil {
		writeAPIError(w, apiError(runtime.CodeValidation, err.Error()))
		return
	}
	source, err := s.stateStore.ReadAgent(r.PathValue("id"))
	if err != nil {
		if errors.Is(err, state.ErrNotFound) {
			writeAPIError(w, apiError(runtime.CodeNotFound, "no such agent: "+r.PathValue("id")))
		} else {
			writeAPIError(w, apiError(runtime.CodeInternal, "cannot read annotation source"))
		}
		return
	}
	if source.Interface != "chat" {
		writeAPIError(w, apiError(runtime.CodeValidation, "terminal agents cannot be annotation sources"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"valid": true})
}

// handleAnnotations accepts one point-in-time review batch. New-task delivery
// composes the existing launch endpoint with this same agent target, keeping one
// delivery path for mail and one event shape for replay/search.
func (s *Server) handleAnnotations(w http.ResponseWriter, r *http.Request) {
	sourceID := r.PathValue("id")
	var body runtime.AnnotationData
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&body); err != nil {
		writeAPIError(w, apiError(runtime.CodeValidation, "invalid JSON body"))
		return
	}
	if err := validateAnnotations(&body); err != nil {
		writeAPIError(w, apiError(runtime.CodeValidation, err.Error()))
		return
	}
	source, err := s.stateStore.ReadAgent(sourceID)
	if err != nil {
		writeAPIError(w, apiError(runtime.CodeNotFound, "no such agent: "+sourceID))
		return
	}
	if source.Interface != "chat" {
		writeAPIError(w, apiError(runtime.CodeValidation, "terminal agents cannot be annotation sources"))
		return
	}
	commandID := strings.TrimSpace(body.CommandID)
	if commandID != "" && !validRequestID(commandID) {
		writeAPIError(w, apiError(runtime.CodeValidation, "invalid annotation command id"))
		return
	}
	body.CommandID = commandID
	targetID := body.Target.AgentID
	if body.Target.Kind == "self" {
		targetID = sourceID
	}
	var receipt state.AnnotationReceipt
	if commandID != "" {
		if !s.claimAttachmentUpload(sourceID) {
			writeAPIError(w, apiError(runtime.CodeConflict, "another annotation is being prepared; retry shortly"))
			return
		}
		defer s.releaseAttachmentUpload(sourceID)
		body.Target.AgentID = targetID
		hash := annotationPayloadHash(sourceID, body)
		receipt, err = s.stateStore.AnnotationReceipt(sourceID, commandID)
		if err == nil {
			if receipt.PayloadHash != hash || receipt.TargetKind != body.Target.Kind || receipt.TargetAgentID != targetID {
				writeAPIError(w, apiError(runtime.CodeConflict, "command id already belongs to different annotation content"))
				return
			}
			if receipt.State == "unknown" {
				writeAPIError(w, apiError(runtime.CodeConflict, "annotation delivery is uncertain; send a new command deliberately"))
				return
			}
			if receipt.State == "recorded" && receipt.SourceSeq > 0 && body.Target.Kind == "self" {
				writeAPIError(w, apiError(runtime.CodeConflict, "annotation delivery is uncertain; send a new command deliberately"))
				return
			}
		} else if !errors.Is(err, state.ErrNotFound) {
			writeAPIError(w, apiError(runtime.CodeInternal, "cannot read annotation receipt"))
			return
		} else {
			receipt = state.AnnotationReceipt{SourceAgentID: sourceID, CommandID: commandID, PayloadHash: hash,
				TargetKind: body.Target.Kind, TargetAgentID: targetID, State: "prepared"}
			if err := s.stateStore.CreateAnnotationReceipt(receipt); err != nil {
				receipt, err = s.stateStore.AnnotationReceipt(sourceID, commandID)
				if err != nil || receipt.PayloadHash != hash || receipt.TargetAgentID != targetID {
					writeAPIError(w, apiError(runtime.CodeInternal, "cannot retain annotation receipt"))
					return
				}
			}
		}
		if receipt.State == "accepted" && receipt.SourceSeq > 0 {
			writeJSON(w, http.StatusAccepted, map[string]any{"accepted": true, "seq": receipt.SourceSeq, "command_id": commandID, "effect_id": receipt.EffectID})
			return
		}
	}

	// Every target is validated and its delivery payload composed before anything
	// is released, so the durable source event can be written first (invariant
	// §15). Delivery is deferred into this closure rather than run inline: the
	// tray is deliberately preserved on failure (FS-13.R3), so a 500 returned
	// after mail was inserted or a turn was started makes the natural retry
	// duplicate that irreversible effect — and FS-13.R5's durable record would be
	// missing for work an agent is already acting on.
	var deliver func() *runtime.APIError
	switch body.Target.Kind {
	case "self":
		if _, err := s.stateStore.ReadRunning(sourceID); err != nil {
			writeAPIError(w, apiError(runtime.CodeConflict, "the source agent is not running"))
			return
		}
		status, err := s.stateStore.ReadStatus(sourceID)
		if err != nil || status.State != "idle" {
			writeAPIError(w, apiError(runtime.CodeConflict, "the source agent must be idle"))
			return
		}
		block := runtime.FormatAnnotationBlock(body)
		deliver = func() *runtime.APIError {
			if err := s.registry.SendPrompt(r.Context(), sourceID, block); err != nil {
				return sessionOpError(err)
			}
			return nil
		}
	case "agent":
		targetID = body.Target.AgentID
		if targetID == sourceID {
			writeAPIError(w, apiError(runtime.CodeValidation, "use target self for the source agent"))
			return
		}
		target, err := s.stateStore.ReadAgent(targetID)
		if err != nil || target.Interface != "chat" {
			writeAPIError(w, apiError(runtime.CodeValidation, "target must be a running chat agent"))
			return
		}
		if _, err := s.stateStore.ReadRunning(targetID); err != nil {
			writeAPIError(w, apiError(runtime.CodeConflict, "target agent is not running"))
			return
		}
		mailBody, err := annotationMailBody(body)
		if err != nil {
			writeAPIError(w, apiError(runtime.CodeValidation, err.Error()))
			return
		}
		deliver = func() *runtime.APIError {
			if commandID != "" {
				return nil
			}
			if _, err := s.stateStore.InsertMessage(state.Message{
				FromAgent: "user", FromAddress: "user@dashboard", FromName: "Dashboard user",
				ToAgent: targetID, Subject: "Annotations", Body: mailBody,
			}); err != nil {
				return apiError(runtime.CodeInternal, err.Error())
			}
			select {
			case s.activationCh <- targetID:
			default:
			}
			if _, err := s.stateMgr.Touch(targetID); err != nil {
				s.log.Debug("touch annotation recipient failed", "agent", targetID, "err", err)
			}
			return nil
		}
	default:
		writeAPIError(w, apiError(runtime.CodeValidation, "target kind must be self or agent"))
		return
	}
	body.Target.AgentID = targetID
	var ev runtime.Event
	if commandID != "" {
		if receipt.SourceSeq > 0 {
			ev.Seq = receipt.SourceSeq
		} else {
			// A canonical append may have completed while the receipt update or
			// HTTP response was lost. Reconcile its embedded command id before
			// attempting another append.
			var found bool
			readErr := transcript.ForEachFile(s.configStore.Home(), sourceID, transcript.ReadOptions{IncludeMeta: true}, func(candidate runtime.Event) error {
				if candidate.Type != runtime.EvAnnotation {
					return nil
				}
				var saved runtime.AnnotationData
				if json.Unmarshal(candidate.Data, &saved) == nil && saved.CommandID == commandID {
					ev, found = candidate, true
				}
				return nil
			})
			if readErr != nil && !errors.Is(readErr, os.ErrNotExist) {
				writeAPIError(w, apiError(runtime.CodeInternal, "cannot reconcile annotation history"))
				return
			}
			if !found {
				var appendErr error
				ev, appendErr = s.appendAnnotation(sourceID, body)
				if appendErr != nil {
					writeAPIError(w, apiError(runtime.CodeInternal, appendErr.Error()))
					return
				}
			}
			if err := s.stateStore.SetAnnotationSourceSeq(sourceID, commandID, ev.Seq); err != nil {
				writeAPIError(w, apiError(runtime.CodeInternal, "annotation recorded; receipt update failed"))
				return
			}
		}
	} else {
		var appendErr error
		ev, appendErr = s.appendAnnotation(sourceID, body)
		if appendErr != nil {
			writeAPIError(w, apiError(runtime.CodeInternal, appendErr.Error()))
			return
		}
	}
	if commandID != "" && body.Target.Kind == "self" {
		if err := s.stateStore.ClaimAnnotationEffect(sourceID, commandID); err != nil {
			writeAPIError(w, apiError(runtime.CodeInternal, "annotation recorded; delivery claim failed"))
			return
		}
	}
	if apiErr := deliver(); apiErr != nil {
		if commandID != "" && body.Target.Kind == "self" {
			_ = s.stateStore.SetAnnotationEffect(sourceID, commandID, "", "unknown")
		}
		writeAPIError(w, apiErr)
		return
	}
	if commandID != "" && body.Target.Kind == "agent" {
		mailBody, _ := annotationMailBody(body)
		receipt, err = s.stateStore.CommitAnnotationMail(sourceID, commandID, state.Message{
			FromAgent: "user", FromAddress: "user@dashboard", FromName: "Dashboard user",
			ToAgent: targetID, Subject: "Annotations", Body: mailBody,
		})
		if err != nil {
			writeAPIError(w, apiError(runtime.CodeInternal, "annotation recorded; mail delivery remains retryable"))
			return
		}
		select {
		case s.activationCh <- targetID:
		default:
		}
		if _, err := s.stateMgr.Touch(targetID); err != nil {
			s.log.Debug("touch annotation recipient failed", "agent", targetID, "err", err)
		}
	} else if commandID != "" {
		if err := s.stateStore.SetAnnotationEffect(sourceID, commandID, fmt.Sprintf("prompt:%d", ev.Seq), "accepted"); err != nil {
			writeAPIError(w, apiError(runtime.CodeInternal, "annotation delivered; receipt update failed"))
			return
		}
	}
	response := map[string]any{"accepted": true, "seq": ev.Seq}
	if commandID != "" {
		response["command_id"] = commandID
	}
	writeJSON(w, http.StatusAccepted, response)
}

func annotationPayloadHash(sourceID string, data runtime.AnnotationData) string {
	raw, _ := json.Marshal(struct {
		Source string                 `json:"source"`
		Data   runtime.AnnotationData `json:"data"`
	}{sourceID, data})
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func validateAnnotations(data *runtime.AnnotationData) error {
	if len(data.Annotations) == 0 || len(data.Annotations) > maxAnnotationCount {
		return fmt.Errorf("annotations must contain 1..%d entries", maxAnnotationCount)
	}
	if runeCount(data.OverallInstruction) > maxAnnotationChars {
		return fmt.Errorf("overall instruction must be at most %d characters", maxAnnotationChars)
	}
	for i := range data.Annotations {
		a := &data.Annotations[i]
		if strings.TrimSpace(a.Excerpt) == "" || strings.TrimSpace(a.Instruction) == "" {
			return fmt.Errorf("annotation %d needs an anchor, excerpt, and instruction", i+1)
		}
		if runeCount(a.Instruction) > maxAnnotationChars {
			return fmt.Errorf("annotation %d instruction must be at most %d characters", i+1, maxAnnotationChars)
		}
		if a.AnchorKind == "file" {
			if a.Seq != 0 || strings.TrimSpace(a.Path) == "" || a.Side != "" ||
				((a.StartLine == 0) != (a.EndLine == 0)) || a.StartLine < 0 || a.EndLine < a.StartLine {
				return fmt.Errorf("annotation %d has an invalid file anchor", i+1)
			}
		} else if a.AnchorKind != "" || a.Seq < 1 || ((a.StartLine > 0 || a.EndLine > 0 || a.Side != "") &&
			(a.Path == "" || (a.Side != "old" && a.Side != "new") || a.StartLine < 1 || a.EndLine < a.StartLine)) {
			return fmt.Errorf("annotation %d has an invalid transcript anchor", i+1)
		}
		a.Excerpt = clipAnnotationExcerpt(a.Excerpt, maxAnnotationChars)
		a.Instruction = strings.TrimSpace(a.Instruction)
	}
	data.OverallInstruction = strings.TrimSpace(data.OverallInstruction)
	return nil
}

func runeCount(s string) int { return len([]rune(s)) }

func clipAnnotationExcerpt(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	marker := []rune(annotationClippedMarker)
	if max <= len(marker) {
		return string(marker[:max])
	}
	return string(r[:max-len(marker)]) + annotationClippedMarker
}

func annotationMailBody(data runtime.AnnotationData) (string, error) {
	if len(runtime.FormatAnnotationBlock(data)) <= maxAnnotationMailBytes {
		return runtime.FormatAnnotationBlock(data), nil
	}
	trimmed := data
	trimmed.Annotations = append([]runtime.Annotation{}, data.Annotations...)
	for i := range trimmed.Annotations {
		trimmed.Annotations[i].Excerpt = ""
	}
	if len(runtime.FormatAnnotationBlock(trimmed)) > maxAnnotationMailBytes {
		return "", fmt.Errorf("annotation anchors and instructions exceed the %d-byte mail limit", maxAnnotationMailBytes)
	}
	for i := range trimmed.Annotations {
		original := data.Annotations[i].Excerpt
		lo, hi, best := 0, len([]rune(original)), ""
		for lo <= hi {
			mid := (lo + hi) / 2
			candidate := clipAnnotationExcerpt(original, mid)
			trimmed.Annotations[i].Excerpt = candidate
			if len(runtime.FormatAnnotationBlock(trimmed)) <= maxAnnotationMailBytes {
				best, lo = candidate, mid+1
			} else {
				hi = mid - 1
			}
		}
		if best == "" && original != "" {
			best = clipAnnotationExcerpt(original, 1)
		}
		trimmed.Annotations[i].Excerpt = best
	}
	body := runtime.FormatAnnotationBlock(trimmed)
	if len(body) > maxAnnotationMailBytes {
		return "", fmt.Errorf("annotation batch exceeds the %d-byte mail limit", maxAnnotationMailBytes)
	}
	return body, nil
}

// appendAnnotation validates a batch and commits it to the source's history
// through its one conversation journal — the live generation's when one is
// running, otherwise the inactive path — before any delivery happens. A failed
// append returns an error first; the journal publishes only a committed record
// (FS-13.R5, TS-01.R39, TS-03.R14, INV §15).
func (s *Server) appendAnnotation(sourceID string, data runtime.AnnotationData) (runtime.Event, error) {
	if err := validateAnnotations(&data); err != nil {
		return runtime.Event{}, err
	}
	// Verify the mail body will fit before persisting.
	if _, err := annotationMailBody(data); err != nil {
		return runtime.Event{}, err
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return runtime.Event{}, err
	}
	return s.registry.AppendHistory(sourceID, runtime.CommitRequest{Event: runtime.Event{Type: runtime.EvAnnotation, Data: raw}})
}

func (s *Server) handleTranscript(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := s.stateStore.ReadAgent(id); err != nil {
		writeAPIError(w, apiError(runtime.CodeNotFound, "no such agent: "+id))
		return
	}
	sinceSeq, err := parseInt64Query(r, "since_seq")
	if err != nil {
		writeAPIError(w, apiError(runtime.CodeValidation, "since_seq must be an integer"))
		return
	}
	limit, err := parseInt64Query(r, "limit")
	if err != nil || limit < 0 {
		writeAPIError(w, apiError(runtime.CodeValidation, "limit must be a positive integer"))
		return
	}
	beforeSeq, err := parseInt64Query(r, "before_seq")
	if err != nil || beforeSeq < 0 {
		writeAPIError(w, apiError(runtime.CodeValidation, "before_seq must be a positive integer"))
		return
	}
	includeMeta := r.URL.Query().Get("include_meta") == "true"
	// A tailnet read is always windowed, even when the phone names no limit
	// (FS-20.R13); loopback callers opt in with limit or before_seq.
	if limit > 0 || beforeSeq > 0 || remoteFrom(r.Context()) != nil {
		tw := &transcriptWindow{sinceSeq: sinceSeq, beforeSeq: beforeSeq, limit: transcriptWindowDefault}
		if limit > 0 {
			tw.limit = int(min(limit, transcriptWindowMax))
		}
		if chat := s.registry.Chat(); chat != nil {
			tw.hostPending = func(toolCallID string) bool { return chat.PermissionPending(id, toolCallID) }
		}
		if err := transcript.ForEachFile(s.configStore.Home(), id, transcript.ReadOptions{IncludeMeta: includeMeta}, tw.visit); err != nil {
			writeAPIError(w, apiError(runtime.CodeInternal, err.Error()))
			return
		}
		tw.unretained = s.unretainedPermissions(id)
		writeJSON(w, http.StatusOK, tw.response(id))
		return
	}
	events, err := transcript.ReadFile(s.configStore.Home(), id, transcript.ReadOptions{
		SinceSeq:    sinceSeq,
		IncludeMeta: includeMeta,
	})
	if err != nil {
		writeAPIError(w, apiError(runtime.CodeInternal, err.Error()))
		return
	}
	body := map[string]any{"agent_id": id, "events": events}
	// A live request whose record did not commit rides beside history, never
	// in it, so the desktop prompt survives a reload (TS-03.R73).
	if unretained := s.unretainedPermissions(id); len(unretained) > 0 {
		body["unretained_permissions"] = unretained
	}
	writeJSON(w, http.StatusOK, body)
}

func (s *Server) handleMessages(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := s.stateStore.ReadAgent(id); err != nil {
		writeAPIError(w, apiError(runtime.CodeNotFound, "no such agent: "+id))
		return
	}
	unreadOnly := r.URL.Query().Get("unread_only") == "true"
	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			writeAPIError(w, apiError(runtime.CodeValidation, "limit must be a positive integer"))
			return
		}
		limit = n
	}
	if limit > 200 {
		limit = 200
	}
	// ListMessages returns the newest N (created_at DESC); the inbox presents
	// newest-first directly, so no reversal is needed. (The prior ASC query +
	// reversal returned the oldest N and dropped new mail past the limit.)
	msgs, err := s.stateStore.ListMessages(id, unreadOnly, limit)
	if err != nil {
		writeAPIError(w, apiError(runtime.CodeInternal, err.Error()))
		return
	}
	unread, err := s.stateStore.UnreadCount(id)
	if err != nil {
		writeAPIError(w, apiError(runtime.CodeInternal, err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"agent_id":     id,
		"unread_count": unread,
		"messages":     msgs,
	})
}

func parseInt64Query(r *http.Request, key string) (int64, error) {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return 0, nil
	}
	return strconv.ParseInt(raw, 10, 64)
}

// handleCancel implements POST /api/sessions/{id}/cancel (techspec §7.4).
func (s *Server) handleCancel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		Target    string `json:"target"`
		EpisodeID string `json:"episode_id"`
		Revision  int64  `json:"revision"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || (body.Target != "turn" && body.Target != "quota_continuation") {
		writeAPIError(w, apiError(runtime.CodeInvalidField, "cancel target must be turn or quota_continuation"))
		return
	}
	if body.Target == "quota_continuation" {
		s.configMu.Lock()
		defer s.configMu.Unlock()
		if _, err := s.stateStore.ReadAgent(id); err != nil {
			writeAPIError(w, apiError(runtime.CodeNotFound, "no such agent: "+id))
			return
		}
		q, err := s.stateStore.GetQuota(id)
		if body.EpisodeID == "" || body.Revision < 1 {
			writeAPIError(w, apiError(runtime.CodeInvalidField, "quota cancellation requires episode_id and revision"))
			return
		}
		if err == nil && q.EpisodeID == body.EpisodeID && q.State == state.QuotaCancelled && body.Revision == q.Revision-1 {
			writeJSON(w, http.StatusAccepted, map[string]any{"cancelled": false})
			return
		}
		if errors.Is(err, state.ErrNotFound) || err == nil && (q.EpisodeID != body.EpisodeID || q.Revision != body.Revision) {
			writeAPIError(w, apiError(runtime.CodeConflict, "quota continuation changed; refresh and try again"))
			return
		}
		if err != nil {
			writeAPIError(w, apiError(runtime.CodeInternal, "quota continuation could not be read"))
			return
		}
		cancelled := false
		if q.State != state.QuotaCancelled && !q.ProviderStarted && (q.State == state.QuotaScheduled || q.State == state.QuotaDeferred || q.State == state.QuotaClaimed) {
			if err := s.stateStore.CancelQuota(id, q.EpisodeID, q.Revision, q.AttemptToken); err != nil {
				writeAPIError(w, apiError(runtime.CodeConflict, "quota continuation changed; refresh and try again"))
				return
			}
			cancelled = true
			_, _ = s.stateMgr.Touch(id)
		}
		writeJSON(w, http.StatusAccepted, map[string]any{"cancelled": cancelled})
		return
	}
	cancelled, err := s.registry.Cancel(r.Context(), id)
	if err != nil {
		if errors.Is(err, runtime.ErrNoHandle) {
			writeAPIError(w, apiError(runtime.CodeNotFound, "no such agent: "+id))
			return
		}
		writeAPIError(w, apiError(runtime.CodeInternal, err.Error()))
		return
	}
	// false = the agent was idle, so the cancel was a no-op (techspec §7.4).
	writeJSON(w, http.StatusAccepted, map[string]any{"cancelled": cancelled})
}

// handleStop implements POST /api/sessions/{id}/stop (techspec §7.5).
func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if ae := s.stopAgent(r.Context(), id); ae != nil {
		writeAPIError(w, ae)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"stopped": true})
}

// stopAgent is the one stop-and-teardown seam (FS-01.R34, TS-01.R16): the Stop
// route and the group-release worker both run exactly this flow, so neither can
// tear down a live transition's registration through its own door (INV §2).
//
// It takes the same exclusive per-agent lifecycle claim every resume and wake
// takes. Registry.Stop cannot distinguish an in-progress resume's nil sentinel
// from "no handle", so an unclaimed stop arriving mid-wake fell into the
// idempotent branch below and tore down the wake's freshly minted hook token, MCP
// session, and hook-settings file while the resume ran on to report success
// (INV §4). Losing the claim returns the same conflict a losing resume returns;
// the caller may retry once the transition settles.
func (s *Server) stopAgent(ctx context.Context, id string) *runtime.APIError {
	release, ok := s.claimQuotaActions(quotaAgentKeys([]string{id}))
	if !ok {
		return apiError(runtime.CodeConflict, "a lifecycle or quota action is already in progress")
	}
	defer release()
	if !s.claimLifecycle(id) {
		return apiError(runtime.CodeConflict, "a resume is already in progress")
	}
	defer s.releaseLifecycle(id)
	return s.stopAgentClaimed(ctx, id)
}

// stopAgentClaimed stops and tears down an agent whose lifecycle claim is already
// held by the caller. Group release reserves all member claims before it stops
// any member, preserving its all-or-nothing contention contract.
func (s *Server) stopAgentClaimed(ctx context.Context, id string) *runtime.APIError {
	snapshot, err := s.stateStore.SnapshotAgentQuota(id)
	if err != nil {
		return apiError(runtime.CodeInternal, "quota owner could not be read")
	}
	if err := s.registry.Stop(ctx, id); err != nil {
		if !errors.Is(err, runtime.ErrNoHandle) {
			return orphanReapAPIError("", err)
		}
		// Not currently running. If the identity still exists, the agent is
		// already stopped — make stop idempotent so a double-click or a
		// lost-response retry reads as success, not a phantom "unknown agent".
		if _, rerr := s.stateStore.ReadAgent(id); rerr != nil {
			return apiError(runtime.CodeNotFound, "no such agent: "+id)
		}
		// Check for orphan runtimes left by a prior crash (invariant §4).
		// An unreaped orphan keeps its row and registration (FS-01.R54).
		if rerr := s.reapOrphanRuntime(id); rerr != nil {
			return orphanReapAPIError("reap orphan agent: ", rerr)
		}
	}
	s.teardownAgentRegistration(id)
	if snapshot != nil {
		if err := s.cancelQuotaSnapshots([]state.QuotaControlSnapshot{*snapshot}); err != nil {
			return apiError(runtime.CodeInternal, "agent stopped but quota schedule could not be invalidated")
		}
	}
	return nil
}

// handlePermission implements POST /api/sessions/{id}/permission (techspec §7.6).
func (s *Server) handlePermission(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		ToolCallID string `json:"tool_call_id"`
		Decision   string `json:"decision"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAPIError(w, apiError(runtime.CodeValidation, "invalid JSON body"))
		return
	}
	if body.Decision != "approve" && body.Decision != "deny" {
		writeAPIError(w, apiError(runtime.CodeValidation, "decision must be approve or deny"))
		return
	}
	if err := s.registry.Permission(r.Context(), id, body.ToolCallID, body.Decision); err != nil {
		writeAPIError(w, permissionError(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"resolved": true, "tool_call_id": body.ToolCallID, "decision": body.Decision,
	})
}

// handleBackgroundTaskStop implements POST /api/sessions/{id}/background-task-stop
// (TS-03.R44). 202 means the runtime accepted the targeted stop; the durable
// task-state event that follows is the terminal truth. The task id is Runtime's
// normalized value; no provider id is accepted from the browser.
func (s *Server) handleBackgroundTaskStop(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		ActivityID string `json:"activity_id"`
		TaskID     string `json:"task_id"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&body); err != nil || body.TaskID == "" || len(body.TaskID) > 512 {
		writeAPIError(w, apiError(runtime.CodeValidation, "task_id is required"))
		return
	}
	err := s.registry.StopBackgroundTask(r.Context(), id, body.TaskID)
	switch {
	case err == nil:
		writeJSON(w, http.StatusAccepted, map[string]any{"accepted": true})
	case errors.Is(err, runtime.ErrNoHandle):
		writeAPIError(w, apiError(runtime.CodeNotFound, "agent not started"))
	case errors.Is(err, runtime.ErrUnknownBackgroundTask):
		writeAPIError(w, apiError(runtime.CodeNotFound, "that background task is not running"))
	case errors.Is(err, runtime.ErrBackgroundTaskControlUnavailable):
		writeAPIError(w, apiError(runtime.CodeBackgroundTaskControlUnavailable, "this agent cannot stop background tasks"))
	default:
		writeAPIError(w, apiError(runtime.CodeConflict, "the agent could not stop that background task; try again"))
	}
}

func (s *Server) handleRename(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAPIError(w, apiError(runtime.CodeValidation, "invalid JSON body"))
		return
	}
	name, ae := normalizeAgentName(body.Name, false)
	if ae != nil {
		writeAPIError(w, ae)
		return
	}
	agent, err := s.stateStore.ReadAgent(id)
	if err != nil {
		writeAPIError(w, apiError(runtime.CodeNotFound, "no such agent: "+id))
		return
	}
	agent.Name = name
	if err := s.stateStore.WriteAgent(agent); err != nil {
		writeAPIError(w, apiError(runtime.CodeInternal, err.Error()))
		return
	}
	if _, err := s.stateMgr.Touch(id); err != nil {
		s.log.Debug("rename state touch failed", "agent", id, "err", err)
	}
	writeJSON(w, http.StatusOK, map[string]any{"agent_id": id, "name": agent.Name})
}

func (s *Server) handleIdentity(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		Name  *string `json:"name"`
		Group *string `json:"group"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAPIError(w, apiError(runtime.CodeValidation, "invalid JSON body"))
		return
	}
	agent, err := s.stateStore.ReadAgent(id)
	if err != nil {
		writeAPIError(w, apiError(runtime.CodeNotFound, "no such agent: "+id))
		return
	}
	if !s.claimLifecycle(id) {
		writeAPIError(w, apiError(runtime.CodeConflict, "a lifecycle transition is already in progress"))
		return
	}
	defer s.releaseLifecycle(id)
	if ae := s.acquireAgentStart(agent.Project, id); ae != nil {
		writeAPIError(w, ae)
		return
	}
	defer s.releaseAgentStart(agent.Project, id)
	agent, err = s.stateStore.ReadAgent(id)
	if err != nil {
		writeAPIError(w, apiError(runtime.CodeNotFound, "no such agent: "+id))
		return
	}
	if body.Name != nil {
		name, ae := normalizeAgentName(*body.Name, false)
		if ae != nil {
			writeAPIError(w, ae)
			return
		}
		agent.Name = name
	}
	if body.Group != nil {
		group, ae := normalizeGroup(*body.Group)
		if ae != nil {
			writeAPIError(w, ae)
			return
		}
		agent.Group = group
	}
	if err := s.stateStore.WriteAgent(agent); err != nil {
		writeAPIError(w, apiError(runtime.CodeInternal, err.Error()))
		return
	}
	if _, err := s.stateMgr.Touch(id); err != nil {
		s.log.Debug("identity state touch failed", "agent", id, "err", err)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"agent_id":  agent.AgentID,
		"group":     agent.Group,
		"name":      agent.Name,
		"role":      agent.Role,
		"project":   agent.Project,
		"backend":   agent.Backend,
		"model":     agent.Model,
		"interface": agent.Interface,
	})
}

// sessionOpError maps a prompt/control error to an APIError (techspec §7.3).
func sessionOpError(err error) *runtime.APIError {
	switch {
	case errors.Is(err, runtime.ErrNoHandle):
		return apiError(runtime.CodeNotFound, "agent not started")
	case errors.Is(err, runtime.ErrTurnInFlight):
		return apiError(runtime.CodeConflict, "a turn is already in flight")
	case errors.Is(err, runtime.ErrHistoryUnavailable):
		// A definite refusal: nothing reached the agent and the draft stays
		// with the person; Stop and approvals remain available (FS-03.R99).
		return apiError(runtime.CodeConflict, "Nothing was sent: this conversation's history can't be saved right now. Your message is kept — try again once storage is available.")
	default:
		return apiError(runtime.CodeInternal, err.Error())
	}
}

// permissionError maps a Permission relay error to an APIError (techspec §7.6).
func permissionError(err error) *runtime.APIError {
	switch {
	case errors.Is(err, runtime.ErrNoHandle):
		return apiError(runtime.CodeNotFound, "agent not started")
	case errors.Is(err, runtime.ErrNoPendingPermission):
		return apiError(runtime.CodeConflict, "no pending permission for that tool_call_id")
	case errors.Is(err, runtime.ErrPermissionAlreadyResolved):
		return apiError(runtime.CodeConflict, "permission already resolved for that tool_call_id")
	case errors.Is(err, runtime.ErrInvalidDecision):
		return apiError(runtime.CodeValidation, "decision must be approve or deny")
	default:
		return apiError(runtime.CodeInternal, err.Error())
	}
}
