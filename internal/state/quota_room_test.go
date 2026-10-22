package state

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func quotaRoomBegin(t *testing.T, st *Store, room string, d ThinkTankDetail, agent, turn string) (ThinkTankAttempt, string) {
	t.Helper()
	ttTurnSeq++
	a, err := st.BeginThinkTankAttempt(ThinkTankBegin{RoomID: room, Revision: d.Room.Revision, AgentID: agent,
		Turn: turn, Generation: "quota-source-generation", TurnID: fmt.Sprintf("quota-source-%d", ttTurnSeq)})
	if err != nil {
		t.Fatal(err)
	}
	page, err := st.ReadThinkTankPage(ThinkTankReadRequest{CallerAgentID: agent, RoomID: room})
	if err != nil || !page.Complete {
		t.Fatalf("read quota source: complete=%v err=%v", page.Complete, err)
	}
	if _, err := st.StageThinkTankTurn(agent, a.Token, ThinkTankReply, "staged before quota", page.ReadReceipt); err != nil {
		t.Fatal(err)
	}
	return a, page.ReadReceipt
}

func quotaRoomSetup(t *testing.T, st *Store, phase string) (ThinkTankDetail, ThinkTankAttempt, int) {
	t.Helper()
	var d ThinkTankDetail
	switch phase {
	case ThinkTankTurnOpening:
		d = ttCreate(t, st, true, ttMember("a", 4), ttMember("b", 4))
	case ThinkTankTurnDiscussion:
		d = ttCreate(t, st, false, ttMember("a", 4), ttMember("b", 4))
	case ThinkTankTurnClosing:
		d = ttCreate(t, st, false, ttMember("a", 2), ttMember("b", 1))
		for i := 0; i < 2; i++ {
			a, _, receipt := ttTurn(t, st, d.Room.RoomID)
			ttSubmit(t, st, a, receipt, ThinkTankReply, "discussion")
		}
		if got := mustTT(t, st, d.Room.RoomID); got.Room.Phase != ThinkTankPhaseClosing {
			t.Fatalf("phase=%s, want closing", got.Room.Phase)
		}
		d = mustTT(t, st, d.Room.RoomID)
	case ThinkTankTurnJudge:
		d = ttCreate(t, st, false, ttMember("a", 1), ttMember("b", 1))
		if _, err := st.db.Exec(`UPDATE think_tanks SET judge_config='{}', judge_status=? WHERE room_id=?`, ThinkTankJudgeWaiting, d.Room.RoomID); err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 2; i++ {
			a, _, receipt := ttTurn(t, st, d.Room.RoomID)
			ttSubmit(t, st, a, receipt, ThinkTankReply, "discussion")
		}
		if got := mustTT(t, st, d.Room.RoomID); got.Room.JudgeStatus != ThinkTankJudgeReady {
			t.Fatalf("judge status=%s", got.Room.JudgeStatus)
		}
		var err error
		d, err = st.ReserveThinkTankJudge(d.Room.RoomID, "judge", "Judge", "judge-project", "judge-generation")
		if err != nil {
			t.Fatal(err)
		}
		d, err = st.MarkThinkTankJudgeLaunched(d.Room.RoomID, "Judge", "", "judge-generation")
		if err != nil {
			t.Fatal(err)
		}
	default:
		t.Fatalf("unknown phase %q", phase)
	}
	for _, m := range d.Members {
		if err := st.WriteAgent(testAgent(m.AgentID, timeNow())); err != nil {
			t.Fatal(err)
		}
	}
	if phase == ThinkTankTurnJudge {
		if err := st.WriteAgent(testAgent("judge", timeNow())); err != nil {
			t.Fatal(err)
		}
	}
	next, ok := NextThinkTankOpportunity(d)
	if !ok || next.Turn != phase {
		t.Fatalf("next=%+v, room=%+v members=%+v, want turn %s", next, d.Room, d.Members, phase)
	}
	a, _ := quotaRoomBegin(t, st, d.Room.RoomID, d, next.AgentID, phase)
	before := 0
	for _, m := range d.Members {
		if m.AgentID == next.AgentID {
			before = m.Completed
		}
	}
	return d, a, before
}

