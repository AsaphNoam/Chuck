package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// hostBusyAgent leaves one turn open on an agent whose provider policy is as
// permissive as Chuck allows: skip-permissions plus automatic approval of the
// stop_agent tool. None of it may resolve a host request (FS-06.A29).
func hostBusyAgent(t *testing.T) (*ChatRuntime, *Handle, <-chan Event, string) {
	t.Helper()
	c, spec := newChatTest(t, "hold_turn")
	hold := filepath.Join(t.TempDir(), "hold")
	spec.Env = append(spec.Env, "FAKEACP_HOLD_FILE="+hold)
	spec.SkipPerms = true
	spec.AutoApproveTools = map[string]struct{}{"stop_agent": {}}
	ctx := context.Background()
	h, err := c.Start(ctx, spec)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() {
		_ = os.WriteFile(hold, []byte("go"), 0o644)
		_ = c.Stop(ctx, h.AgentID)
	})
	ch, unsub, err := c.Subscribe(h.AgentID)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(unsub)
	if err := c.SendPrompt(ctx, h.AgentID, "first"); err != nil {
		t.Fatal(err)
	}
	waitForChunk(t, ch, "working")
	c.mu.Lock()
	gen := c.agents[h.AgentID].generation
	c.mu.Unlock()
	return c, h, ch, gen
}

func openPeerStop(t *testing.T, c *ChatRuntime, agentID, gen string) *HostPermission {
	t.Helper()
	h, err := c.RequestHostPermission(agentID, HostPermissionRequest{
		Generation: gen, Origin: PermissionOriginPeerStop, Name: "Stop Nova", Reason: "stuck loop",
		PeerStop: &PeerStopSubject{
			Requester: PeerIdentity{AgentID: agentID, Name: "Atlas", Project: "my-app"},
			Target:    PeerIdentity{AgentID: "a_target", Name: "Nova", Project: "other"},
		},
	})
	if err != nil {
		t.Fatalf("RequestHostPermission: %v", err)
	}
	return h
}

func waitDecision(t *testing.T, h *HostPermission, ctx context.Context) string {
	t.Helper()
	got := make(chan string, 1)
	go func() { got <- h.Wait(ctx) }()
	select {
	case d := <-got:
		return d
	case <-time.After(3 * time.Second):
		t.Fatal("host decision never arrived")
		return ""
	}
}

// FS-03.R91: the host request is an ordinary waiting permission that only the
// operator decision route resolves, despite skip-permissions and Chuck-tool
// automatic approval; approval is admitted once.
func TestHostPermissionApprovedOnlyByOperator(t *testing.T) {
	c, h, ch, gen := hostBusyAgent(t)
	hp := openPeerStop(t, c, h.AgentID, gen)

	ev := waitForEvent(t, ch, EvPermissionRequest)
	var data PermissionRequestData
	if err := json.Unmarshal(ev.Data, &data); err != nil {
		t.Fatal(err)
	}
	if data.ToolCallID != hp.ID || data.Origin != PermissionOriginPeerStop || data.AutoApproved ||
		data.PeerStop == nil || data.PeerStop.Target.AgentID != "a_target" || len(data.Options) != 2 {
		t.Fatalf("request = %+v", data)
	}
	if st, _ := c.store.ReadStatus(h.AgentID); st.State != "waiting_input" {
		t.Fatalf("status = %q, want waiting_input", st.State)
	}
	select {
	case d := <-hp.decision:
		t.Fatalf("host request resolved without the operator: %q", d)
	case <-time.After(100 * time.Millisecond):
	}

	if err := c.Permission(context.Background(), h.AgentID, hp.ID, "approve"); err != nil {
		t.Fatalf("approve: %v", err)
	}
	if d := waitDecision(t, hp, context.Background()); d != HostApprove {
		t.Fatalf("decision = %q", d)
	}
	resolved := waitForEvent(t, ch, EvPermissionResolved)
	var rd PermissionResolvedData
	_ = json.Unmarshal(resolved.Data, &rd)
	if rd.ToolCallID != hp.ID || rd.Decision != "approve" {
		t.Fatalf("resolved = %+v", rd)
	}
	if err := c.Permission(context.Background(), h.AgentID, hp.ID, "deny"); !errors.Is(err, ErrPermissionAlreadyResolved) {
		t.Fatalf("second decision = %v", err)
	}
	if err := hp.Admit(); err != nil {
		t.Fatalf("Admit: %v", err)
	}
	if err := hp.Admit(); !errors.Is(err, ErrHostPermissionStale) {
		t.Fatalf("second Admit = %v", err)
	}
}

