package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// FS-03.R88, TS-04.R92 — a "mode"-category option wins over the legacy field,
// grouped entries flatten, and provider copy is bounded where it enters.
func TestDecodeSessionModes(t *testing.T) {
	got, ok := decodeSessionModes(json.RawMessage(`{"configOptions":[
		{"id":"model","type":"select","currentValue":"m","options":[{"value":"m"}]},
		{"id":"mode","category":"mode","type":"select","currentValue":"plan","options":[
			{"value":"default","name":"Manual","description":"Prompts"},
			{"name":"More","options":[{"value":"plan","name":"Plan Mode"}]}]}],
		"modes":{"currentModeId":"legacy","availableModes":[{"id":"legacy","name":"Legacy"}]}}`))
	if !ok || got.configID != "mode" || got.Current != "plan" || len(got.Options) != 2 || got.Options[1].ID != "plan" || got.Options[0].Name != "Manual" {
		t.Fatalf("config-option modes = %+v ok=%v", got, ok)
	}

	legacy, ok := decodeSessionModes(json.RawMessage(`{"modes":{"currentModeId":"auto","availableModes":[{"id":"read-only","name":"Read Only"},{"id":"auto"}]}}`))
	if !ok || legacy.configID != "" || legacy.Current != "auto" || len(legacy.Options) != 2 || legacy.Options[1].Name != "auto" {
		t.Fatalf("legacy modes = %+v ok=%v", legacy, ok)
	}

	var many []string
	for i := 0; i < maxSessionModes+4; i++ {
		many = append(many, fmt.Sprintf(`{"id":"m%d","name":%q}`, i, strings.Repeat("n", 100)))
	}
	bounded, ok := decodeSessionModes(json.RawMessage(`{"modes":{"currentModeId":"m0","availableModes":[` + strings.Join(many, ",") + `]}}`))
	if !ok || len(bounded.Options) != maxSessionModes || len([]rune(bounded.Options[0].Name)) != maxSessionModeName {
		t.Fatalf("bounded = %d options, name %d runes", len(bounded.Options), len([]rune(bounded.Options[0].Name)))
	}

	for _, raw := range []string{`{}`, `{"configOptions":[]}`, `{"modes":{"availableModes":[]}}`, `not json`} {
		if _, ok := decodeSessionModes(json.RawMessage(raw)); ok {
			t.Fatalf("decodeSessionModes(%s) ok = true", raw)
		}
	}
}

// A provider-initiated mode change (Claude leaving plan mode) is live state:
// it updates the snapshot, and an id the session never offered is ignored.
func TestModeUpdatesReplaceLiveState(t *testing.T) {
	c := NewChatRuntime(nil)
	as := &agentState{agentID: "a1", journal: NewJournal("a1", nil)}
	c.agents["a1"] = as
	update := func(body string) {
		c.onNotification(as, "session/update", json.RawMessage(`{"sessionId":"","update":`+body+`}`))
	}
	update(`{"sessionUpdate":"config_option_update","configOptions":[{"id":"mode","category":"mode","currentValue":"plan","options":[{"value":"default"},{"value":"plan"}]}]}`)
	update(`{"sessionUpdate":"current_mode_update","currentModeId":"default"}`)
	update(`{"sessionUpdate":"current_mode_update","currentModeId":"invented"}`)
	got, err := c.SessionModes("a1")
	if err != nil || got.Current != "default" || len(got.Options) != 2 {
		t.Fatalf("modes = %+v err=%v", got, err)
	}
	if _, err := c.SessionModes("nope"); !errors.Is(err, ErrNoHandle) {
		t.Fatalf("unknown agent err = %v", err)
	}
}

// FS-03.A67 — against the fake adapter's real wire shape: the advertised modes
// arrive at launch, a change goes through set_config_option while the adapter
// announces it before answering, and an unoffered mode never reaches the wire.
func TestSetSessionModeAppliesAdvertisedMode(t *testing.T) {
	c, spec := newChatTest(t, "stream_text")
	logPath := filepath.Join(t.TempDir(), "config.ndjson")
	spec.Env = append(spec.Env, "FAKEACP_MODE_OPTIONS=default,acceptEdits,plan", "FAKEACP_CONFIG_LOG="+logPath)
	h, err := c.Start(context.Background(), spec)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { c.Stop(context.Background(), h.AgentID) })

	modes, err := c.SessionModes(h.AgentID)
	if err != nil || modes.Current != "default" || len(modes.Options) != 3 || modes.Options[2].Name != "Mode plan" {
		t.Fatalf("launch modes = %+v err=%v", modes, err)
	}
	modes, err = c.SetSessionMode(context.Background(), h.AgentID, "plan")
	if err != nil || modes.Current != "plan" {
		t.Fatalf("SetSessionMode(plan) = %+v err=%v", modes, err)
	}
	if _, err := c.SetSessionMode(context.Background(), h.AgentID, "bypassPermissions"); !errors.Is(err, ErrSettingRejected) {
		t.Fatalf("unoffered mode err = %v, want ErrSettingRejected", err)
	}
	raw, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "bypassPermissions") || !strings.Contains(string(raw), `{"configId":"mode","sessionId":"fake-sess-1","value":"plan"}`) {
		t.Fatalf("config calls = %s", raw)
	}
}

// An empty acknowledgement must not erase a provider report, even when that
// report repeats the pre-call value (FS-03.A67, TS-04.R92).
func TestSetSessionModePreservesProviderReport(t *testing.T) {
	for _, tc := range []struct {
		name, reported, want string
		legacy, ignored      bool
	}{
		{"legacy unchanged", "default", "default", true, true},
		{"legacy changed", "acceptEdits", "acceptEdits", true, true},
		{"legacy accepted notification", "plan", "plan", true, false},
		{"legacy no notification", "", "plan", true, false},
		{"config unchanged", "default", "default", false, true},
		{"config changed", "acceptEdits", "acceptEdits", false, true},
		{"config accepted", "", "plan", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, spec := newChatTest(t, "stream_text")
			spec.Env = append(spec.Env, "FAKEACP_MODE_OPTIONS=default,acceptEdits,plan", "FAKEACP_MODE_REPORTED="+tc.reported)
			if tc.legacy {
				spec.Env = append(spec.Env, "FAKEACP_LEGACY_MODES=1")
			}
			h, err := c.Start(context.Background(), spec)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { c.Stop(context.Background(), h.AgentID) })
			got, err := c.SetSessionMode(context.Background(), h.AgentID, "plan")
			if got.Current != tc.want || errors.Is(err, ErrSettingIgnored) != tc.ignored || (!tc.ignored && err != nil) {
				t.Fatalf("mode = %q, err = %v; want %q, ignored=%v", got.Current, err, tc.want, tc.ignored)
			}
			live, err := c.SessionModes(h.AgentID)
			if err != nil || live.Current != tc.want {
				t.Fatalf("live = %+v, err=%v", live, err)
			}
		})
	}
}

func TestSetSessionModeWithoutAdvertisementIsUnsupported(t *testing.T) {
	c, spec := newChatTest(t, "stream_text")
	h, err := c.Start(context.Background(), spec)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { c.Stop(context.Background(), h.AgentID) })
	modes, err := c.SessionModes(h.AgentID)
	if err != nil || modes.Options == nil || len(modes.Options) != 0 {
		t.Fatalf("modes = %#v err=%v; want a non-nil empty list", modes, err)
	}
	if _, err := c.SetSessionMode(context.Background(), h.AgentID, "plan"); !errors.Is(err, ErrSettingUnsupported) {
		t.Fatalf("err = %v, want ErrSettingUnsupported", err)
	}
}