func continueQuotaRoom(t *testing.T, st *Store, d ThinkTankDetail, source ThinkTankAttempt, completed int) ThinkTankAttempt {
	t.Helper()
	now := timeNow().UTC().Truncate(time.Second)
	observed := now.Add(-2 * time.Minute)
	reset := now.Add(-time.Minute)
	if _, err := st.ObserveOwnedQuota(QuotaObservation{AgentID: source.AgentID, EpisodeID: "quota-episode-" + source.Turn,
		SourceGeneration: source.Generation, ExecutingTurn: source.TurnID, ObservedAt: observed, ResetAt: &reset,
		DueAt: &now, State: QuotaScheduled}); err != nil {
		t.Fatal(err)
	}
	failed, err := scanThinkTankAttempt(st.db.QueryRow(`SELECT `+thinkTankAttemptColumns+` FROM think_tank_attempts WHERE attempt_id=?`, source.AttemptID))
	if err != nil {
		t.Fatal(err)
	}
	if failed.State != ThinkTankAttemptFailed || failed.Failure != "Quota reached" || failed.Message != "staged before quota" || failed.Disposition != ThinkTankReply {
		t.Fatalf("quota source did not remain immutable history: %+v", failed)
	}
	entries, err := st.ListThinkTankEntries(d.Room.RoomID, 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	quota, err := st.GetQuota(source.AgentID)
	if err != nil {
		t.Fatal(err)
	}
	if quota.Owner.Kind != "room" || quota.Owner.RoomID != d.Room.RoomID || quota.Owner.AttemptID != source.AttemptID || quota.Owner.Turn != source.Turn {
		t.Fatalf("quota owner = %+v", quota.Owner)
	}
	if err := st.ClaimQuota(quota.AgentID, quota.EpisodeID, quota.Revision, "room-quota-claim"); err != nil {
		t.Fatal(err)
	}
	quota, err = st.GetQuota(source.AgentID)
	if err != nil {
		t.Fatal(err)
	}
	current := mustTT(t, st, d.Room.RoomID)
	newAttempt, err := st.BeginThinkTankAttempt(ThinkTankBegin{RoomID: d.Room.RoomID, Revision: current.Room.Revision,
		AgentID: source.AgentID, Turn: source.Turn, Generation: "quota-resumed-generation", TurnID: "quota-resumed-turn",
		QuotaContinuation: &ThinkTankQuotaContinuation{EpisodeID: quota.EpisodeID, Revision: quota.Revision,
			AttemptToken: quota.AttemptToken, SourceAttemptID: source.AttemptID}})
	if err != nil {
		t.Fatal(err)
	}
	if newAttempt.AttemptID == source.AttemptID || newAttempt.Token == source.Token || newAttempt.Head != source.Head {
		t.Fatalf("continuation reused old authority or changed read head: old=%+v new=%+v", source, newAttempt)
	}
	claimed, err := st.GetQuota(source.AgentID)
	if err != nil || !claimed.ProviderStarted || claimed.ContinuedGeneration != newAttempt.Generation || claimed.ContinuedTurn != newAttempt.TurnID {
		t.Fatalf("quota provider boundary was not committed with the room attempt: quota=%+v err=%v", claimed, err)
	}
	if len(entries) != len(mustEntries(t, st, d.Room.RoomID)) {
		t.Fatal("quota continuation published staged source text")
	}
	for _, member := range mustTT(t, st, d.Room.RoomID).Members {
		if member.AgentID == source.AgentID && member.Completed != completed {
			t.Fatalf("quota charged member: %+v", member)
		}
	}
	if len(mustTT(t, st, d.Room.RoomID).Running) == 0 {
		t.Fatal("new continuation is not the active room attempt")
	}
	return newAttempt
}

func mustEntries(t *testing.T, st *Store, room string) []ThinkTankEntry {
	t.Helper()
	entries, err := st.ListThinkTankEntries(room, 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	return entries
}

func TestQuotaThinkTankContinuationPhases(t *testing.T) {
	for _, phase := range []string{ThinkTankTurnOpening, ThinkTankTurnDiscussion, ThinkTankTurnClosing, ThinkTankTurnJudge} {
		t.Run(phase, func(t *testing.T) {
			st, _ := newTestStore(t)
			d, source, completed := quotaRoomSetup(t, st, phase)
			newAttempt := continueQuotaRoom(t, st, d, source, completed)
			if newAttempt.Turn != phase {
				t.Fatalf("turn=%s", newAttempt.Turn)
			}
			for _, member := range mustTT(t, st, d.Room.RoomID).Members {
				if member.AgentID == source.AgentID && member.Completed != completed {
					t.Fatalf("quota changed completed count: got %d want %d", member.Completed, completed)
				}
			}
		})
	}
}

func TestQuotaThinkTankContinuationRespectsPauseAndEnd(t *testing.T) {
	for _, action := range []string{"pause", "end"} {
		t.Run(action, func(t *testing.T) {
			st, _ := newTestStore(t)
			d, source, _ := quotaRoomSetup(t, st, ThinkTankTurnDiscussion)
			now := timeNow().UTC().Truncate(time.Second)
			observed := now.Add(-2 * time.Minute)
			reset := now.Add(-time.Minute)
			if _, err := st.ObserveOwnedQuota(QuotaObservation{AgentID: source.AgentID, EpisodeID: "quota-" + action, SourceGeneration: source.Generation,
				ExecutingTurn: source.TurnID, ObservedAt: observed, ResetAt: &reset, DueAt: &now, State: QuotaScheduled}); err != nil {
				t.Fatal(err)
			}
			q, _ := st.GetQuota(source.AgentID)
			if err := st.ClaimQuota(q.AgentID, q.EpisodeID, q.Revision, "pause-end-token"); err != nil {
				t.Fatal(err)
			}
			q, _ = st.GetQuota(source.AgentID)
			if action == "pause" {
				_, err := st.PauseThinkTank(d.Room.RoomID)
				if err != nil {
					t.Fatal(err)
				}
				if err := st.RecoverThinkTanks(); err != nil {
					t.Fatal(err)
				}
				if got := mustTT(t, st, d.Room.RoomID).Room.Control; got != ThinkTankPaused {
					t.Fatalf("startup recovery overrode operator pause: control=%q", got)
				}
			} else {
				_, err := st.EndThinkTank(d.Room.RoomID)
				if err != nil {
					t.Fatal(err)
				}
			}
			current := mustTT(t, st, d.Room.RoomID)
			_, err := st.BeginThinkTankAttempt(ThinkTankBegin{RoomID: d.Room.RoomID, Revision: current.Room.Revision, AgentID: source.AgentID,
				Turn: source.Turn, Generation: "new", TurnID: "new-turn", QuotaContinuation: &ThinkTankQuotaContinuation{EpisodeID: q.EpisodeID,
					Revision: q.Revision, AttemptToken: q.AttemptToken, SourceAttemptID: source.AttemptID}})
			if !errors.Is(err, ErrThinkTankConflict) {
				t.Fatalf("continuation after %s = %v", action, err)
			}
		})
	}
}

func TestQuotaThinkTankOpeningContinuationCanJoinConcurrentOpenings(t *testing.T) {
	st, _ := newTestStore(t)
	d := ttCreate(t, st, true, ttMember("a", 3), ttMember("b", 3))
	for _, id := range []string{"a", "b"} {
		if err := st.WriteAgent(testAgent(id, timeNow())); err != nil {
			t.Fatal(err)
		}
	}
	a, _ := quotaRoomBegin(t, st, d.Room.RoomID, d, "a", ThinkTankTurnOpening)
	b, _ := quotaRoomBegin(t, st, d.Room.RoomID, d, "b", ThinkTankTurnOpening)
	now := timeNow().UTC().Truncate(time.Second)
	observed := now.Add(-2 * time.Minute)
	reset := now.Add(-time.Minute)
	if _, err := st.ObserveOwnedQuota(QuotaObservation{AgentID: "a", EpisodeID: "quota-openings", SourceGeneration: a.Generation, ExecutingTurn: a.TurnID,
		ObservedAt: observed, ResetAt: &reset, DueAt: &now, State: QuotaScheduled}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.ObserveOwnedQuota(QuotaObservation{AgentID: "b", EpisodeID: "quota-openings-b", SourceGeneration: b.Generation, ExecutingTurn: b.TurnID,
		ObservedAt: observed, ResetAt: &reset, DueAt: &now, State: QuotaScheduled}); err != nil {
		t.Fatal(err)
	}
	q, _ := st.GetQuota("a")
	if err := st.ClaimQuota(q.AgentID, q.EpisodeID, q.Revision, "openings-token"); err != nil {
		t.Fatal(err)
	}
	q, _ = st.GetQuota("a")
	current := mustTT(t, st, d.Room.RoomID)
	next, err := st.BeginThinkTankAttempt(ThinkTankBegin{RoomID: d.Room.RoomID, Revision: current.Room.Revision, AgentID: "a", Turn: ThinkTankTurnOpening,
		Generation: "a-new", TurnID: "a-new-turn", QuotaContinuation: &ThinkTankQuotaContinuation{EpisodeID: q.EpisodeID, Revision: q.Revision,
			AttemptToken: q.AttemptToken, SourceAttemptID: a.AttemptID}})
	if err != nil {
		t.Fatal(err)
	}
	qb, err := st.GetQuota("b")
	if err != nil {
		t.Fatal(err)
	}
	if err := st.ClaimQuota(qb.AgentID, qb.EpisodeID, qb.Revision, "openings-token-b"); err != nil {
		t.Fatal(err)
	}
	qb, _ = st.GetQuota("b")
	current = mustTT(t, st, d.Room.RoomID)
	nextB, err := st.BeginThinkTankAttempt(ThinkTankBegin{RoomID: d.Room.RoomID, Revision: current.Room.Revision, AgentID: "b", Turn: ThinkTankTurnOpening,
		Generation: "b-new", TurnID: "b-new-turn", QuotaContinuation: &ThinkTankQuotaContinuation{EpisodeID: qb.EpisodeID, Revision: qb.Revision,
			AttemptToken: qb.AttemptToken, SourceAttemptID: b.AttemptID}})
	if err != nil {
		t.Fatal(err)
	}
	room := mustTT(t, st, d.Room.RoomID)
	if next.AgentID != "a" || nextB.AgentID != "b" || len(room.Running) != 2 || room.Room.Hold != "Quota reached" {
		t.Fatalf("continuation=%+v concurrent=%+v", next, mustTT(t, st, d.Room.RoomID).Running)
	}
}

func TestReleaseQuotaRoomHoldWaitsForEveryRoomEpisode(t *testing.T) {
	st, _ := newTestStore(t)
	d, source, _ := quotaRoomSetup(t, st, ThinkTankTurnDiscussion)
	now := timeNow().UTC().Add(-3 * time.Hour).Truncate(time.Second)
	reset := now.Add(time.Hour)
	due := reset.Add(time.Hour)
	if _, err := st.ObserveOwnedQuota(QuotaObservation{AgentID: source.AgentID, EpisodeID: "quota-release-hold",
		SourceGeneration: source.Generation, ExecutingTurn: source.TurnID, ObservedAt: now, ResetAt: &reset, DueAt: &due, State: QuotaScheduled}); err != nil {
		t.Fatal(err)
	}
	quota, err := st.GetQuota(source.AgentID)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.ReleaseQuotaRoomHold(d.Room.RoomID); err != nil {
		t.Fatal(err)
	}
	if got := mustTT(t, st, d.Room.RoomID).Room.Hold; got != "Quota reached" {
		t.Fatalf("released with unresolved episode: hold=%q", got)
	}
	if err := st.ClaimQuota(quota.AgentID, quota.EpisodeID, quota.Revision, "release-room-hold-token"); err != nil {
		t.Fatal(err)
	}
	quota, err = st.GetQuota(source.AgentID)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.MarkQuotaProviderStarted(quota.AgentID, quota.EpisodeID, quota.Revision, quota.AttemptToken, "release-generation", "release-turn"); err != nil {
		t.Fatal(err)
	}
	quota, err = st.GetQuota(source.AgentID)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.MarkQuotaNotice(quota.AgentID, quota.EpisodeID, quota.Revision, quota.AttemptToken, "release-notice"); err != nil {
		t.Fatal(err)
	}
	quota, err = st.GetQuota(source.AgentID)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.MarkQuotaNoticeConfirmed(quota.AgentID, quota.EpisodeID, quota.Revision, quota.AttemptToken, quota.NoticeKey); err != nil {
		t.Fatal(err)
	}
	quota, err = st.GetQuota(source.AgentID)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.ConfirmQuotaSuccess(quota.AgentID, quota.EpisodeID, quota.Revision, quota.AttemptToken, "release-generation", "release-turn"); err != nil {
		t.Fatal(err)
	}
	quota, err = st.GetQuota(source.AgentID)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.CompleteQuota(quota.AgentID, quota.EpisodeID, quota.Revision, quota.AttemptToken, "release-generation", "release-turn"); err != nil {
		t.Fatal(err)
	}
	if err := st.ReleaseQuotaRoomHold(d.Room.RoomID); err != nil {
		t.Fatal(err)
	}
	room := mustTT(t, st, d.Room.RoomID).Room
	if room.Hold != "" || room.Revision <= d.Room.Revision {
		t.Fatalf("settled quota hold was not released: %+v", room)
	}
}

func TestRecoverThinkTanksPreservesQuotaRoomHold(t *testing.T) {
	st, _ := newTestStore(t)
	d, source, _ := quotaRoomSetup(t, st, ThinkTankTurnDiscussion)
	now := timeNow().UTC().Truncate(time.Second)
	observed := now.Add(-2 * time.Minute)
	reset := now.Add(-time.Minute)
	if _, err := st.ObserveOwnedQuota(QuotaObservation{AgentID: source.AgentID, EpisodeID: "quota-restart", SourceGeneration: source.Generation, ExecutingTurn: source.TurnID,
		ObservedAt: observed, ResetAt: &reset, DueAt: &now, State: QuotaScheduled}); err != nil {
		t.Fatal(err)
	}
	if err := st.RecoverThinkTanks(); err != nil {
		t.Fatal(err)
	}
	room := mustTT(t, st, d.Room.RoomID)
	if room.Room.Control != ThinkTankRunning || room.Room.Hold != "Quota reached" {
		t.Fatalf("quota room was generically recovered: %+v", room.Room)
	}
}

func TestRecoverThinkTanksDoesNotReholdCancelledRoomQuota(t *testing.T) {
	st, _ := newTestStore(t)
	d, source, _ := quotaRoomSetup(t, st, ThinkTankTurnDiscussion)
	now := timeNow().UTC().Truncate(time.Second)
	reset := now.Add(time.Minute)
	due := reset.Add(time.Minute)
	if _, err := st.ObserveOwnedQuota(QuotaObservation{AgentID: source.AgentID, EpisodeID: "quota-cancelled-restart",
		SourceGeneration: source.Generation, ExecutingTurn: source.TurnID, ObservedAt: now,
		ResetAt: &reset, DueAt: &due, State: QuotaScheduled}); err != nil {
		t.Fatal(err)
	}
	snapshot, err := st.SnapshotAgentQuota(source.AgentID)
	if err != nil || snapshot == nil {
		t.Fatalf("quota snapshot=%+v err=%v", snapshot, err)
	}
	if _, err := st.CancelQuotaSnapshot(*snapshot); err != nil {
		t.Fatal(err)
	}
	if _, err := st.ResumeThinkTank(d.Room.RoomID); err != nil {
		t.Fatal(err)
	}
	if err := st.RecoverThinkTanks(); err != nil {
		t.Fatal(err)
	}
	if room := mustTT(t, st, d.Room.RoomID).Room; room.Hold == "Quota reached" {
		t.Fatalf("cancelled quota was reheld after restart: %+v", room)
	}
}

func TestRecoverThinkTanksFencesQuotaContinuationWithoutGenericSettlement(t *testing.T) {
	st, _ := newTestStore(t)
	d, source, completed := quotaRoomSetup(t, st, ThinkTankTurnDiscussion)
	now := timeNow().UTC().Truncate(time.Second)
	observed := now.Add(-2 * time.Minute)
	reset := now.Add(-time.Minute)
	if _, err := st.ObserveOwnedQuota(QuotaObservation{AgentID: source.AgentID, EpisodeID: "quota-active-restart", SourceGeneration: source.Generation,
		ExecutingTurn: source.TurnID, ObservedAt: observed, ResetAt: &reset, DueAt: &now, State: QuotaScheduled}); err != nil {
		t.Fatal(err)
	}
	q, err := st.GetQuota(source.AgentID)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.ClaimQuota(q.AgentID, q.EpisodeID, q.Revision, "active-restart-token"); err != nil {
		t.Fatal(err)
	}
	q, _ = st.GetQuota(source.AgentID)
	room := mustTT(t, st, d.Room.RoomID)
	resumed, err := st.BeginThinkTankAttempt(ThinkTankBegin{RoomID: d.Room.RoomID, Revision: room.Room.Revision,
		AgentID: source.AgentID, Turn: source.Turn, Generation: "quota-active-generation", TurnID: "quota-active-turn",
		QuotaContinuation: &ThinkTankQuotaContinuation{EpisodeID: q.EpisodeID, Revision: q.Revision, AttemptToken: q.AttemptToken, SourceAttemptID: source.AttemptID}})
	if err != nil {
		t.Fatal(err)
	}
	q, _ = st.GetQuota(source.AgentID)
	if err := st.MarkQuotaProviderStarted(q.AgentID, q.EpisodeID, q.Revision, q.AttemptToken, resumed.Generation, resumed.TurnID); err != nil {
		t.Fatal(err)
	}
	if err := st.RecoverQuotaClaims(); err != nil {
		t.Fatal(err)
	}
	if err := st.RecoverThinkTanks(); err != nil {
		t.Fatal(err)
	}
	q, err = st.GetQuota(source.AgentID)
	if err != nil || q.ContinuedGeneration != resumed.Generation || q.ContinuedTurn != resumed.TurnID {
		t.Fatalf("restart lost the admitted quota owner tuple: quota=%+v err=%v", q, err)
	}
	room = mustTT(t, st, d.Room.RoomID)
	if room.Room.Control != ThinkTankRunning || room.Room.Hold != "Quota reached" {
		t.Fatalf("generic recovery superseded quota hold: %+v", room.Room)
	}
	current, err := scanThinkTankAttempt(st.db.QueryRow(`SELECT `+thinkTankAttemptColumns+` FROM think_tank_attempts WHERE attempt_id=?`, resumed.AttemptID))
	if err != nil || current.State != ThinkTankAttemptFailed || current.EntrySeq != 0 || current.Message != "" {
		t.Fatalf("active uncertain continuation settled or published: %+v err=%v", current, err)
	}
	for _, member := range room.Members {
		if member.AgentID == source.AgentID && member.Completed != completed {
			t.Fatalf("startup charged quota attempt: %+v", member)
		}
	}
}
