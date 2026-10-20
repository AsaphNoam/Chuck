package configsource

import (
	"os"
	"path/filepath"
	"testing"
)

func writeChainFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// TS-04.R101: the nested-cwd fixtures mirror the pinned-native probe where
// config/read selects the repository-root project layer over the user layer.
func TestCodexDeveloperInstructionsProjectChain(t *testing.T) {
	setup := func(t *testing.T) (home, repo, nested string) {
		root := t.TempDir()
		home = filepath.Join(root, "home")
		repo = filepath.Join(root, "repo")
		nested = filepath.Join(repo, "packages", "app")
		writeChainFile(t, filepath.Join(home, "config.toml"), "developer_instructions = \"USER\"\n")
		if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(nested, 0o700); err != nil {
			t.Fatal(err)
		}
		return home, repo, nested
	}

	t.Run("ancestor only project layer wins over user", func(t *testing.T) {
		home, repo, nested := setup(t)
		writeChainFile(t, filepath.Join(repo, ".codex", "config.toml"), "developer_instructions = \"PROJECT\"\n")
		got, err := CodexDeveloperInstructions(home, nested, nil)
		if err != nil || got != "PROJECT" {
			t.Fatalf("got %q, %v; want PROJECT", got, err)
		}
	})

	t.Run("nearer layer overrides ancestor", func(t *testing.T) {
		home, repo, nested := setup(t)
		writeChainFile(t, filepath.Join(repo, ".codex", "config.toml"), "developer_instructions = \"PROJECT\"\n")
		writeChainFile(t, filepath.Join(repo, "packages", ".codex", "config.toml"), "developer_instructions = \"MID\"\n")
		got, err := CodexDeveloperInstructions(home, nested, nil)
		if err != nil || got != "MID" {
			t.Fatalf("got %q, %v; want MID", got, err)
		}
		writeChainFile(t, filepath.Join(nested, ".codex", "config.toml"), "developer_instructions = \"APP\"\n")
		got, err = CodexDeveloperInstructions(home, nested, nil)
		if err != nil || got != "APP" {
			t.Fatalf("got %q, %v; want APP", got, err)
		}
	})

	t.Run("outside a repository only cwd is read", func(t *testing.T) {
		root := t.TempDir()
		home := filepath.Join(root, "home")
		writeChainFile(t, filepath.Join(home, "config.toml"), "developer_instructions = \"USER\"\n")
		parent := filepath.Join(root, "parent")
		cwd := filepath.Join(parent, "cwd")
		writeChainFile(t, filepath.Join(parent, ".codex", "config.toml"), "developer_instructions = \"PARENT\"\n")
		if err := os.MkdirAll(cwd, 0o700); err != nil {
			t.Fatal(err)
		}
		got, err := CodexDeveloperInstructions(home, cwd, nil)
		if err != nil || got != "USER" {
			t.Fatalf("got %q, %v; want USER", got, err)
		}
		writeChainFile(t, filepath.Join(cwd, ".codex", "config.toml"), "developer_instructions = \"CWD\"\n")
		got, err = CodexDeveloperInstructions(home, cwd, nil)
		if err != nil || got != "CWD" {
			t.Fatalf("got %q, %v; want CWD", got, err)
		}
	})

	t.Run("malformed or non-string ancestor layer fails closed", func(t *testing.T) {
		for name, body := range map[string]string{
			"malformed":  "developer_instructions = \n",
			"non-string": "developer_instructions = 3\n",
		} {
			home, repo, nested := setup(t)
			writeChainFile(t, filepath.Join(repo, ".codex", "config.toml"), body)
			if got, err := CodexDeveloperInstructions(home, nested, nil); err == nil {
				t.Fatalf("%s: got %q, nil error", name, got)
			}
		}
	})

	t.Run("configured project_root_markers", func(t *testing.T) {
		home, repo, nested := setup(t)
		writeChainFile(t, filepath.Join(repo, ".codex", "config.toml"), "developer_instructions = \"PROJECT\"\n")
		writeChainFile(t, filepath.Join(home, "config.toml"),
			"developer_instructions = \"USER\"\nproject_root_markers = []\n")
		if got, err := CodexDeveloperInstructions(home, nested, nil); err != nil || got != "USER" {
			t.Fatalf("empty markers: got %q, %v; want USER", got, err)
		}
		writeChainFile(t, filepath.Join(repo, "MARK"), "")
		writeChainFile(t, filepath.Join(home, "config.toml"),
			"developer_instructions = \"USER\"\nproject_root_markers = [\"MARK\"]\n")
		if got, err := CodexDeveloperInstructions(home, nested, nil); err != nil || got != "PROJECT" {
			t.Fatalf("MARK: got %q, %v; want PROJECT", got, err)
		}
		writeChainFile(t, filepath.Join(home, "config.toml"), "project_root_markers = \"x\"\n")
		if _, err := CodexDeveloperInstructions(home, nested, nil); err == nil {
			t.Fatal("non-list project_root_markers accepted")
		}
	})

	// Native precedence for markers: system < user < runtime overlay < managed.
	t.Run("effective project_root_markers", func(t *testing.T) {
		system := t.TempDir()
		prev := codexSystemDir
		codexSystemDir = system
		t.Cleanup(func() { codexSystemDir = prev })
		home, repo, nested := setup(t)
		writeChainFile(t, filepath.Join(repo, "MARK"), "")
		writeChainFile(t, filepath.Join(repo, ".codex", "config.toml"), "developer_instructions = \"PROJECT\"\n")
		writeChainFile(t, filepath.Join(home, "config.toml"), "developer_instructions = \"USER\"\nproject_root_markers = []\n")
		overlay := map[string]any{"project_root_markers": []any{"MARK"}}
		if got, err := CodexDeveloperInstructions(home, nested, overlay); err != nil || got != "PROJECT" {
			t.Fatalf("overlay MARK: got %q, %v; want PROJECT", got, err)
		}
		writeChainFile(t, filepath.Join(system, "config.toml"), "project_root_markers = [\"MARK\"]\n")
		if got, err := CodexDeveloperInstructions(home, nested, nil); err != nil || got != "USER" {
			t.Fatalf("system below user: got %q, %v; want USER", got, err)
		}
		writeChainFile(t, filepath.Join(system, "managed_config.toml"), "project_root_markers = []\n")
		if got, err := CodexDeveloperInstructions(home, nested, overlay); err != nil || got != "USER" {
			t.Fatalf("managed above overlay: got %q, %v; want USER", got, err)
		}
		writeChainFile(t, filepath.Join(system, "managed_config.toml"), "project_root_markers = 1\n")
		if _, err := CodexDeveloperInstructions(home, nested, nil); err == nil {
			t.Fatal("non-list managed project_root_markers accepted")
		}
	})
}
