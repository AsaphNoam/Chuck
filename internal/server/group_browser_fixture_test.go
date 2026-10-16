package server

import (
	"context"
	"encoding/json"
	"net"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/backend/credcheck"
	"github.com/AsaphNoam/Chuck/internal/config"
	"github.com/AsaphNoam/Chuck/internal/remote"
	"github.com/AsaphNoam/Chuck/internal/state"
)

// Opt-in isolated browser fixture for the group-journey.mjs acceptance run.
func TestGroupBrowserFixture(t *testing.T) {
	ready := os.Getenv("CHUCK_GROUP_BROWSER_READY")
	if ready == "" {
		t.Skip("manual browser fixture")
	}
	s := testServer(t, true)
	defer s.registry.Shutdown(context.Background())
	s.registry.Chat().SetCommand(buildFakeACP(t))
	if os.Getenv("CHUCK_PERSONA_BROWSER_ONBOARDING") == "1" {
		// The persona journey validates launch setup, not real-provider login.
		s.credCheck = func(context.Context, config.Backend, config.Model, map[string]string) credcheck.CredResult {
			if _, err := os.Stat(filepath.Join(s.configStore.Home(), "persona-readiness-blocked")); err == nil {
				return credcheck.CredResult{Status: "missing_credentials"}
			}
			return credcheck.CredResult{Status: "ok"}
		}
	}
	cfg, err := s.configStore.ReadConfig()
	if err != nil {
		t.Fatal(err)
	}
	cfg.OnboardingComplete = true
	if err := s.configStore.WriteConfig(cfg); err != nil {
		t.Fatal(err)
	}
	if err := s.configStore.WriteProject("other", config.Project{Title: "Other project", Cwd: t.TempDir()}); err != nil {
		t.Fatal(err)
	}
	localListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	desktop := httptest.NewUnstartedServer(s.routes())
	desktop.Listener.Close()
	desktop.Listener = localListener
	desktop.Start()
	defer desktop.Close()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	domain := listener.Addr().String()
	phone := httptest.NewUnstartedServer(s.remoteRoutes(domain, func(context.Context, string) (remote.Peer, error) {
		return remote.Peer{StableID: "browser-phone", Login: "fixture@example.com"}, nil
	}))
	phone.Listener.Close()
	phone.Listener = listener
	phone.StartTLS()
	defer phone.Close()
	token := pairTestDevice(t, s, "browser-device", "browser-phone")
	quotaAgent := ""
	if os.Getenv("CHUCK_QUOTA_BROWSER_KNOWN") == "1" {
		quotaAgent = launchAndWaitIdle(t, desktop, "implementer", "other")
		generation := s.registry.Generation(quotaAgent)
		reset := time.Now().UTC().Add(-time.Second)
		q, err := s.stateStore.ObserveQuota(state.QuotaObservation{
			AgentID: quotaAgent, EpisodeID: "browser-known-reset", SourceGeneration: generation,
			ExecutingTurn: "browser-interrupted-turn", Owner: state.QuotaOwner{Kind: "ordinary"},
			ObservedAt: reset.Add(-2 * time.Minute), ResetAt: quotaResetFor(reset), DueAt: &reset, State: state.QuotaScheduled,
		})
		if err != nil {
			t.Fatal(err)
		}
		s.startQuotaContinuation(context.Background(), q)
		deadline := time.Now().Add(15 * time.Second)
		for {
			q, err = s.stateStore.GetQuota(quotaAgent)
			if err != nil {
				t.Fatal(err)
			}
			if q.State == state.QuotaContinued {
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("known-reset browser continuation did not settle: %+v", q)
			}
			time.Sleep(10 * time.Millisecond)
		}
		waitStatus(t, s, quotaAgent, "idle")
	}
	data, _ := json.Marshal(map[string]string{"desktop": desktop.URL, "phone": phone.URL, "cookie": remoteDeviceCookie, "token": token, "home": s.configStore.Home(), "quota_agent": quotaAgent})
	if err := os.WriteFile(ready, data, 0600); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 900; i++ {
		if _, err := os.Stat(ready + ".done"); err == nil {
			return
		}
		time.Sleep(time.Second)
	}
	t.Fatal("browser fixture timed out")
}
