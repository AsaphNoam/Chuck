package server

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
	"github.com/AsaphNoam/Chuck/internal/transcript"
)

// FS-01.A27: permission-pending tool intent, metadata and stale/child events
// cannot corroborate automatic continuation; actual root execution can.
func TestQuotaProgressRequiresExecutingRootActivity(t *testing.T) {
	for _, tc := range []struct {
		name, typ, body string
		child           bool
		want            bool
	}{
		{"pending tool", runtime.EvToolCall, `{"tool_call_id":"tool-1","name":"Read","status":"pending"}`, false, false},
		{"executing tool", runtime.EvToolCall, `{"tool_call_id":"tool-1","name":"Read","status":"in_progress"}`, false, true},
		{"root text", runtime.EvAssistantText, `{"delta":"Started"}`, false, true},
		{"empty text", runtime.EvAssistantText, `{"delta":""}`, false, false},
		{"child text", runtime.EvAssistantText, `{"delta":"Started"}`, true, false},
		{"notice", runtime.EvNotice, `{"severity":"info","title":"Retrying"}`, false, false},
	} {
		ev := runtime.Event{AgentID: "agent", Generation: "generation", TurnID: "turn", Type: tc.typ, Data: json.RawMessage(tc.body)}
		if tc.child {
			ev.ActivityID = "child"
		}
		if got := qualifyingQuotaProgress(ev); got != tc.want {
			t.Fatalf("%s progress=%v, want %v", tc.name, got, tc.want)
		}
	}
}

