package runtime

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/AsaphNoam/Chuck/internal/backend"
)

// FS-09.A17 / FS-18.A13: adding Chuck guidance must preserve native setup.
// PROMPT-1: Codex 0.159.3 request capture proves that a thread/start config
// developer_instructions value replaces the native config.toml value. Unlike
// TestCodexDeveloperInstructionsEnv, the prior instruction originates in the
// mirrored native config, not an existing CODEX_CONFIG environment overlay.
func TestCodexNativeDeveloperInstructionsPreservedInvestigation(t *testing.T) {
	t.Skip("PROMPT-1: native config.toml developer instructions are overwritten; unskip with the fix")

	profile := t.TempDir()
	const native = "Keep all changes compatible with the supported production schema."
	if err := os.WriteFile(filepath.Join(profile, "config.toml"),
		[]byte("developer_instructions = "+strconv.Quote(native)+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	ad, ok := backend.For("codex-acp")
	if !ok {
		t.Fatal("codex adapter missing")
	}
	spec := LaunchSpec{
		Cwd:          profile,
		BackendType:  "codex-acp",
		SystemPrompt: "Chuck project and persona guidance.",
		Env:          []string{"CODEX_HOME=" + profile},
	}
	cmd, err := NewChatRuntime(nil).spawnCmd(ad, spec)
	if err != nil {
		t.Fatal(err)
	}
	var overlay struct {
		DeveloperInstructions string `json:"developer_instructions"`
	}
	if err := json.Unmarshal([]byte(envValue(cmd.Env, "CODEX_CONFIG")), &overlay); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{native, spec.StartSystemPrompt()} {
		if !strings.Contains(overlay.DeveloperInstructions, want) {
			t.Errorf("effective developer instruction override lost %q", want)
		}
	}
}
