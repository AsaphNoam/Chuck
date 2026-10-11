package server

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/config"
	"github.com/AsaphNoam/Chuck/internal/state"
)

// FS-01.R34 / TS-10.R43: an owner reserves its captured agents, automatic
// contention returns to the due queue, and unrelated Stop remains available.
func TestQuotaOwnerActionScopesContentionAndDefersAdmission(t *testing.T) {
	s := testServer(t, true)
	q := seedQuotaHTTP(t, s, "owner-action-target")
	other := seedQuotaHTTP(t, s, "unrelated-stop-target")
	if err := s.stateStore.ClaimQuota(q.AgentID, q.EpisodeID, q.Revision, "owner-action-attempt"); err != nil {
		t.Fatal(err)
	}
	q, err := s.stateStore.GetQuota(q.AgentID)
	if err != nil {
		t.Fatal(err)
	}
	entered, release, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }); <-done }
	handler := s.quotaControlHandler(func(*http.Request) ([]state.QuotaControlSnapshot, error) {
		return s.captureQuotaSnapshots([]string{q.AgentID})
	}, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(entered)
		<-release
		w.WriteHeader(http.StatusConflict) // preserve the deferral for the next tick
	}), "task")
	go func() {
		defer close(done)
		r := newLocalRequest(http.MethodPost, "/api/tasks/owner/cancel", nil)
		r.SetPathValue("id", "owner")
		handler.ServeHTTP(httptest.NewRecorder(), r)
	}()
	defer unblock()
	<-entered
	if w := quotaHTTP(t, s, http.MethodPost, "/api/sessions/"+q.AgentID+"/stop", ""); w.Code != http.StatusConflict {
		t.Fatalf("target Stop during owner action: %d %s", w.Code, w.Body.String())
	}
	// Even while the policy lock is held, unrelated Stop does not join it.
	s.quotaControlMu.Lock()
	w := quotaHTTP(t, s, http.MethodPost, "/api/sessions/"+other.AgentID+"/stop", "")
	s.quotaControlMu.Unlock()
	if w.Code != http.StatusOK {
		t.Fatalf("unrelated Stop: %d %s", w.Code, w.Body.String())
	}
	s.startQuotaContinuation(context.Background(), q)
	fresh, err := s.stateStore.GetQuota(q.AgentID)
	if err != nil || fresh.State != state.QuotaDeferred || fresh.ProviderStarted {
		t.Fatalf("contended admission left stuck claim: %+v err=%v", fresh, err)
	}
	due, err := s.stateStore.DueQuotaIntents(fresh.DueAt.Add(time.Second), state.MaxQuotaDueBatch)
	if err != nil || len(due) != 1 || due[0].AgentID != q.AgentID {
		t.Fatalf("deferral absent from next due pass: %+v err=%v", due, err)
	}
	unblock()
	if ae := s.stopAgent(context.Background(), q.AgentID); ae != nil {
		t.Fatalf("reservation not released: %v", ae)
	}
}

