package server

import (
	"encoding/json"
	"net/http"
	"regexp"
	"sync"
	"time"
)

// Browser diagnostics carry only allowlisted words: no console text, path,
// stack or page content ever reaches the log (TS-03.R92, INV §22).
var (
	browserDiagReasons = map[string]bool{"error": true, "unhandled_rejection": true, "render_error": true, "shell_missing": true}
	browserDiagRoutes  = map[string]bool{
		"dashboard": true, "project": true, "agent": true, "think_tank": true, "archive": true,
		"pipelines": true, "tasks": true, "settings": true, "other": true,
	}
	browserDiagMounts = map[string]bool{"mounted": true, "missing": true}
	browserDiagVisit  = regexp.MustCompile(`^[a-z0-9]{8,32}$`)
)

// browserDiagPerMinute bounds what a misbehaving page can write to the log.
const browserDiagPerMinute = 30

type browserDiagLimiter struct {
	mu     sync.Mutex
	window time.Time
	count  int
}

func (l *browserDiagLimiter) allow(now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if now.Sub(l.window) >= time.Minute {
		l.window, l.count = now, 0
	}
	l.count++
	return l.count <= browserDiagPerMinute
}

type browserDiagnostic struct {
	Reason     string `json:"reason"`
	RouteKind  string `json:"route_kind"`
	MountState string `json:"mount_state"`
	VisitID    string `json:"visit_id"`
}

// handleBrowserDiagnostic records one desktop browser failure in the dashboard
// log (TS-01.R15). Desktop-only: the phone allowlist does not name it.
func (s *Server) handleBrowserDiagnostic(w http.ResponseWriter, r *http.Request) {
	var body browserDiagnostic
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<10)).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid diagnostic")
		return
	}
	if !browserDiagReasons[body.Reason] || !browserDiagRoutes[body.RouteKind] || !browserDiagMounts[body.MountState] || !browserDiagVisit.MatchString(body.VisitID) {
		writeError(w, http.StatusBadRequest, "invalid diagnostic")
		return
	}
	if s.browserDiag.allow(time.Now()) {
		s.log.Warn("browser diagnostic", "reason", body.Reason, "route_kind", body.RouteKind, "mount_state", body.MountState, "visit_id", body.VisitID)
	}
	w.WriteHeader(http.StatusNoContent)
}
