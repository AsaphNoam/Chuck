package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
)

func quotaHTTP(t *testing.T, s *Server, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := newLocalRequest(method, path, strings.NewReader(body))
	w := httptest.NewRecorder()
	s.routes().ServeHTTP(w, r)
	return w
}

func seedQuotaHTTP(t *testing.T, s *Server, id string) state.QuotaIntent {
	t.Helper()
	if err := s.stateStore.WriteAgent(state.Agent{AgentID: id, Name: id, Role: "default", Project: "my-app", Backend: "claude", Model: "sonnet", Interface: "chat", CreatedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	observed := now.Add(-2 * time.Minute)
	reset := now.Add(-time.Minute)
	due := now
	q, err := s.stateStore.ObserveQuota(state.QuotaObservation{AgentID: id, EpisodeID: "episode", SourceGeneration: "generation", ExecutingTurn: "turn", Owner: state.QuotaOwner{Kind: "ordinary"}, ObservedAt: observed, ResetAt: &reset, DueAt: &due, State: state.QuotaScheduled})
	if err != nil {
		t.Fatal(err)
	}
	return q
}

// FS-01.A26: a stopped conversation's schedule is cancellable independently of
// an executing turn, with stale-target rejection and idempotent replay.
func TestQuotaCancellationExplicitStoppedTarget(t *testing.T) {
	s := testServer(t, true)
	q := seedQuotaHTTP(t, s, "quota-stopped")
	for _, body := range []string{`{}`, `{"target":"other"}`, `{"target":"quota_continuation"}`} {
		w := quotaHTTP(t, s, http.MethodPost, "/api/sessions/"+q.AgentID+"/cancel", body)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("invalid cancel %s: %d %s", body, w.Code, w.Body.String())
		}
	}
	w := quotaHTTP(t, s, http.MethodPost, "/api/sessions/"+q.AgentID+"/cancel", `{"target":"quota_continuation","episode_id":"stale","revision":1}`)
	if w.Code != http.StatusConflict {
		t.Fatalf("stale: %d %s", w.Code, w.Body.String())
	}
	body := `{"target":"quota_continuation","episode_id":"episode","revision":1}`
	w = quotaHTTP(t, s, http.MethodPost, "/api/sessions/"+q.AgentID+"/cancel", body)
	if w.Code != http.StatusAccepted || !strings.Contains(w.Body.String(), `"cancelled":true`) {
		t.Fatalf("cancel: %d %s", w.Code, w.Body.String())
	}
	w = quotaHTTP(t, s, http.MethodPost, "/api/sessions/"+q.AgentID+"/cancel", body)
	if w.Code != http.StatusAccepted || !strings.Contains(w.Body.String(), `"cancelled":false`) {
		t.Fatalf("replay: %d %s", w.Code, w.Body.String())
	}
}

// FS-04.A33 / FS-01.A26: effective policy is default-on, false saves before
// cancellation, and re-enable never resurrects a cancelled intent.
func TestQuotaConfigLiveDisableAndValidation(t *testing.T) {
	s := testServer(t, true)
	q := seedQuotaHTTP(t, s, "quota-config")
	w := quotaHTTP(t, s, http.MethodPut, "/api/config", `{"auto_continue":null}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("null config: %d %s", w.Code, w.Body.String())
	}
	w = quotaHTTP(t, s, http.MethodPut, "/api/config", `{"auto_continue":false}`)
	if w.Code != http.StatusOK {
		t.Fatalf("disable: %d %s", w.Code, w.Body.String())
	}
	q, err := s.stateStore.GetQuota(q.AgentID)
	if err != nil || q.State != state.QuotaCancelled {
		t.Fatalf("pending after disable: %+v %v", q, err)
	}
	w = quotaHTTP(t, s, http.MethodPut, "/api/config", `{"auto_continue":true}`)
	if w.Code != http.StatusOK {
		t.Fatalf("enable: %d %s", w.Code, w.Body.String())
	}
	q, _ = s.stateStore.GetQuota(q.AgentID)
	if q.State != state.QuotaCancelled {
		t.Fatalf("re-enable resurrected %+v", q)
	}
}

// FS-01.A23: normalized wire evidence produces serialized quota state without
// leaking source authority/provider titles and without inventing a reset.
func TestQuotaUnknownResetProjection(t *testing.T) {
	s := testServer(t, true)
	q := seedQuotaHTTP(t, s, "quota-projection")
	if err := s.settleQuotaTurn(q.AgentID, "generation", "another-turn", &runtime.QuotaEvidence{FailureID: "typed-failure", Revision: 1}, false); err != nil {
		t.Fatal(err)
	}
	u, err := s.stateMgr.Touch(q.AgentID)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(u)
	if err != nil {
		t.Fatal(err)
	}
	if u.Quota == nil || u.Quota.ContinuationState != state.QuotaUnknownReset || u.Quota.ResetAt != nil || u.Quota.DueAt != nil {
		t.Fatalf("projection: %s", raw)
	}
	if strings.Contains(string(raw), "typed-failure") || strings.Contains(string(raw), "source_generation") || strings.Contains(string(raw), "attempt_token") {
		t.Fatalf("authority leaked: %s", raw)
	}
}

// FS-04.A33: partial cancellation failure reports the policy already saved to
// disk; automatic admission still refuses the disabled policy.
func TestQuotaConfigReportsSavedFalseWhenCancellationFails(t *testing.T) {
	s := testServer(t, true)
	if _, err := s.stateStore.DB().Exec(`DROP TABLE quota_intents`); err != nil {
		t.Fatal(err)
	}
	w := quotaHTTP(t, s, http.MethodPut, "/api/config", `{"auto_continue":false}`)
	if w.Code != http.StatusInternalServerError || !strings.Contains(w.Body.String(), `"auto_continue":false`) {
		t.Fatalf("partial disable: %d %s", w.Code, w.Body.String())
	}
	cfg, err := s.configStore.ReadConfig()
	if err != nil || cfg.AutoContinue {
		t.Fatalf("actual saved policy: %+v %v", cfg, err)
	}
}

func TestQuotaFailedTurnPropagatesStateReadError(t *testing.T) {
	s := testServer(t, true)
	if _, err := s.stateStore.DB().Exec(`DROP TABLE quota_intents`); err != nil {
		t.Fatal(err)
	}
	if err := s.settleQuotaTurn("agent", "generation", "turn", nil, false); err == nil {
		t.Fatal("failed turn swallowed quota state read error")
	}
}
