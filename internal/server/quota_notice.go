package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
	"github.com/AsaphNoam/Chuck/internal/transcript"
)

const quotaNoticeBatch = 8

// A single lock covers both live-writer and stopped-writer retries. Notice
// appends are rare, and serializing them prevents a receipt repair racing the
// event-sink append for the same conversation.
var quotaNoticeAppendMu sync.Mutex

func (s *Server) confirmQuotaProgress(ev runtime.Event) error {
	if !qualifyingQuotaProgress(ev) || s.stateStore == nil {
		return nil
	}
	q, err := s.stateStore.GetQuota(ev.AgentID)
	if errors.Is(err, state.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if q.State != state.QuotaClaimed || !q.ProviderStarted || q.ContinuedGeneration != ev.Generation || q.ContinuedTurn != ev.TurnID {
		return nil
	}
	if q.NoticeConfirmed {
		return nil
	}
	return s.commitQuotaNotice(q)
}

func qualifyingQuotaProgress(ev runtime.Event) bool {
	if ev.AgentID == "" || ev.Generation == "" || ev.TurnID == "" || ev.ActivityID != "" {
		return false
	}
	switch ev.Type {
	case runtime.EvAssistantText:
		var data runtime.AssistantTextData
		return json.Unmarshal(ev.Data, &data) == nil && data.Delta != ""
	case runtime.EvToolCall:
		var data runtime.ToolCallData
		return json.Unmarshal(ev.Data, &data) == nil && data.ToolCallID != "" && data.Name != "" && data.Status == "in_progress"
	case runtime.EvToolResult:
		var data runtime.ToolResultData
		return json.Unmarshal(ev.Data, &data) == nil && data.ToolCallID != "" && (data.Status == "completed" || data.Status == "failed")
	case runtime.EvTurnEnd:
		var data runtime.TurnEndData
		return json.Unmarshal(ev.Data, &data) == nil && data.StopReason == "end_turn" && data.Quota == nil
	default:
		return false
	}
}

func quotaNoticeKey(q state.QuotaIntent) string {
	sum := sha256.Sum256([]byte(q.AgentID + "\x00" + q.EpisodeID + "\x00" + q.AttemptToken))
	return "quota-notice-" + hex.EncodeToString(sum[:])
}

func (s *Server) commitQuotaNotice(q state.QuotaIntent) error {
	quotaNoticeAppendMu.Lock()
	defer quotaNoticeAppendMu.Unlock()
	ev, _, created, err := s.ensureQuotaNoticeLocked(q)
	if err == nil && created && ev.AgentID != "" && s.eventBus != nil {
		s.eventBus.PublishRuntimeEvent(ev)
	}
	return err
}

func (s *Server) ensureQuotaNoticeLocked(q state.QuotaIntent) (runtime.Event, state.QuotaIntent, bool, error) {
	current, err := s.stateStore.GetQuota(q.AgentID)
	if err != nil {
		return runtime.Event{}, state.QuotaIntent{}, false, err
	}
	if !quotaNoticeClaimMatches(current, q.ContinuedGeneration, q.ContinuedTurn) {
		return runtime.Event{}, current, false, nil
	}
	if current.NoticeConfirmed {
		return runtime.Event{}, current, false, nil
	}
	key := quotaNoticeKey(current)
	if current.NoticeKey != "" && current.NoticeKey != key {
		return runtime.Event{}, current, false, state.ErrQuotaConflict
	}
	if current.NoticeKey == "" {
		if err := s.stateStore.MarkQuotaNotice(current.AgentID, current.EpisodeID, current.Revision, current.AttemptToken, key); err != nil {
			return runtime.Event{}, current, false, err
		}
		current, err = s.stateStore.GetQuota(current.AgentID)
		if err != nil {
			return runtime.Event{}, current, false, err
		}
	}
	data := runtime.NoticeData{
		Severity: "info", Title: runtime.QuotaContinuationNoticeTitle,
		Origin: "chuck", ContinuationAttempt: key,
	}
	ev, found, err := findQuotaNotice(s.configStore.Home(), current.AgentID, key)
	if err != nil {
		return runtime.Event{}, current, false, err
	}
	created := false
	if !found {
		if s.registry != nil {
			ev, err = s.registry.AppendQuotaNoticeAndSync(current.AgentID, data)
		} else {
			err = runtime.ErrNoHandle
		}
		if err != nil {
			if s.registry != nil && s.registry.Owns(current.AgentID) {
				return runtime.Event{}, current, false, err // never race a failed active writer
			}
			ev, err = s.appendStoppedQuotaNotice(current, data)
			if err != nil {
				return runtime.Event{}, current, false, err
			}
		}
		created = true
	}
	current, err = s.stateStore.GetQuota(current.AgentID)
	if err != nil {
		return runtime.Event{}, current, false, err
	}
	if !current.NoticeConfirmed {
		if err := s.stateStore.MarkQuotaNoticeConfirmed(current.AgentID, current.EpisodeID, current.Revision, current.AttemptToken, key); err != nil {
			return runtime.Event{}, current, false, err
		}
		current, err = s.stateStore.GetQuota(current.AgentID)
	}
	return ev, current, created, err
}

func (s *Server) confirmQuotaSuccess(agentID, generation, turn string) error {
	q, err := s.stateStore.GetQuota(agentID)
	if err != nil || !quotaNoticeClaimMatches(q, generation, turn) {
		return err
	}
	if !q.SuccessConfirmed {
		if err := s.stateStore.ConfirmQuotaSuccess(agentID, q.EpisodeID, q.Revision, q.AttemptToken, generation, turn); err != nil {
			return err
		}
		q, err = s.stateStore.GetQuota(agentID)
		if err != nil {
			return err
		}
	}
	quotaNoticeAppendMu.Lock()
	defer quotaNoticeAppendMu.Unlock()
	ev, q, created, err := s.ensureQuotaNoticeLocked(q)
	if err != nil {
		return err
	}
	if q.NoticeKey == "" || !q.NoticeConfirmed {
		return state.ErrQuotaConflict
	}
	if err := s.stateStore.CompleteQuota(agentID, q.EpisodeID, q.Revision, q.AttemptToken, generation, turn); err != nil {
		latest, readErr := s.stateStore.GetQuota(agentID)
		if readErr != nil || latest.State != state.QuotaContinued || latest.NoticeKey == "" {
			return err
		}
	}
	if created && ev.AgentID != "" && s.eventBus != nil {
		s.eventBus.PublishRuntimeEvent(ev)
	}
	return nil
}

func quotaNoticeClaimMatches(q state.QuotaIntent, generation, turn string) bool {
	return q.State == state.QuotaClaimed && q.ProviderStarted && q.ContinuedGeneration == generation && q.ContinuedTurn == turn
}

func findQuotaNotice(home, agentID, key string) (runtime.Event, bool, error) {
	var found runtime.Event
	matched := false
	err := transcript.ForEachFile(home, agentID, transcript.ReadOptions{}, func(ev runtime.Event) error {
		if ev.Type != runtime.EvNotice {
			return nil
		}
		var notice runtime.NoticeData
		if json.Unmarshal(ev.Data, &notice) == nil && notice.Origin == "chuck" && notice.ContinuationAttempt == key {
			found, matched = ev, true
			return errQuotaNoticeFound
		}
		return nil
	})
	if errors.Is(err, errQuotaNoticeFound) {
		err = nil
	}
	return found, matched, err
}

var errQuotaNoticeFound = errors.New("quota notice found")

func (s *Server) appendStoppedQuotaNotice(q state.QuotaIntent, data runtime.NoticeData) (runtime.Event, error) {
	if s.registry != nil && s.registry.Generation(q.AgentID) != "" {
		return runtime.Event{}, fmt.Errorf("quota notice writer is still active")
	}
	if !s.claimLifecycle(q.AgentID) {
		return runtime.Event{}, fmt.Errorf("quota notice lifecycle is busy")
	}
	defer s.releaseLifecycle(q.AgentID)
	if s.registry != nil && s.registry.Generation(q.AgentID) != "" {
		return runtime.Event{}, fmt.Errorf("quota notice writer became active")
	}
	if _, err := s.stateStore.ReadRunning(q.AgentID); err == nil {
		return runtime.Event{}, fmt.Errorf("quota notice agent is marked running without an owned writer")
	} else if !errors.Is(err, state.ErrNotFound) {
		return runtime.Event{}, err
	}
	// Recheck while holding the lifecycle claim in case the earlier check raced
	// the tail of a provider append.
	if ev, found, err := findQuotaNotice(s.configStore.Home(), q.AgentID, data.ContinuationAttempt); err != nil {
		return runtime.Event{}, err
	} else if found {
		return ev, nil
	}
	w, err := transcript.Open(s.configStore.Home(), q.AgentID, nil)
	if err != nil {
		return runtime.Event{}, err
	}
	defer w.Close()
	raw, err := json.Marshal(data)
	if err != nil {
		return runtime.Event{}, err
	}
	ev := runtime.Event{
		AgentID: q.AgentID, Generation: q.ContinuedGeneration, TurnID: q.ContinuedTurn,
		Seq: w.NextSeq(), Type: runtime.EvNotice, Data: raw, Ts: time.Now().UTC().Format(time.RFC3339),
	}
	if err := w.Append(ev); err != nil {
		return runtime.Event{}, err
	}
	if err := w.Sync(); err != nil {
		return runtime.Event{}, err
	}
	return ev, nil
}

// repairQuotaNotices drains only receipt-backed claimed attempts. Provider
// prompts are never retried here; an existing transcript event makes the repair
// an idempotent CompleteQuota transition.
func (s *Server) repairQuotaNotices(ctx context.Context) error {
	items, err := s.stateStore.PendingQuotaNotices(quotaNoticeBatch)
	if err != nil {
		return err
	}
	for _, q := range items {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err := s.commitQuotaNotice(q); err != nil {
			s.log.Warn("repair quota continuation notice", "agent", q.AgentID, "err", err)
			continue
		}
		latest, err := s.stateStore.GetQuota(q.AgentID)
		if err != nil || latest.State != state.QuotaClaimed {
			continue
		}
		if latest.SuccessConfirmed {
			if err := s.confirmQuotaSuccess(latest.AgentID, latest.ContinuedGeneration, latest.ContinuedTurn); err != nil {
				s.log.Warn("complete confirmed quota continuation", "agent", q.AgentID, "err", err)
			}
			continue
		}
		if s.registry != nil && s.registry.Owns(latest.AgentID) && s.registry.Generation(latest.AgentID) == latest.ContinuedGeneration {
			continue // this provider turn may still be running
		}
		if err := s.stateStore.HoldQuota(latest.AgentID, latest.EpisodeID, latest.Revision, latest.AttemptToken, "uncertain_delivery"); err != nil {
			s.log.Warn("hold interrupted quota continuation", "agent", q.AgentID, "err", err)
		}
	}
	return nil
}
