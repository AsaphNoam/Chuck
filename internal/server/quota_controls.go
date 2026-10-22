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

// Manual controls reserve only their target agents/owner. The policy mutex is
// reserved for global disable versus automatic admission, never manual startup.
// Keys live only while an action is in flight, and a set is claimed atomically.
func (s *Server) claimQuotaActions(keys []string) (func(), bool) {
	s.quotaActionMu.Lock()
	defer s.quotaActionMu.Unlock()
	for _, key := range keys {
		if s.quotaActions[key] {
			return nil, false
		}
	}
	if s.quotaActions == nil {
		s.quotaActions = make(map[string]bool)
	}
	for _, key := range keys {
		s.quotaActions[key] = true
	}
	return func() {
		s.quotaActionMu.Lock()
		defer s.quotaActionMu.Unlock()
		for _, key := range keys {
			delete(s.quotaActions, key)
		}
	}, true
}

func quotaAgentKeys(ids []string) []string {
	keys := make([]string, 0, len(ids))
	for _, id := range ids {
		keys = append(keys, "agent:"+id)
	}
	return keys
}

func quotaOwnerKeys(owner state.QuotaOwner) []string {
	var keys []string
	if owner.TaskID != "" {
		keys = append(keys, "task:"+owner.TaskID)
	}
	if owner.RunID != "" {
		keys = append(keys, "pipeline:"+owner.RunID)
	}
	if owner.RoomID != "" {
		keys = append(keys, "room:"+owner.RoomID)
	}
	return keys
}

func (s *Server) quotaAdmissionKeys(q state.QuotaIntent) ([]string, error) {
	keys := append(quotaAgentKeys([]string{q.AgentID}), quotaOwnerKeys(q.Owner)...)
	// A pipeline Stop also owns its borrowed tasks and room-backed stage.
	// Include that durable parent even before it has a cancellable snapshot.
	if q.Owner.Kind == "task" {
		lineage, err := s.stateStore.ReadTaskLineage(q.Owner.TaskID)
		if err != nil && !errors.Is(err, state.ErrNotFound) {
			return nil, err
		}
		if lineage.PipelineRunID != "" {
			keys = append(keys, "pipeline:"+lineage.PipelineRunID)
		}
	}
	if q.Owner.Kind == "room" {
		detail, err := s.stateStore.ReadThinkTank(q.Owner.RoomID)
		if err != nil {
			return nil, err
		}
		if detail.Room.PipelineRunID != "" {
			keys = append(keys, "pipeline:"+detail.Room.PipelineRunID)
		}
	}
	return keys, nil
}

func (s *Server) quotaControlHandler(capture quotaControlCapture, next http.Handler, ownerScope ...string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := "agent:" + r.PathValue("id")
		if len(ownerScope) > 0 {
			key = ownerScope[0] + ":" + r.PathValue("id")
			if ownerScope[0] == "project" {
				key = "project:" + r.PathValue("project")
			}
		}
		release, ok := s.claimQuotaActions([]string{key})
		if !ok {
			writeAPIError(w, apiError(runtime.CodeConflict, "an action on this agent or owner is already in progress"))
			return
		}
		defer release()
		snapshots, err := capture(r)
		if err != nil {
			if err == errQuotaControlBodyTooLarge {
				writeAPIError(w, apiError(runtime.CodeValidation, "request body exceeds size limit"))
				return
			}
			writeAPIError(w, apiError(runtime.CodeInternal, "quota owner could not be read"))
			return
		}
		var keys []string
		reserved := map[string]bool{key: true}
		for _, snapshot := range snapshots {
			if agentKey := "agent:" + snapshot.AgentID; agentKey != key {
				keys = append(keys, agentKey)
				reserved[agentKey] = true
			}
		}
		releaseAgents, ok := s.claimQuotaActions(keys)
		if !ok {
			writeAPIError(w, apiError(runtime.CodeConflict, "an action on this agent or owner is already in progress"))
			return
		}
		defer releaseAgents()
		// An agent action may have completed while an owner captured its set.
		// Re-read the episodes after all target reservations are held.
		snapshots, err = capture(r)
		if err != nil {
			writeAPIError(w, apiError(runtime.CodeInternal, "quota owner could not be read"))
			return
		}
		for _, snapshot := range snapshots {
			if !reserved["agent:"+snapshot.AgentID] {
				writeAPIError(w, apiError(runtime.CodeConflict, "quota owner changed; refresh and try again"))
				return
			}
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
	for _, snapshot := range snapshots {
		for _, id := range changed {
			if snapshot.AgentID == id {
				s.registry.ClearQuotaBlock(id, snapshot.SourceGeneration, snapshot.ExecutingTurn)
			}
		}
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
	var scope []string
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
		scope = []string{"project"}
	case "DELETE /api/tasks/{id}", "POST /api/tasks/{id}/cancel", "POST /api/tasks/{id}/result", "POST /api/tasks/{id}/retry", "POST /api/tasks/{id}/rearm":
		scope = []string{"task"}
		capture = s.quotaOwnerCapture(func(r *http.Request) state.QuotaOwner {
			return state.QuotaOwner{Kind: "task", TaskID: r.PathValue("id")}
		})
	case "DELETE /api/pipeline-runs/{id}", "POST /api/pipeline-runs/{id}/stop", "POST /api/pipeline-runs/{id}/continue",
		"POST /api/pipeline-runs/{id}/retry", "POST /api/pipeline-runs/{id}/replace":
		scope = []string{"pipeline"}
		capture = func(r *http.Request) ([]state.QuotaControlSnapshot, error) {
			return s.stateStore.SnapshotQuotaOwnerID("pipeline", "run_id", r.PathValue("id"))
		}
	case "DELETE /api/think-tanks/{id}", "POST /api/think-tanks/{id}/pause",
		"POST /api/think-tanks/{id}/resume", "POST /api/think-tanks/{id}/end", "POST /api/think-tanks/{id}/retry":
		scope = []string{"room"}
		capture = func(r *http.Request) ([]state.QuotaControlSnapshot, error) {
			return s.stateStore.SnapshotQuotaOwnerID("room", "room_id", r.PathValue("id"))
		}
	default:
		return next
	}
	return s.quotaControlHandler(capture, next, scope...)
}
