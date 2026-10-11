package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
)

const (
	quotaControlRequestLimit  = 1 << 20
	quotaControlResponseLimit = 2 << 20
)

var errQuotaControlBodyTooLarge = fmt.Errorf("quota control request body exceeds limit")

// quotaControlResponse buffers the small JSON response until its owner mutation
// and the captured quota CAS have both completed.
type quotaControlResponse struct {
	header   http.Header
	status   int
	body     bytes.Buffer
	tooLarge bool
}

func newQuotaControlResponse() *quotaControlResponse {
	return &quotaControlResponse{header: make(http.Header)}
}

func (w *quotaControlResponse) Header() http.Header { return w.header }
func (w *quotaControlResponse) WriteHeader(code int) {
	if w.status == 0 {
		w.status = code
	}
}
func (w *quotaControlResponse) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	if w.body.Len()+len(p) > quotaControlResponseLimit {
		w.tooLarge = true
		return len(p), nil
	}
	return w.body.Write(p)
}
func (w *quotaControlResponse) flush(dst http.ResponseWriter) {
	for key, values := range w.header {
		for _, value := range values {
			dst.Header().Add(key, value)
		}
	}
	status := w.status
	if status == 0 {
		status = http.StatusOK
	}
	dst.WriteHeader(status)
	_, _ = dst.Write(w.body.Bytes())
}

type quotaControlCapture func(*http.Request) ([]state.QuotaControlSnapshot, error)

func (s *Server) quotaControlHandler(capture quotaControlCapture, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.quotaControlMu.Lock()
		defer s.quotaControlMu.Unlock()
		snapshots, err := capture(r)
		if err != nil {
			if err == errQuotaControlBodyTooLarge {
				writeAPIError(w, apiError(runtime.CodeValidation, "request body exceeds size limit"))
				return
			}
			writeAPIError(w, apiError(runtime.CodeInternal, "quota owner could not be read"))
			return
		}
		buffer := newQuotaControlResponse()
		next.ServeHTTP(buffer, r)
		if buffer.status >= 200 && buffer.status < 300 {
			if err := s.cancelQuotaSnapshots(snapshots); err != nil {
				writeAPIError(w, apiError(runtime.CodeInternal, fmt.Sprintf("action applied but quota schedule could not be invalidated: %v", err)))
				return
			}
			if buffer.tooLarge {
				writeAPIError(w, apiError(runtime.CodeInternal, "action applied but response exceeded the buffered response limit"))
				return
			}
		}
		buffer.flush(w)
	})
}

func (s *Server) quotaAgentCapture(r *http.Request) ([]state.QuotaControlSnapshot, error) {
	snapshot, err := s.stateStore.SnapshotAgentQuota(r.PathValue("id"))
	if err != nil || snapshot == nil {
		return nil, err
	}
	return []state.QuotaControlSnapshot{*snapshot}, nil
}

func (s *Server) quotaProjectCapture(r *http.Request) ([]state.QuotaControlSnapshot, error) {
	agents, err := s.stateStore.ListAgents()
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, agent := range agents {
		if agent.Project == r.PathValue("project") {
			ids = append(ids, agent.AgentID)
		}
	}
	return s.captureQuotaSnapshots(ids)
}

func (s *Server) quotaOwnerCapture(owner func(*http.Request) state.QuotaOwner) quotaControlCapture {
	return func(r *http.Request) ([]state.QuotaControlSnapshot, error) {
		return s.stateStore.SnapshotOwnerQuotas(owner(r))
	}
}

func (s *Server) captureQuotaSnapshots(agentIDs []string) ([]state.QuotaControlSnapshot, error) {
	var snapshots []state.QuotaControlSnapshot
	for _, agentID := range agentIDs {
		snapshot, err := s.stateStore.SnapshotAgentQuota(agentID)
		if err != nil {
			return nil, err
		}
		if snapshot != nil {
			snapshots = append(snapshots, *snapshot)
		}
	}
	return snapshots, nil
}

func (s *Server) cancelQuotaSnapshots(snapshots []state.QuotaControlSnapshot) error {
	changed, err := s.stateStore.CancelQuotaSnapshots(snapshots)
	if err != nil {
		return err
	}
	return s.publishQuotaAgents(changed)
}

func (s *Server) publishQuotaAgents(ids []string) error {
	var failures []error
	for _, id := range ids {
		if _, err := s.stateMgr.Touch(id); err != nil {
			failures = append(failures, fmt.Errorf("publish %s: %w", id, err))
		}
	}
	return errors.Join(failures...)
}

func (s *Server) wrapQuotaControl(pattern string, next http.Handler) http.Handler {
	var capture quotaControlCapture
	switch pattern {
	case "POST /api/sessions/{id}/prompt", "POST /api/sessions/{id}/steer",
		"POST /api/sessions/{id}/resume", "POST /api/sessions/{id}/switch-runtime", "POST /api/sessions/{id}/archive":
		capture = s.quotaAgentCapture
	case "POST /api/sessions/{id}/cancel":
		capture = func(r *http.Request) ([]state.QuotaControlSnapshot, error) {
			body, err := io.ReadAll(io.LimitReader(r.Body, quotaControlRequestLimit+1))
			if err != nil {
				return nil, err
			}
			if len(body) > quotaControlRequestLimit {
				return nil, errQuotaControlBodyTooLarge
			}
			r.Body = io.NopCloser(bytes.NewReader(body))
			var request struct {
				Target string `json:"target"`
			}
			if err := json.Unmarshal(body, &request); err != nil || request.Target != "turn" {
				return nil, nil
			}
			return s.quotaAgentCapture(r)
		}
	case "POST /api/projects/{project}/archive":
		capture = s.quotaProjectCapture
	case "DELETE /api/tasks/{id}", "POST /api/tasks/{id}/cancel", "POST /api/tasks/{id}/result", "POST /api/tasks/{id}/retry", "POST /api/tasks/{id}/rearm":
		capture = s.quotaOwnerCapture(func(r *http.Request) state.QuotaOwner {
			return state.QuotaOwner{Kind: "task", TaskID: r.PathValue("id")}
		})
	case "DELETE /api/pipeline-runs/{id}", "POST /api/pipeline-runs/{id}/stop", "POST /api/pipeline-runs/{id}/continue",
		"POST /api/pipeline-runs/{id}/retry", "POST /api/pipeline-runs/{id}/replace":
		capture = func(r *http.Request) ([]state.QuotaControlSnapshot, error) {
			return s.stateStore.SnapshotQuotaOwnerID("pipeline", "run_id", r.PathValue("id"))
		}
	case "DELETE /api/think-tanks/{id}", "POST /api/think-tanks/{id}/pause",
		"POST /api/think-tanks/{id}/resume", "POST /api/think-tanks/{id}/end", "POST /api/think-tanks/{id}/retry":
		capture = func(r *http.Request) ([]state.QuotaControlSnapshot, error) {
			return s.stateStore.SnapshotQuotaOwnerID("room", "room_id", r.PathValue("id"))
		}
	default:
		return next
	}
	return s.quotaControlHandler(capture, next)
}
