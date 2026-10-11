package state

import (
	"errors"
	"testing"
	"time"
)

func TestPendingQuotaNoticesSeparateAppendAndProgressOnlyRecovery(t *testing.T) {
	st, _ := newTestStore(t)
	due := timeNow().Add(-time.Minute).UTC().Truncate(time.Second)
	withReceipt := quotaFixture(t, st, "quota_notice_pending", "episode-1", due)
	withoutReceipt := quotaFixture(t, st, "quota_uncertain_pending", "episode-1", due)
	for _, q := range []QuotaIntent{withReceipt, withoutReceipt} {
		if err := st.ClaimQuota(q.AgentID, q.EpisodeID, q.Revision, "attempt-"+q.AgentID); err != nil {
			t.Fatal(err)
		}
		q, _ = st.GetQuota(q.AgentID)
		if err := st.MarkQuotaProviderStarted(q.AgentID, q.EpisodeID, q.Revision, q.AttemptToken, "continued-generation", "continued-turn"); err != nil {
			t.Fatal(err)
		}
	}
	q, _ := st.GetQuota(withReceipt.AgentID)
	if err := st.MarkQuotaNotice(q.AgentID, q.EpisodeID, q.Revision, q.AttemptToken, "notice-key"); err != nil {
		t.Fatal(err)
	}
	items, err := st.PendingQuotaNotices(MaxQuotaDueBatch)
	if err != nil || len(items) != 1 || items[0].AgentID != withReceipt.AgentID {
		t.Fatalf("pending notices = %+v, %v; want only receipt-backed claim", items, err)
	}
	current, _ := st.GetQuota(withReceipt.AgentID)
	if err := st.MarkQuotaNoticeConfirmed(current.AgentID, current.EpisodeID, current.Revision, current.AttemptToken, current.NoticeKey); err != nil {
		t.Fatal(err)
	}
	if items, err := st.PendingQuotaNotices(MaxQuotaDueBatch); err != nil || len(items) != 0 {
		t.Fatalf("acknowledged progress remained pending: %+v %v", items, err)
	}
	if err := st.RecoverQuotaClaims(); err != nil {
		t.Fatal(err)
	}
	withReceipt, _ = st.GetQuota(withReceipt.AgentID)
	withoutReceipt, _ = st.GetQuota(withoutReceipt.AgentID)
	if withReceipt.State != QuotaNeedsAttention || withReceipt.NoticeKey != "notice-key" || !withReceipt.NoticeConfirmed {
		t.Fatalf("progress-only confirmed claim was not held: %+v", withReceipt)
	}
	if withoutReceipt.State != QuotaNeedsAttention || withoutReceipt.ReasonCode != "uncertain_delivery" {
		t.Fatalf("unreceipted started claim was re-admitted: %+v", withoutReceipt)
	}
	if _, err := st.PendingQuotaNotices(MaxQuotaDueBatch + 1); !errors.Is(err, ErrQuotaConflict) {
		t.Fatalf("unbounded notice batch accepted: %v", err)
	}
}