func seedQuotaNoticeClaim(t *testing.T, s *Server, id string, confirmed bool) state.QuotaIntent {
	t.Helper()
	if err := s.indexer.UpsertSessionMeta(id, runtime.SessionMetaData{
		Name: id, Role: "default", Project: "my-app", Backend: "claude", Model: "sonnet", Interface: "chat", CreatedAt: time.Now().UTC().Add(-time.Hour).Format(time.RFC3339),
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.stateStore.WriteAgent(state.Agent{AgentID: id, Name: id, Role: "default", Project: "my-app", Backend: "claude", Model: "sonnet", Interface: "chat", CreatedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	// Claims are admitted through due schedules; due follows the known reset,
	// which has elapsed since the observation.
	observed := now.Add(-time.Minute)
	reset := now.Add(-30 * time.Second)
	due := now.Add(-time.Second)
	q, err := s.stateStore.ObserveQuota(state.QuotaObservation{AgentID: id, EpisodeID: "notice-episode", SourceGeneration: "source-gen", ExecutingTurn: "source-turn", Owner: state.QuotaOwner{Kind: "ordinary"}, ObservedAt: observed, ResetAt: &reset, DueAt: &due, State: state.QuotaScheduled})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.stateStore.ClaimQuota(id, q.EpisodeID, q.Revision, "notice-attempt"); err != nil {
		t.Fatal(err)
	}
	q, err = s.stateStore.GetQuota(id)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.stateStore.MarkQuotaProviderStarted(id, q.EpisodeID, q.Revision, q.AttemptToken, "continued-gen", "continued-turn"); err != nil {
		t.Fatal(err)
	}
	q, err = s.stateStore.GetQuota(id)
	if err != nil {
		t.Fatal(err)
	}
	key := quotaNoticeKey(q)
	if err := s.stateStore.MarkQuotaNotice(id, q.EpisodeID, q.Revision, q.AttemptToken, key); err != nil {
		t.Fatal(err)
	}
	q, err = s.stateStore.GetQuota(id)
	if err != nil {
		t.Fatal(err)
	}
	if confirmed {
		if err := s.stateStore.ConfirmQuotaSuccess(id, q.EpisodeID, q.Revision, q.AttemptToken, q.ContinuedGeneration, q.ContinuedTurn); err != nil {
			t.Fatal(err)
		}
		q, err = s.stateStore.GetQuota(id)
		if err != nil {
			t.Fatal(err)
		}
	}
	w, err := transcript.Open(s.configStore.Home(), id, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return q
}

func TestQuotaNoticeRepairHoldsProgressWithoutSuccessReceipt(t *testing.T) {
	s := testServer(t, true)
	q := seedQuotaNoticeClaim(t, s, "quota-notice-progress", false)
	if err := s.repairQuotaNotices(context.Background()); err != nil {
		t.Fatal(err)
	}
	got, err := s.stateStore.GetQuota(q.AgentID)
	if err != nil || got.State != state.QuotaNeedsAttention || got.ReasonCode != "uncertain_delivery" {
		t.Fatalf("progress-only recovery: %+v err=%v", got, err)
	}
	events, err := transcript.ReadFile(s.configStore.Home(), q.AgentID, transcript.ReadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, ev := range events {
		if ev.Type != runtime.EvNotice {
			continue
		}
		var n runtime.NoticeData
		if json.Unmarshal(ev.Data, &n) == nil && n.ContinuationAttempt == q.NoticeKey {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("repaired continuation notices=%d, want one", count)
	}
	if err := s.repairQuotaNotices(context.Background()); err != nil {
		t.Fatal(err)
	}
	events, _ = transcript.ReadFile(s.configStore.Home(), q.AgentID, transcript.ReadOptions{})
	count = 0
	for _, ev := range events {
		if ev.Type == runtime.EvNotice {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("notice duplicated on retry: %d", count)
	}
}

func TestQuotaNoticeRepairCompletesOnlyConfirmedSuccess(t *testing.T) {
	s := testServer(t, true)
	q := seedQuotaNoticeClaim(t, s, "quota-notice-success", true)
	busEvents, unsubscribe := s.eventBus.Subscribe()
	defer unsubscribe()
	progress, _ := json.Marshal(runtime.AssistantTextData{Delta: "visible reply"})
	ev := runtime.Event{AgentID: q.AgentID, Generation: q.ContinuedGeneration, TurnID: q.ContinuedTurn, Type: runtime.EvAssistantText, Data: progress}
	if err := s.confirmQuotaProgress(ev); err != nil {
		t.Fatal(err)
	}
	if err := s.confirmQuotaProgress(ev); err != nil {
		t.Fatal(err)
	}
	if err := s.repairQuotaNotices(context.Background()); err != nil {
		t.Fatal(err)
	}
	got, err := s.stateStore.GetQuota(q.AgentID)
	if err != nil || got.State != state.QuotaContinued {
		t.Fatalf("confirmed recovery: %+v err=%v", got, err)
	}
	events, err := transcript.ReadFile(s.configStore.Home(), q.AgentID, transcript.ReadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, ev := range events {
		if ev.Type == runtime.EvNotice {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("confirmed recovery notices=%d, want one", count)
	}
	busNotices := 0
	for {
		select {
		case event := <-busEvents:
			if event.Type != "new_message" {
				continue
			}
			if runtimeEvent, ok := event.Data.(runtime.Event); ok && runtimeEvent.Type == runtime.EvNotice {
				busNotices++
			}
		default:
			if busNotices != 1 {
				t.Fatalf("published quota notice events=%d, want exactly one", busNotices)
			}
			return
		}
	}
}

func TestQuotaNoticeRepairIndexesOnceAcrossAppendCrash(t *testing.T) {
	for _, tc := range []struct {
		name            string
		alreadyAppended bool
		alreadyIndexed  bool
		laterIndexed    bool
	}{
		{name: "new notice"},
		{name: "transcript append before index", alreadyAppended: true},
		{name: "old live index before receipt", alreadyAppended: true, alreadyIndexed: true},
		{name: "unindexed notice before later event", alreadyAppended: true, laterIndexed: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := testServer(t, true)
			q := seedQuotaNoticeClaim(t, s, "quota-index-repair", true)
			if tc.alreadyAppended {
				w, err := transcript.Open(s.configStore.Home(), q.AgentID, nil)
				if err != nil {
					t.Fatal(err)
				}
				data, err := json.Marshal(runtime.NoticeData{Severity: "info", Title: runtime.QuotaContinuationNoticeTitle, Origin: "chuck", ContinuationAttempt: q.NoticeKey})
				if err != nil {
					t.Fatal(err)
				}
				ev := runtime.Event{AgentID: q.AgentID, Generation: q.ContinuedGeneration, TurnID: q.ContinuedTurn, Seq: w.NextSeq(), Type: runtime.EvNotice, Data: data, Ts: time.Now().UTC().Format(time.RFC3339)}
				if err := w.Append(ev); err != nil {
					t.Fatal(err)
				}
				if tc.alreadyIndexed {
					if err := s.indexer.OnEvent(q.AgentID, ev); err != nil {
						t.Fatal(err)
					}
				}
				if tc.laterIndexed {
					data, err := json.Marshal(runtime.AssistantTextData{Delta: "later reply"})
					if err != nil {
						t.Fatal(err)
					}
					later := runtime.Event{AgentID: q.AgentID, Generation: q.ContinuedGeneration, TurnID: q.ContinuedTurn, Seq: ev.Seq + 1, Type: runtime.EvAssistantText, Data: data, Ts: time.Now().UTC().Add(time.Second).Format(time.RFC3339)}
					if err := w.Append(later); err != nil {
						t.Fatal(err)
					}
					if err := s.indexer.OnEvent(q.AgentID, later); err != nil {
						t.Fatal(err)
					}
				}
				if err := w.Sync(); err != nil {
					t.Fatal(err)
				}
				if err := w.Close(); err != nil {
					t.Fatal(err)
				}
			}
			if err := s.repairQuotaNotices(context.Background()); err != nil {
				t.Fatal(err)
			}
			events, err := transcript.ReadFile(s.configStore.Home(), q.AgentID, transcript.ReadOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if len(events) < 1 || events[0].Type != runtime.EvNotice {
				t.Fatalf("transcript events=%+v, want notice first", events)
			}
			var count, lastSeq int64
			var updatedAt string
			readIndex := func() {
				t.Helper()
				if err := s.stateStore.DB().QueryRow(`SELECT event_count, last_seq, updated_at FROM sessions WHERE agent_id=?`, q.AgentID).Scan(&count, &lastSeq, &updatedAt); err != nil {
					t.Fatal(err)
				}
			}
			readIndex()
			last := events[len(events)-1]
			if count != int64(len(events)) || lastSeq != last.Seq || updatedAt != last.Ts {
				t.Fatalf("index after repair: count=%d seq=%d updated=%s, want %d/%d/%s", count, lastSeq, updatedAt, len(events), last.Seq, last.Ts)
			}
			if err := s.repairQuotaNotices(context.Background()); err != nil {
				t.Fatal(err)
			}
			readIndex()
			if count != int64(len(events)) || lastSeq != last.Seq || updatedAt != last.Ts {
				t.Fatalf("index changed on retry: count=%d seq=%d updated=%s", count, lastSeq, updatedAt)
			}
		})
	}
}

func TestQuotaNoticeRepairRetriesFailedIndexTransaction(t *testing.T) {
	s := testServer(t, true)
	q := seedQuotaNoticeClaim(t, s, "quota-index-failure", true)
	busEvents, unsubscribe := s.eventBus.Subscribe()
	defer unsubscribe()
	countPublishedNotices := func() int {
		count := 0
		for {
			select {
			case event := <-busEvents:
				if notice, ok := event.Data.(runtime.Event); ok && notice.Type == runtime.EvNotice {
					count++
				}
			default:
				return count
			}
		}
	}
	if _, err := s.stateStore.DB().Exec(`CREATE TRIGGER fail_notice_index BEFORE UPDATE OF event_count ON sessions BEGIN SELECT RAISE(FAIL, 'index unavailable'); END`); err != nil {
		t.Fatal(err)
	}
	if err := s.repairQuotaNotices(context.Background()); err != nil {
		t.Fatal(err)
	}
	got, err := s.stateStore.GetQuota(q.AgentID)
	if err != nil {
		t.Fatal(err)
	}
	if got.NoticeConfirmed {
		t.Fatal("notice receipt acknowledged without session metadata")
	}
	if count := countPublishedNotices(); count != 0 {
		t.Fatalf("notices published before index repair=%d, want zero", count)
	}
	if _, err := s.stateStore.DB().Exec(`DROP TRIGGER fail_notice_index`); err != nil {
		t.Fatal(err)
	}
	if err := s.repairQuotaNotices(context.Background()); err != nil {
		t.Fatal(err)
	}
	got, err = s.stateStore.GetQuota(q.AgentID)
	if err != nil || !got.NoticeConfirmed || got.State != state.QuotaContinued {
		t.Fatalf("recovered notice receipt: %+v err=%v", got, err)
	}
	if count := countPublishedNotices(); count != 1 {
		t.Fatalf("notices published after index repair=%d, want one", count)
	}
	if err := s.repairQuotaNotices(context.Background()); err != nil {
		t.Fatal(err)
	}
	if count := countPublishedNotices(); count != 0 {
		t.Fatalf("duplicate notice publications on repeat repair=%d", count)
	}
	events, err := transcript.ReadFile(s.configStore.Home(), q.AgentID, transcript.ReadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	var count, lastSeq int64
	if err := s.stateStore.DB().QueryRow(`SELECT event_count, last_seq FROM sessions WHERE agent_id=?`, q.AgentID).Scan(&count, &lastSeq); err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || count != 1 || lastSeq != events[0].Seq {
		t.Fatalf("after retry: transcript=%d events, index count=%d seq=%d", len(events), count, lastSeq)
	}
}
