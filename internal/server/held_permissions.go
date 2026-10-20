package server

import (
	"cmp"
	"encoding/json"
	"slices"
	"strings"

	"github.com/AsaphNoam/Chuck/internal/runtime"
)

// A permission request whose transcript record did not commit is still pending
// in the runtime, so its approval controls must stay reachable without
// history (FS-03.R99, TS-03.R73). The server holds the request here as derived,
// never-persisted live state. It is read only while the live generation still
// awaits it, so it is bounded by that generation's pending set; resolution and
// generation exit drop it.

func (s *Server) holdUnretainedPermission(ev runtime.Event) {
	if ev.Type != runtime.EvPermissionRequest {
		return
	}
	var data runtime.PermissionRequestData
	if json.Unmarshal(ev.Data, &data) != nil || data.ToolCallID == "" || data.AutoApproved {
		return
	}
	s.permissionMu.Lock()
	s.heldPermissions[permissionToolKey(ev.AgentID, ev.Generation, data.ToolCallID)] = ev
	s.permissionMu.Unlock()
}

func (s *Server) dropUnretainedPermission(ev runtime.Event) {
	var data runtime.PermissionResolvedData
	if json.Unmarshal(ev.Data, &data) != nil {
		return
	}
	s.permissionMu.Lock()
	delete(s.heldPermissions, permissionToolKey(ev.AgentID, ev.Generation, data.ToolCallID))
	s.permissionMu.Unlock()
}

// unretainedPermissions returns the agent's held requests that its current
// generation still awaits, oldest first. Any other held entry is stale and is
// dropped.
func (s *Server) unretainedPermissions(agentID string) []runtime.Event {
	if s.registry == nil {
		return nil
	}
	chat := s.registry.Chat()
	generation := s.registry.Generation(agentID)
	prefix := agentID + "\x00"
	type held struct {
		key, toolCallID string
		ev              runtime.Event
	}
	var candidates []held
	s.permissionMu.Lock()
	for key, ev := range s.heldPermissions {
		if strings.HasPrefix(key, prefix) {
			candidates = append(candidates, held{key: key, toolCallID: key[strings.LastIndexByte(key, 0)+1:], ev: ev})
		}
	}
	s.permissionMu.Unlock()
	var out []runtime.Event
	var stale []string
	for _, c := range candidates {
		if chat != nil && generation != "" && c.ev.Generation == generation && chat.PermissionPending(agentID, c.toolCallID) {
			out = append(out, c.ev)
		} else {
			stale = append(stale, c.key)
		}
	}
	if len(stale) > 0 {
		s.permissionMu.Lock()
		for _, key := range stale {
			delete(s.heldPermissions, key)
		}
		s.permissionMu.Unlock()
	}
	slices.SortFunc(out, func(a, b runtime.Event) int { return cmp.Compare(a.Seq, b.Seq) })
	return out
}