// FS-06.A31: deny, timeout, request cancellation and requester turn
// cancellation each end the request without an admissible approval.
func TestHostPermissionNonApprovalOutcomes(t *testing.T) {
	t.Run("deny", func(t *testing.T) {
		c, h, _, gen := hostBusyAgent(t)
		hp := openPeerStop(t, c, h.AgentID, gen)
		if err := c.Permission(context.Background(), h.AgentID, hp.ID, "deny"); err != nil {
			t.Fatal(err)
		}
		if d := waitDecision(t, hp, context.Background()); d != HostDeny {
			t.Fatalf("decision = %q", d)
		}
		if err := hp.Admit(); !errors.Is(err, ErrHostPermissionStale) {
			t.Fatalf("Admit after deny = %v", err)
		}
	})
	t.Run("timeout", func(t *testing.T) {
		t.Setenv("PERMISSION_TIMEOUT", "50ms")
		c, h, _, gen := hostBusyAgent(t)
		hp := openPeerStop(t, c, h.AgentID, gen)
		if d := waitDecision(t, hp, context.Background()); d != HostTimeout {
			t.Fatalf("decision = %q", d)
		}
	})
	t.Run("request context ends", func(t *testing.T) {
		c, h, ch, gen := hostBusyAgent(t)
		hp := openPeerStop(t, c, h.AgentID, gen)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if d := waitDecision(t, hp, ctx); d != HostCancelled {
			t.Fatalf("decision = %q", d)
		}
		resolved := waitForEvent(t, ch, EvPermissionResolved)
		var rd PermissionResolvedData
		_ = json.Unmarshal(resolved.Data, &rd)
		if rd.Decision != "cancelled" {
			t.Fatalf("resolved = %+v", rd)
		}
		if err := c.Permission(context.Background(), h.AgentID, hp.ID, "approve"); !errors.Is(err, ErrPermissionAlreadyResolved) {
			t.Fatalf("late approve = %v", err)
		}
	})
	t.Run("turn cancelled after approval", func(t *testing.T) {
		c, h, _, gen := hostBusyAgent(t)
		hp := openPeerStop(t, c, h.AgentID, gen)
		if err := c.Permission(context.Background(), h.AgentID, hp.ID, "approve"); err != nil {
			t.Fatal(err)
		}
		if d := waitDecision(t, hp, context.Background()); d != HostApprove {
			t.Fatalf("decision = %q", d)
		}
		if _, err := c.Cancel(context.Background(), h.AgentID); err != nil {
			t.Fatal(err)
		}
		if err := hp.Admit(); !errors.Is(err, ErrHostPermissionStale) {
			t.Fatalf("Admit after requester cancel = %v", err)
		}
	})
	t.Run("requester stopped while pending", func(t *testing.T) {
		c, h, _, gen := hostBusyAgent(t)
		hp := openPeerStop(t, c, h.AgentID, gen)
		_ = c.Stop(context.Background(), h.AgentID)
		if d := waitDecision(t, hp, context.Background()); d != HostCancelled {
			t.Fatalf("decision = %q", d)
		}
	})
}

// FS-06.A31: a request is bound to the requester's current generation and an
// executing turn.
func TestHostPermissionRequiresCurrentTurn(t *testing.T) {
	c, h, _, gen := hostBusyAgent(t)
	if _, err := c.RequestHostPermission(h.AgentID, HostPermissionRequest{Generation: gen + "-old",
		Origin: PermissionOriginPeerStop, Name: "Stop"}); !errors.Is(err, ErrHostPermissionStale) {
		t.Fatalf("stale generation = %v", err)
	}
}
