package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"unicode/utf8"
)

// Bounds on a provider's mode advertisement (INV §8/§16): the list and its copy
// are rendered as header controls, so they are clamped where they enter.
const (
	maxSessionModes       = 16
	maxSessionModeIDBytes = 64
	maxSessionModeName    = 40
	maxSessionModeDesc    = 160
)

// SessionModeOption is one provider-advertised permission mode.
type SessionModeOption struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// SessionModes is the live session's permission-mode advertisement
// (FS-03.R88, TS-04.R92). It is live state, never persisted: every launch,
// resume or fork decodes it again from the new session, so a stale choice can
// never outlive the process that honored it (INV §1, §3).
type SessionModes struct {
	Current string              `json:"current"`
	Options []SessionModeOption `json:"options"`
	// configID names the session config option to set; empty means the legacy
	// ACP `modes` field, set through session/set_mode.
	configID string
}

func (m SessionModes) offers(id string) bool {
	for _, option := range m.Options {
		if option.ID == id {
			return true
		}
	}
	return false
}

func clampRunes(text string, limit int) string {
	if utf8.RuneCountInString(text) <= limit {
		return text
	}
	runes := []rune(text)
	return string(runes[:limit-1]) + "…"
}

func appendModeOption(out []SessionModeOption, id, name, description string) []SessionModeOption {
	if id == "" || len(id) > maxSessionModeIDBytes || len(out) >= maxSessionModes {
		return out
	}
	if name == "" {
		name = id
	}
	return append(out, SessionModeOption{ID: id, Name: clampRunes(name, maxSessionModeName), Description: clampRunes(description, maxSessionModeDesc)})
}

// decodeSessionModes reads a permission-mode advertisement from a session
// response or a config_option_update. A config option in ACP's "mode" category
// wins; the older `modes` field is the fallback. ok is false when neither is
// present, so a caller keeps what it had rather than clearing it (INV §7).
func decodeSessionModes(raw json.RawMessage) (SessionModes, bool) {
	type entry struct {
		Value       string  `json:"value"`
		Name        string  `json:"name"`
		Description string  `json:"description"`
		Options     []entry `json:"options"`
	}
	var response struct {
		ConfigOptions []struct {
			ID           string          `json:"id"`
			ConfigID     string          `json:"configId"`
			Category     string          `json:"category"`
			CurrentValue json.RawMessage `json:"currentValue"`
			Options      []entry         `json:"options"`
		} `json:"configOptions"`
		Modes *struct {
			CurrentModeID  string `json:"currentModeId"`
			AvailableModes []struct {
				ID          string `json:"id"`
				Name        string `json:"name"`
				Description string `json:"description"`
			} `json:"availableModes"`
		} `json:"modes"`
	}
	if json.Unmarshal(raw, &response) != nil {
		return SessionModes{}, false
	}
	for _, option := range response.ConfigOptions {
		if option.Category != "mode" {
			continue
		}
		id := option.ID
		if id == "" {
			id = option.ConfigID
		}
		out := SessionModes{Current: normalizeConfigValue(option.CurrentValue), Options: []SessionModeOption{}, configID: id}
		for _, item := range option.Options {
			if len(item.Options) == 0 {
				out.Options = appendModeOption(out.Options, item.Value, item.Name, item.Description)
				continue
			}
			for _, nested := range item.Options {
				out.Options = appendModeOption(out.Options, nested.Value, nested.Name, nested.Description)
			}
		}
		if id != "" && len(out.Options) > 0 {
			return out, true
		}
	}
	if response.Modes == nil {
		return SessionModes{}, false
	}
	out := SessionModes{Current: response.Modes.CurrentModeID, Options: []SessionModeOption{}}
	for _, mode := range response.Modes.AvailableModes {
		out.Options = appendModeOption(out.Options, mode.ID, mode.Name, mode.Description)
	}
	return out, len(out.Options) > 0
}

