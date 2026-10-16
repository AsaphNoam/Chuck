package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/messaging"
	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
	"github.com/AsaphNoam/Chuck/internal/transcript"
)

// peerStopFixture is a requester with an open turn and an idle target, both
// real fake-ACP chat agents behind the real routes (FS-06.A29–A31).
func peerStopFixture(t *testing.T) (*Server, *httptest.Server, messaging.SessionIdentity, string) {
	t.Helper()
	srv, ts, requester, _ := busyChatServer(t, nil)
	target := launchAndWaitIdle(t, ts, "impl", "tmpproj")
	return srv, ts, messaging.SessionIdentity{AgentID: requester, Generation: srv.registry.Generation(requester)}, target
}

func startPeerStop(srv *Server, requester messaging.SessionIdentity, target string) (<-chan messaging.PeerStopResult, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	out := make(chan messaging.PeerStopResult, 1)
	go func() { out <- srv.StopPeer(ctx, requester, target, "it is looping") }()
	return out, cancel
}

// waitHostRequest returns the pending host request recorded on the requester.
func waitHostRequest(t *testing.T, srv *Server, requester string) runtime.PermissionRequestData {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		events, _ := transcript.ReadFile(srv.configStore.Home(), requester, transcript.ReadOptions{})
		for i := len(events) - 1; i >= 0; i-- {
			if events[i].Type != runtime.EvPermissionRequest {
				continue
			}
			var d runtime.PermissionRequestData
			if json.Unmarshal(events[i].Data, &d) == nil && d.Origin == runtime.PermissionOriginPeerStop {
				return d
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("no host peer Stop request was recorded")
	return runtime.PermissionRequestData{}
}

func decide(t *testing.T, ts *httptest.Server, requester, id, decision string) {
	t.Helper()
	resp, body := post(t, ts.URL+"/api/sessions/"+requester+"/permission", map[string]string{"tool_call_id": id, "decision": decision})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("permission %s = %d: %s", decision, resp.StatusCode, body)
	}
}

func awaitResult(t *testing.T, ch <-chan messaging.PeerStopResult) messaging.PeerStopResult {
	t.Helper()
	select {
	case res := <-ch:
		return res
	case <-time.After(10 * time.Second):
		t.Fatal("peer Stop never returned")
		return messaging.PeerStopResult{}
	}
}

func isRunning(t *testing.T, srv *Server, id string) bool {
	t.Helper()
	_, err := srv.stateStore.ReadRunning(id)
	if err != nil && !errors.Is(err, state.ErrNotFound) {
		t.Fatal(err)
	}
	return err == nil
}

func peerNotices(t *testing.T, srv *Server, id string) []runtime.PeerStopNotice {
	t.Helper()
	events, err := transcript.ReadFile(srv.configStore.Home(), id, transcript.ReadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	var out []runtime.PeerStopNotice
	for _, ev := range events {
		var n runtime.NoticeData
		if ev.Type == runtime.EvNotice && json.Unmarshal(ev.Data, &n) == nil && n.PeerStop != nil {
			out = append(out, *n.PeerStop)
		}
	}
	return out
}

// FS-06.A29/A30: the target stays live until the operator approves this
// request; denial changes nothing, and approval runs ordinary Stop with
// attributed notices on both agents and a later ordinary Resume.
func TestPeerStopRequiresOperatorApproval(t *testing.T) {
	srv, ts, requester, target := peerStopFixture(t)

	denied, _ := startPeerStop(srv, requester, target)
	req := waitHostRequest(t, srv, requester.AgentID)
	if req.PeerStop == nil || req.PeerStop.Target.AgentID != target || req.PeerStop.Requester.AgentID != requester.AgentID ||
		req.Reason != "it is looping" || req.AutoApproved {
		t.Fatalf("host request = %+v", req)
	}
	time.Sleep(100 * time.Millisecond)
	if !isRunning(t, srv, target) {
		t.Fatal("target stopped before any decision")
	}
	decide(t, ts, requester.AgentID, req.ToolCallID, "deny")
	if res := awaitResult(t, denied); res.Code != "peer_stop_denied" || res.RequestID != req.ToolCallID {
		t.Fatalf("deny result = %+v", res)
	}
	if !isRunning(t, srv, target) || len(peerNotices(t, srv, target)) != 0 {
		t.Fatal("denied Stop changed the target")
	}

	approved, _ := startPeerStop(srv, requester, target)
	var second runtime.PermissionRequestData
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
		if second = waitHostRequest(t, srv, requester.AgentID); second.ToolCallID != req.ToolCallID {
			break
		}
	}
	decide(t, ts, requester.AgentID, second.ToolCallID, "approve")
	res := awaitResult(t, approved)
	if res.Code != "" || res.Outcome != "stopped" || res.RequestID != second.ToolCallID {
		t.Fatalf("approve result = %+v", res)
	}
	if isRunning(t, srv, target) {
		t.Fatal("approved Stop left the target running")
	}
	if got := peerNotices(t, srv, target); len(got) != 1 || got[0].Phase != "approved" || got[0].RequesterID != requester.AgentID {
		t.Fatalf("target notices = %+v", got)
	}
	if got := peerNotices(t, srv, requester.AgentID); len(got) != 1 || got[0].Phase != "stopped" || got[0].TargetID != target {
		t.Fatalf("requester notices = %+v", got)
	}

	// Stop preserves history and a later ordinary Resume still works.
	resp, body := post(t, ts.URL+"/api/sessions/"+target+"/resume", map[string]string{})
	if resp.StatusCode >= 300 {
		t.Fatalf("resume after peer Stop = %d: %s", resp.StatusCode, body)
	}
	if !isRunning(t, srv, target) {
		t.Fatal("resume did not restart the target")
	}
}

// FS-06.A31: an already-stopped target is a no-op without a prompt; an
// approval never transfers to a later target generation; and requester
// cancellation before admission ends the request without stopping anything.
func TestPeerStopRefusesStaleOrSettledTargets(t *testing.T) {
	t.Run("already stopped", func(t *testing.T) {
		srv, ts, requester, target := peerStopFixture(t)
		if resp, body := post(t, ts.URL+"/api/sessions/"+target+"/stop", map[string]string{}); resp.StatusCode != http.StatusOK {
			t.Fatalf("stop = %d: %s", resp.StatusCode, body)
		}
		res := srv.StopPeer(context.Background(), requester, target, "done")
		if res.Code != "" || res.Outcome != "already_stopped" || res.RequestID != "" {
			t.Fatalf("result = %+v", res)
		}
	})
	t.Run("target restarted before approval", func(t *testing.T) {
		srv, ts, requester, target := peerStopFixture(t)
		ch, _ := startPeerStop(srv, requester, target)
		req := waitHostRequest(t, srv, requester.AgentID)
		post(t, ts.URL+"/api/sessions/"+target+"/stop", map[string]string{})
		if resp, body := post(t, ts.URL+"/api/sessions/"+target+"/resume", map[string]string{}); resp.StatusCode >= 300 {
			t.Fatalf("resume = %d: %s", resp.StatusCode, body)
		}
		decide(t, ts, requester.AgentID, req.ToolCallID, "approve")
		if res := awaitResult(t, ch); res.Code != "peer_stop_stale" {
			t.Fatalf("result = %+v", res)
		}
		if !isRunning(t, srv, target) || len(peerNotices(t, srv, target)) != 0 {
			t.Fatal("stale approval stopped or annotated the later generation")
		}
	})
	t.Run("requester call cancelled while pending", func(t *testing.T) {
		srv, ts, requester, target := peerStopFixture(t)
		ch, cancel := startPeerStop(srv, requester, target)
		req := waitHostRequest(t, srv, requester.AgentID)
		cancel()
		if res := awaitResult(t, ch); res.Code != "peer_stop_cancelled" {
			t.Fatalf("result = %+v", res)
		}
		resp, _ := post(t, ts.URL+"/api/sessions/"+requester.AgentID+"/permission",
			map[string]string{"tool_call_id": req.ToolCallID, "decision": "approve"})
		if resp.StatusCode == http.StatusOK || !isRunning(t, srv, target) {
			t.Fatalf("late approval status %d, target running %v", resp.StatusCode, isRunning(t, srv, target))
		}
	})
	t.Run("one outstanding request per requester", func(t *testing.T) {
		srv, _, requester, target := peerStopFixture(t)
		_, cancel := startPeerStop(srv, requester, target)
		defer cancel()
		waitHostRequest(t, srv, requester.AgentID)
		if res := srv.StopPeer(context.Background(), requester, target, "again"); res.Code != "peer_control_busy" {
			t.Fatalf("second request = %+v", res)
		}
	})
	t.Run("stale requester generation", func(t *testing.T) {
		srv, _, requester, target := peerStopFixture(t)
		requester.Generation += "-old"
		if res := srv.StopPeer(context.Background(), requester, target, "x"); res.Code != "peer_stop_stale" {
			t.Fatalf("result = %+v", res)
		}
	})
}
