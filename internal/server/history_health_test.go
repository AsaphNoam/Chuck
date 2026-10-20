package server

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
)

// TS-03.R73: history health reaches the agent projection from the store, not
// from the failing transcript; no row stays absent (legacy unknown), and a
// reason appears only while degraded.
func TestAgentStateProjectsHistoryHealth(t *testing.T) {
	srv := testServer(t, true)
	agentID := seedHookAgent(t, srv)
	got, err := srv.stateMgr.Touch(agentID)
	if err != nil || got.HistoryState != "" || got.HistoryReason != "" {
		t.Fatalf("legacy projection = %q/%q, %v; want absent", got.HistoryState, got.HistoryReason, err)
	}
	if err := srv.stateStore.WriteHistoryHealth(agentID, state.HistoryHealth{State: "degraded", Reason: "sync_failed", ReservedSeq: 9, OpenWriter: true}); err != nil {
		t.Fatal(err)
	}
	if got, _ = srv.stateMgr.Touch(agentID); got.HistoryState != "degraded" || got.HistoryReason != "sync_failed" {
		t.Fatalf("degraded projection = %q/%q", got.HistoryState, got.HistoryReason)
	}
	if err := srv.stateStore.WriteHistoryHealth(agentID, state.HistoryHealth{State: "ok", Reason: "stale"}); err != nil {
		t.Fatal(err)
	}
	if got, _ = srv.stateMgr.Touch(agentID); got.HistoryState != "ok" || got.HistoryReason != "" {
		t.Fatalf("recovered projection = %q/%q", got.HistoryState, got.HistoryReason)
	}
}

// FS-03.R99: the storage latch is a definite refusal, so the composer keeps
// the draft and shows why, and no raw storage error crosses the API.
func TestHistoryUnavailableIsADefiniteRefusal(t *testing.T) {
	ae := sessionOpError(errors.Join(errors.New("write /x: no space left on device"), runtime.ErrHistoryUnavailable))
	if ae.HTTPStatus() != http.StatusConflict || ae.Code != runtime.CodeConflict {
		t.Fatalf("latch maps to %d %s", ae.HTTPStatus(), ae.Code)
	}
	if ae.Message == "" || strings.Contains(ae.Message, "no space") || strings.Contains(ae.Message, "/x") {
		t.Fatalf("latch message = %q", ae.Message)
	}
}