// decodeModeUpdate recognizes the two root session/update kinds that move the
// mode: a full config_option_update, or a current_mode_update naming the new id.
func decodeModeUpdate(params json.RawMessage) (modes SessionModes, current string, ok bool) {
	var su struct {
		Update json.RawMessage `json:"update"`
	}
	if json.Unmarshal(params, &su) != nil {
		return SessionModes{}, "", false
	}
	var kind struct {
		SessionUpdate string `json:"sessionUpdate"`
		CurrentModeID string `json:"currentModeId"`
	}
	if json.Unmarshal(su.Update, &kind) != nil {
		return SessionModes{}, "", false
	}
	switch kind.SessionUpdate {
	case "current_mode_update":
		return SessionModes{}, kind.CurrentModeID, kind.CurrentModeID != ""
	case "config_option_update":
		modes, found := decodeSessionModes(su.Update)
		return modes, "", found
	}
	return SessionModes{}, "", false
}

// setModes replaces the advertisement. It has its own lock because the
// transport delivers notifications on its read goroutine, which must never wait
// on mu while a request holding mu waits for that goroutine's response.
func (as *agentState) setModes(modes SessionModes) {
	as.modeMu.Lock()
	as.modes = modes
	as.modeMu.Unlock()
}

func (as *agentState) setCurrentMode(id string) {
	as.modeMu.Lock()
	if as.modes.offers(id) {
		as.modes.Current = id
	}
	as.modeMu.Unlock()
}

func (as *agentState) currentModes() SessionModes {
	as.modeMu.Lock()
	defer as.modeMu.Unlock()
	out := as.modes
	out.Options = append([]SessionModeOption{}, as.modes.Options...)
	return out
}

// onModeUpdate consumes a root mode update; true means the frame was handled.
func (as *agentState) onModeUpdate(params json.RawMessage) bool {
	modes, current, ok := decodeModeUpdate(params)
	if !ok {
		return false
	}
	if current != "" {
		as.setCurrentMode(current)
	} else {
		as.setModes(modes)
	}
	return true
}

// SessionModes returns the live permission-mode advertisement. A session that
// advertises none returns an empty option list.
func (c *ChatRuntime) SessionModes(agentID string) (SessionModes, error) {
	as, err := c.lookup(agentID)
	if err != nil {
		return SessionModes{}, err
	}
	return as.currentModes(), nil
}

// SetSessionMode switches a live session to one of its advertised permission
// modes without restarting it. Callers serialize changes with the per-agent
// lifecycle claim; mu is held only to read the session identity, never across
// the call (see setModes).
func (c *ChatRuntime) SetSessionMode(ctx context.Context, agentID, id string) (SessionModes, error) {
	as, err := c.lookup(agentID)
	if err != nil {
		return SessionModes{}, err
	}
	modes := as.currentModes()
	if len(modes.Options) == 0 {
		return modes, fmt.Errorf("%w: permission mode", ErrSettingUnsupported)
	}
	if !modes.offers(id) {
		return modes, fmt.Errorf("%w: mode %q is not offered by this session", ErrSettingRejected, id)
	}
	as.mu.Lock()
	transport, sessionID := as.transport, as.sessionID
	as.mu.Unlock()
	var result json.RawMessage
	if modes.configID != "" {
		result, err = transport.Call(ctx, "session/set_config_option", map[string]any{"sessionId": sessionID, "configId": modes.configID, "value": id})
	} else {
		result, err = transport.Call(ctx, "session/set_mode", map[string]any{"sessionId": sessionID, "modeId": id})
	}
	if err != nil {
		return as.currentModes(), fmt.Errorf("%w: mode %q: %s", ErrSettingRejected, id, err)
	}
	if reported, ok := decodeSessionModes(result); ok {
		as.setModes(reported)
	} else {
		as.setCurrentMode(id)
	}
	after := as.currentModes()
	if after.Current != "" && after.Current != id {
		return after, fmt.Errorf("%w: mode is %q after requesting %q", ErrSettingIgnored, after.Current, id)
	}
	return after, nil
}
