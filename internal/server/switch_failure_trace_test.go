package server

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/AsaphNoam/Chuck/internal/runtime"
)

// UR-2026-11-21 J7: the browser went away mid-switch (Claude → Codex) and the
// request context's cancellation aborted the target start after the old
// runtime had stopped, so the switch rolled back and answered 500 with no log
// line. Once the switch has begun tearing down the source it must finish
// independently of the client (FS-01.R13/R26).
func TestSwitchCompletesAfterClientDisconnect(t *testing.T) {
	srv, ts := switchTestServer(t)
	id := launchAndWaitIdle(t, ts, "impl", "tmpproj")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // the client is already gone when the handler runs
	req := httptest.NewRequest(http.MethodPost, "/api/sessions/"+id+"/switch-runtime",
		strings.NewReader(`{"backend":"codex","model":"gpt-5.5"}`)).WithContext(ctx)
	req.SetPathValue("id", id)
	rec := httptest.NewRecorder()
	srv.handleSwitchRuntime(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("switch after disconnect = %d: %s", rec.Code, rec.Body.String())
	}
	if a, err := srv.stateStore.ReadAgent(id); err != nil || a.Backend != "codex" || a.Model != "gpt-5.5" {
		t.Fatalf("identity = %+v, %v; want codex/gpt-5.5", a, err)
	}
	if run, err := srv.stateStore.ReadRunning(id); err != nil || srv.registry.Generation(id) != run.Generation {
		t.Fatalf("target runtime not running: %+v, %v", run, err)
	}
}

// A switch that fails after stopping the source leaves one bounded Chuck log
// record naming the agent, the operation and the typed reason (INV 22), for
// both the rolled-back and the unrecoverable outcome.
func TestSwitchFailureIsLogged(t *testing.T) {
	for _, tc := range []struct {
		name, code    string
		breakRollback bool
	}{
		{"rolled back", runtime.CodeSwitchFailedRolledBack, false},
		{"rollback failed", runtime.CodeSwitchFailed, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv, ts := switchTestServer(t)
			id := launchAndWaitIdle(t, ts, "impl", "tmpproj")
			var logs bytes.Buffer
			var mu sync.Mutex
			srv.log = slog.New(slog.NewJSONHandler(&lockedWriter{w: &logs, mu: &mu}, nil))
			srv.terminal.SetCommand("/nonexistent/chuck-no-such-binary")
			if tc.breakRollback {
				installAppliedIdentityFailure(t, srv, "NEW.interface = 'chat'")
			}

			resp, body := post(t, ts.URL+"/api/sessions/"+id+"/switch-runtime", map[string]string{"interface": "terminal"})
			if resp.StatusCode != http.StatusInternalServerError || apiErrorCode(t, body) != tc.code {
				t.Fatalf("switch = %d: %s", resp.StatusCode, body)
			}
			mu.Lock()
			out := logs.String()
			mu.Unlock()
			var line string
			for _, l := range strings.Split(out, "\n") {
				if strings.Contains(l, `"operation":"switch-runtime"`) {
					line = l
				}
			}
			if line == "" || !strings.Contains(line, `"agent_id":"`+id+`"`) || !strings.Contains(line, `"code":"`+tc.code+`"`) ||
				!strings.Contains(line, `"reason":"`) {
				t.Fatalf("no correlated switch failure record; logs:\n%s", out)
			}
		})
	}
}
