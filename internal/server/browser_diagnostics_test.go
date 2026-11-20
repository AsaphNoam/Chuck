package server

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func postBrowserDiagnostic(t *testing.T, h http.Handler, body string) int {
	t.Helper()
	req := newLocalRequest(http.MethodPost, "/api/diagnostics/browser", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code
}

func browserDiagnosticLines(buf *bytes.Buffer) []string {
	var lines []string
	for _, line := range strings.Split(buf.String(), "\n") {
		if strings.Contains(line, `"msg":"browser diagnostic"`) {
			lines = append(lines, line)
		}
	}
	return lines
}

// TS-03.R92 / INV §22: a desktop browser failure reaches the dashboard log as
// allowlisted words only, and a page cannot flood it.
func TestBrowserDiagnosticLogsOnlyAllowlistedFields(t *testing.T) {
	s := testServer(t, true)
	var buf bytes.Buffer
	s.log = slog.New(slog.NewJSONHandler(&buf, nil))
	h := s.routes()

	if code := postBrowserDiagnostic(t, h, `{"reason":"shell_missing","route_kind":"agent","mount_state":"missing","visit_id":"abc123def456","message":"secret page text"}`); code != http.StatusNoContent {
		t.Fatalf("valid diagnostic status = %d, want 204", code)
	}
	lines := browserDiagnosticLines(&buf)
	if len(lines) != 1 {
		t.Fatalf("diagnostic records = %d, want 1: %s", len(lines), buf.String())
	}
	for _, want := range []string{`"level":"WARN"`, `"reason":"shell_missing"`, `"route_kind":"agent"`, `"mount_state":"missing"`, `"visit_id":"abc123def456"`} {
		if !strings.Contains(lines[0], want) {
			t.Errorf("record missing %s: %s", want, lines[0])
		}
	}
	if strings.Contains(buf.String(), "secret page text") {
		t.Fatalf("unlisted field reached the log: %s", buf.String())
	}

	for _, bad := range []string{
		`{"reason":"Uncaught TypeError: x","route_kind":"agent","mount_state":"mounted","visit_id":"abc123def456"}`,
		`{"reason":"error","route_kind":"/agent/a_1","mount_state":"mounted","visit_id":"abc123def456"}`,
		`{"reason":"error","route_kind":"agent","mount_state":"mounted","visit_id":"../x"}`,
		`not json`,
	} {
		if code := postBrowserDiagnostic(t, h, bad); code != http.StatusBadRequest {
			t.Errorf("diagnostic %s status = %d, want 400", bad, code)
		}
	}
	if strings.Contains(buf.String(), "Uncaught TypeError") || strings.Contains(buf.String(), "/agent/a_1") {
		t.Fatalf("refused diagnostic text reached the log: %s", buf.String())
	}

	for i := 0; i < browserDiagPerMinute+5; i++ {
		postBrowserDiagnostic(t, h, `{"reason":"error","route_kind":"dashboard","mount_state":"mounted","visit_id":"abc123def456"}`)
	}
	if got := len(browserDiagnosticLines(&buf)); got != browserDiagPerMinute {
		t.Fatalf("diagnostic records after a burst = %d, want the per-minute cap %d", got, browserDiagPerMinute)
	}
}
