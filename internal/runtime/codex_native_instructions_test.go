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

// FS-09.R32/A17, TS-04.R14: Codex request capture proves that the overlay's
// developer_instructions replaces the native config.toml value, so adding Chuck
// guidance must carry the native value the child would otherwise load.

func writeCodexTOML(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func codexOverlayInstructions(t *testing.T, env []string) (string, bool) {
	t.Helper()
	raw := envValue(env, "CODEX_CONFIG")
	if raw == "" {
		return "", false
	}
	var overlay map[string]any
	if err := json.Unmarshal([]byte(raw), &overlay); err != nil {
		t.Fatalf("CODEX_CONFIG is not JSON: %v", err)
	}
	value, ok := overlay["developer_instructions"].(string)
	return value, ok
}

func TestCodexNativeDeveloperInstructionsPreserved(t *testing.T) {
	const native = "Keep all changes compatible with the supported production schema."
	const prompt = "Chuck project and persona guidance."
	newSpec := func(t *testing.T, config string) (LaunchSpec, string) {
		profile := t.TempDir()
		if config != "" {
			writeCodexTOML(t, filepath.Join(profile, "config.toml"), config)
		}
		return LaunchSpec{
			Cwd:          t.TempDir(),
			BackendType:  "codex-acp",
			SystemPrompt: prompt,
			Env:          []string{"CODEX_HOME=" + profile},
		}, profile
	}
	launch := func(t *testing.T, spec LaunchSpec) []string {
		t.Helper()
		env, err := codexLaunchEnv(spec.Env, spec)
		if err != nil {
			t.Fatalf("codexLaunchEnv: %v", err)
		}
		return env
	}

	t.Run("user config", func(t *testing.T) {
		spec, _ := newSpec(t, "developer_instructions = "+strconv.Quote(native)+"\n")
		got, _ := codexOverlayInstructions(t, launch(t, spec))
		if want := native + "\n\n" + prompt; got != want {
			t.Fatalf("developer_instructions = %q, want %q", got, want)
		}
		resume := spec
		resume.RuntimeSystemPrompt = prompt + "\n\nPrior-turn primer."
		got, _ = codexOverlayInstructions(t, launch(t, resume))
		if want := native + "\n\n" + resume.StartSystemPrompt(); got != want {
			t.Fatalf("resume developer_instructions = %q, want %q", got, want)
		}
	})

	t.Run("active profile then project override", func(t *testing.T) {
		spec, _ := newSpec(t, "developer_instructions = \"user\"\nprofile = \"work\"\n"+
			"[profiles.work]\ndeveloper_instructions = \"profile\"\n")
		got, _ := codexOverlayInstructions(t, launch(t, spec))
		if want := "profile\n\n" + prompt; got != want {
			t.Fatalf("profile developer_instructions = %q, want %q", got, want)
		}
		writeCodexTOML(t, filepath.Join(spec.Cwd, ".codex", "config.toml"), "developer_instructions = \"project\"\n")
		got, _ = codexOverlayInstructions(t, launch(t, spec))
		if want := "project\n\n" + prompt; got != want {
			t.Fatalf("project developer_instructions = %q, want %q", got, want)
		}
	})

	t.Run("environment overlay takes precedence", func(t *testing.T) {
		spec, _ := newSpec(t, "developer_instructions = "+strconv.Quote(native)+"\n")
		spec.Env = append(spec.Env, `CODEX_CONFIG={"developer_instructions":"Overlay."}`)
		got, _ := codexOverlayInstructions(t, launch(t, spec))
		if want := "Overlay.\n\n" + prompt; got != want {
			t.Fatalf("developer_instructions = %q, want %q", got, want)
		}
	})

	t.Run("empty prompt leaves native config alone", func(t *testing.T) {
		spec, _ := newSpec(t, "developer_instructions = "+strconv.Quote(native)+"\n")
		spec.SystemPrompt = ""
		if _, ok := codexOverlayInstructions(t, launch(t, spec)); ok {
			t.Fatal("empty prompt added an overriding developer_instructions")
		}
	})

	t.Run("malformed native config fails closed", func(t *testing.T) {
		for name, config := range map[string]string{
			"malformed TOML":   "developer_instructions = \n",
			"non-string value": "developer_instructions = 3\n",
		} {
			spec, _ := newSpec(t, config)
			if _, err := codexLaunchEnv(spec.Env, spec); err == nil {
				t.Fatalf("%s: launch env succeeded", name)
			}
		}
	})
}

// The merge must read the profile generation published for this child, not
// the stale mirror present when the command was composed.
func TestCodexStartUsesRefreshedNativeInstructions(t *testing.T) {
	personal := filepath.Join(t.TempDir(), "codex")
	writeCodexTOML(t, filepath.Join(personal, "config.toml"), "developer_instructions = \"Fresh native.\"\n")
	t.Setenv("CODEX_HOME", personal)
	profile := filepath.Join(t.TempDir(), "codex")
	writeCodexTOML(t, filepath.Join(profile, "config.toml"), "developer_instructions = \"Stale mirror.\"\n")

	out := filepath.Join(t.TempDir(), "overlay")
	c := NewChatRuntime(nil)
	c.command = "/bin/sh"
	c.cmdArgs = []string{"-c", `printf %s "$CODEX_CONFIG" > "$0"`, out}
	ad, ok := backend.For("codex-acp")
	if !ok {
		t.Fatal("codex adapter missing")
	}
	spec := LaunchSpec{
		Cwd:          t.TempDir(),
		BackendType:  "codex-acp",
		SystemPrompt: "Chuck guidance.",
		Env:          []string{"HOME=/x", "CODEX_HOME=" + profile},
	}
	cmd, err := c.spawnCmd(ad, spec)
	if err != nil {
		t.Fatalf("spawnCmd: %v", err)
	}
	if err := c.startCmd(cmd, spec); err != nil {
		t.Fatalf("startCmd: %v", err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("wait: %v", err)
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := codexOverlayInstructions(t, []string{"CODEX_CONFIG=" + string(raw)})
	if want := "Fresh native.\n\nChuck guidance."; got != want {
		t.Fatalf("child developer_instructions = %q, want %q", got, want)
	}
	if strings.Contains(got, "Stale") {
		t.Fatal("child received the stale mirrored instructions")
	}
}
