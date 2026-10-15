package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/AsaphNoam/Chuck/internal/messaging"
	"github.com/AsaphNoam/Chuck/internal/state"
)

// tokenRoundTripper injects the per-agent session token header on every request.
type tokenRoundTripper struct {
	token string
	base  http.RoundTripper
}

func (t tokenRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set(messaging.TokenHeader, t.token)
	return t.base.RoundTrip(r)
}

// TestMCPRouteMounted proves the in-process MCP messaging server is reachable
// through the real dashboard mux at /mcp: a go-sdk client connects over the
// streamable HTTP transport and round-trips a `list_agents` tool call, with the
// caller resolved from the per-agent session token (Phase 5).
func TestMCPRouteMounted(t *testing.T) {
	srv := testServer(t, true)
	srv.messaging.Register("tok-route", "a_route")

	ts := httptest.NewServer(srv.routes())
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	transport := &mcp.StreamableClientTransport{
		Endpoint:   ts.URL + "/mcp",
		HTTPClient: &http.Client{Transport: tokenRoundTripper{token: "tok-route", base: http.DefaultTransport}},
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "v0.0.1"}, nil)
	cs, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("connect via dashboard mux: %v", err)
	}
	defer cs.Close()

	res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "list_agents", Arguments: nil})
	if err != nil {
		t.Fatalf("call list_agents: %v", err)
	}
	if res.IsError || len(res.Content) == 0 {
		t.Fatalf("list_agents failed: %+v", res)
	}
	if _, ok := res.Content[0].(*mcp.TextContent); !ok {
		t.Fatalf("list_agents content = %+v, want TextContent", res.Content[0])
	}
}

func TestSendMessagePublishesMailStateForWakingAndDeferred(t *testing.T) {
	for _, wake := range []bool{true, false} {
		t.Run(map[bool]string{true: "waking", false: "deferred"}[wake], func(t *testing.T) {
			srv := testServer(t, true)
			for _, agent := range []state.Agent{
				{AgentID: "a_sender", Name: "Atlas", Role: "implementer", Project: "my-app", Backend: "claude", Model: "sonnet", Interface: "chat", CreatedAt: time.Now().UTC()},
				{AgentID: "a_recipient", Name: "Nova", Role: "reviewer", Project: "my-app", Backend: "claude", Model: "sonnet", Interface: "chat", CreatedAt: time.Now().UTC()},
			} {
				if err := srv.stateStore.WriteAgent(agent); err != nil {
					t.Fatal(err)
				}
				if err := srv.stateStore.WriteRunning(state.RunningEntry{AgentID: agent.AgentID, PID: 1, SessionID: "s_" + agent.AgentID, Interface: "chat", StartedAt: time.Now().UTC()}); err != nil {
					t.Fatal(err)
				}
			}
			srv.messaging.Register("tok-sender", "a_sender")
			updates, unsubscribe := srv.eventBus.Subscribe()
			defer unsubscribe()

			ts := httptest.NewServer(srv.routes())
			defer ts.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			transport := &mcp.StreamableClientTransport{Endpoint: ts.URL + "/mcp",
				HTTPClient: &http.Client{Transport: tokenRoundTripper{token: "tok-sender", base: http.DefaultTransport}}}
			client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "v0.0.1"}, nil)
			cs, err := client.Connect(ctx, transport, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer cs.Close()
			result, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "send_message", Arguments: map[string]any{
				"to": "a_recipient", "body": "Please review", "wake": wake,
			}})
			if err != nil || result.IsError {
				t.Fatalf("send_message = %+v, %v", result, err)
			}

			gotRecipient, gotSender := false, false
			for !gotRecipient || !gotSender {
				select {
				case event := <-updates:
					if event.Type != "state_update" {
						continue
					}
					update, ok := event.Data.(state.AgentStateUpdate)
					if !ok {
						t.Fatalf("state_update payload = %T", event.Data)
					}
					switch update.AgentID {
					case "a_recipient":
						if update.UnreadMessages == 1 {
							gotRecipient = true
						}
					case "a_sender":
						if update.LastSentAt != "" {
							gotSender = true
						}
					}
				case <-ctx.Done():
					t.Fatalf("missing mail state_update: recipient=%v sender=%v", gotRecipient, gotSender)
				}
			}
		})
	}
}