func TestQuotaControlWrapperPreservesRefusalAndReplacedEpisode(t *testing.T) {
	s := testServer(t, true)
	q := seedQuotaHTTP(t, s, "quota-control-wrapper")
	route := s.quotaControlHandler(s.quotaAgentCapture, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"error":"refused"}`, http.StatusConflict)
	}))
	request := func(next http.Handler) *httptest.ResponseRecorder {
		r := newLocalRequest(http.MethodPost, "/api/sessions/"+q.AgentID+"/prompt", nil)
		r.SetPathValue("id", q.AgentID)
		w := httptest.NewRecorder()
		next.ServeHTTP(w, r)
		return w
	}
	if w := request(route); w.Code != http.StatusConflict {
		t.Fatalf("refusal status=%d", w.Code)
	}
	current, err := s.stateStore.GetQuota(q.AgentID)
	if err != nil || current.State != state.QuotaScheduled {
		t.Fatalf("refused action changed quota: %+v err=%v", current, err)
	}
	accepted := s.quotaControlHandler(s.quotaAgentCapture, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		now := time.Now().UTC().Truncate(time.Second)
		reset, due := now.Add(time.Minute), now.Add(2*time.Minute)
		_, err := s.stateStore.ObserveQuota(state.QuotaObservation{AgentID: q.AgentID, EpisodeID: "episode-replaced", SourceGeneration: "new-generation", ExecutingTurn: "new-turn",
			Owner: state.QuotaOwner{Kind: "ordinary"}, ObservedAt: now, ResetAt: &reset, DueAt: &due, State: state.QuotaScheduled})
		if err != nil {
			t.Errorf("replace episode: %v", err)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	if w := request(accepted); w.Code != http.StatusAccepted {
		t.Fatalf("accepted status=%d body=%s", w.Code, w.Body.String())
	}
	current, err = s.stateStore.GetQuota(q.AgentID)
	if err != nil || current.EpisodeID != "episode-replaced" || current.State != state.QuotaScheduled {
		t.Fatalf("manual action cancelled replacement episode: %+v err=%v", current, err)
	}
}

func TestProjectArchiveFailurePreservesPendingQuota(t *testing.T) {
	s := testServer(t, true)
	q := seedQuotaHTTP(t, s, "quota-project-archive-failure")
	s.writeProject = func(string, config.Project) error { return fmt.Errorf("injected archive publication failure") }
	w := quotaHTTP(t, s, http.MethodPost, "/api/projects/my-app/archive", "")
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("archive status=%d body=%s", w.Code, w.Body.String())
	}
	current, err := s.stateStore.GetQuota(q.AgentID)
	if err != nil || current.EpisodeID != q.EpisodeID || current.State != state.QuotaScheduled {
		t.Fatalf("failed project archive changed quota: %+v err=%v", current, err)
	}
}

func TestSuccessfulGroupAndProjectArchiveCancelCapturedQuota(t *testing.T) {
	for _, action := range []string{"group", "project"} {
		t.Run(action, func(t *testing.T) {
			s := testServer(t, true)
			q := seedQuotaHTTP(t, s, "quota-archive-"+action)
			if action == "group" {
				agent, err := s.stateStore.ReadAgent(q.AgentID)
				if err != nil {
					t.Fatal(err)
				}
				agent.Group = "ops"
				if err := s.stateStore.WriteAgent(agent); err != nil {
					t.Fatal(err)
				}
				w := quotaHTTP(t, s, http.MethodPost, "/api/projects/my-app/groups/archive", `{"group":"ops"}`)
				if w.Code != http.StatusOK {
					t.Fatalf("group archive status=%d body=%s", w.Code, w.Body.String())
				}
			} else {
				w := quotaHTTP(t, s, http.MethodPost, "/api/projects/my-app/archive", "")
				if w.Code != http.StatusOK {
					t.Fatalf("project archive status=%d body=%s", w.Code, w.Body.String())
				}
			}
			current, err := s.stateStore.GetQuota(q.AgentID)
			if err != nil || current.State != state.QuotaCancelled {
				t.Fatalf("archive did not cancel quota: %+v err=%v", current, err)
			}
		})
	}
}

// Archive publishes every committed member once, including members without any quota history.
func TestGroupArchivePublishesMixedQuotaMembers(t *testing.T) {
	s := testServer(t, true)
	q := seedQuotaHTTP(t, s, "mixed-quota")
	for _, id := range []string{q.AgentID, "mixed-ordinary"} {
		agent := state.Agent{AgentID: id, Name: id, Project: "my-app", Group: "ops", Interface: "chat", CreatedAt: time.Now().UTC()}
		if err := s.stateStore.WriteAgent(agent); err != nil {
			t.Fatal(err)
		}
		if _, err := s.stateMgr.Touch(id); err != nil {
			t.Fatal(err)
		}
	}
	events, unsubscribe := s.eventBus.Subscribe()
	defer unsubscribe()
	w := quotaHTTP(t, s, http.MethodPost, "/api/projects/my-app/groups/archive", `{"group":"ops"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("archive: %d %s", w.Code, w.Body.String())
	}
	counts := map[string]int{}
	for len(events) > 0 {
		ev := <-events
		if ev.Type != "state_update" {
			continue
		}
		u := ev.Data.(state.AgentStateUpdate)
		if !u.Archived {
			t.Fatalf("published unarchived member: %+v", u)
		}
		counts[u.AgentID]++
	}
	for _, id := range []string{q.AgentID, "mixed-ordinary"} {
		if counts[id] != 1 {
			t.Fatalf("%s updates=%d, want one", id, counts[id])
		}
	}
	snapshot, _, disconnect := s.eventBus.SubscribeWithSnapshot()
	disconnect()
	if len(snapshot) != 2 {
		t.Fatalf("reconnect members: %+v", snapshot)
	}
	for _, u := range snapshot {
		if !u.Archived || u.AgentID == q.AgentID && (u.Quota == nil || u.Quota.ContinuationState != state.QuotaCancelled) {
			t.Fatalf("reconnect archive projection: %+v", u)
		}
	}
}
