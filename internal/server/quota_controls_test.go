package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/config"
	"github.com/AsaphNoam/Chuck/internal/state"
)

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
