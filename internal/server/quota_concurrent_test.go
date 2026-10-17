package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/state"
)

// FS-01.A23: simultaneous terminal failures from agents sharing one backend
// remain independent unknown-reset interruptions in list, detail, and SSE.
func TestConcurrentQuotaFailuresStayScopedPerAgent(t *testing.T) {
	t.Setenv("FAKEACP_CAPS", "1")
	t.Setenv("FAKEACP_FAILURE_RESULT", `{"id":"shared-backend-limit","revision":1,"category":"limit","severity":"error","title":"Account quota exhausted","actions":[]}`)
	s, ts, _ := activationTestServer(t)
	ids := make([]string, 3)
	gens := make(map[string]string, len(ids))
	for i := range ids {
		ids[i] = launchAndWaitIdle(t, ts, "impl", "tmpproj")
		gens[ids[i]] = s.registry.Generation(ids[i])
	}
	events, unsubscribe := s.eventBus.Subscribe()
	defer unsubscribe()

	type promptResponse struct {
		status int
		err    error
	}
	responses := make(chan promptResponse, len(ids))
	for _, id := range ids {
		id := id
		go func() {
			data, _ := json.Marshal(map[string]string{"text": "continue"})
			resp, err := http.Post(ts.URL+"/api/sessions/"+id+"/prompt", "application/json", bytes.NewReader(data))
			if err != nil {
				responses <- promptResponse{err: err}
				return
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			responses <- promptResponse{status: resp.StatusCode}
		}()
	}
	for range ids {
		result := <-responses
		if result.err != nil || result.status != http.StatusAccepted {
			t.Fatalf("prompt accepted with status %d: %v", result.status, result.err)
		}
	}

	byID := make(map[string]state.QuotaIntent, len(ids))
	deadline := time.Now().Add(10 * time.Second)
	for len(byID) != len(ids) {
		for _, id := range ids {
			if _, ok := byID[id]; ok {
				continue
			}
			q, err := s.stateStore.GetQuota(id)
			if err == nil {
				byID[id] = q
			}
		}
		if len(byID) == len(ids) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("quota failures did not settle for every agent: %v", byID)
		}
		time.Sleep(10 * time.Millisecond)
	}

	episodes := make(map[string]bool, len(ids))
	for _, id := range ids {
		q := byID[id]
		if q.AgentID != id || q.SourceGeneration != gens[id] || q.ExecutingTurn == "" || q.EpisodeID == "" || q.Revision < 1 {
			t.Fatalf("quota record not scoped to %s: %+v", id, q)
		}
		if episodes[q.EpisodeID] {
			t.Fatalf("agents shared quota episode %q", q.EpisodeID)
		}
		episodes[q.EpisodeID] = true
		// An unknown reset schedules only an isolated check, never a fabricated
		// reset or a continuation (FS-01.R49).
		if q.State != state.QuotaScheduled || q.NextAction != state.QuotaActionCheck || q.ResetAt != nil || q.DueAt == nil {
			t.Fatalf("unknown provider reset for %s: %+v, want a pending check", id, q)
		}
	}

	var listed []sessionEntry
	resp, body := quotaGET(t, ts.URL+"/api/sessions")
	if resp.StatusCode != http.StatusOK || json.Unmarshal(body, &listed) != nil {
		t.Fatalf("sessions list: status=%d body=%s", resp.StatusCode, body)
	}
	assertNoQuotaAuthority(t, body)
	for _, id := range ids {
		var projection *state.QuotaProjection
		for _, entry := range listed {
			if entry.AgentID == id {
				projection = entry.Quota
				break
			}
		}
		assertQuotaProjection(t, id, byID[id], projection)
		resp, body = quotaGET(t, ts.URL+"/api/sessions/"+id)
		var detail sessionResponse
		if resp.StatusCode != http.StatusOK || json.Unmarshal(body, &detail) != nil {
			t.Fatalf("session detail %s: status=%d body=%s", id, resp.StatusCode, body)
		}
		assertQuotaProjection(t, id, byID[id], detail.Quota)
		assertNoQuotaAuthority(t, body)
		// A fresh recomputation publishes the same per-agent projection to SSE.
		if _, err := s.stateMgr.Touch(id); err != nil {
			t.Fatal(err)
		}
	}

	seen := map[string]state.QuotaProjection{}
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	for len(seen) < len(ids) {
		select {
		case ev := <-events:
			if ev.Type != "state_update" {
				continue
			}
			raw, _ := json.Marshal(ev.Data)
			assertNoQuotaAuthority(t, raw)
			var update state.AgentStateUpdate
			if json.Unmarshal(raw, &update) != nil || update.Quota == nil {
				continue
			}
			seen[update.AgentID] = *update.Quota
		case <-timer.C:
			t.Fatalf("SSE projections missing per-agent updates: got %v", seen)
		}
	}
	for _, id := range ids {
		projection := seen[id]
		assertQuotaProjection(t, id, byID[id], &projection)
	}
}

func assertQuotaProjection(t *testing.T, id string, q state.QuotaIntent, got *state.QuotaProjection) {
	t.Helper()
	if got == nil || got.EpisodeID != q.EpisodeID || got.Revision != q.Revision || got.ContinuationState != state.QuotaScheduled ||
		got.NextAction != "check" || got.ResetAt != nil || got.DueAt == nil || !got.DueAt.Equal(*q.DueAt) {
		t.Fatalf("projection for %s = %+v, want episode=%s revision=%d unknown reset with a pending check", id, got, q.EpisodeID, q.Revision)
	}
}

func assertNoQuotaAuthority(t *testing.T, body []byte) {
	t.Helper()
	for _, secret := range []string{"source_generation", "attempt_token", "shared-backend-limit", "Account quota exhausted"} {
		if strings.Contains(string(body), secret) {
			t.Fatalf("quota projection leaked authority/provider data %q: %s", secret, body)
		}
	}
}

func quotaGET(t *testing.T, url string) (*http.Response, []byte) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	return resp, body
}
