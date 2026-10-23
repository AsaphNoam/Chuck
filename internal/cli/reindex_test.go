package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AsaphNoam/Chuck/internal/config"
	"github.com/AsaphNoam/Chuck/internal/index"
	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
	"github.com/AsaphNoam/Chuck/internal/transcript"
)

// FS-05.A27: the actual command reports partial repair, preserves the failed
// conversation and repairs its healthy neighbor. Restoring the source permits
// the ordinary retry and completion message.
func TestReindexCommandPartialRepairAndRetry(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CHUCK_HOME", home)
	cfg := config.NewWithHome(home)
	if err := cfg.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := cfg.SeedIfAbsent(); err != nil {
		t.Fatal(err)
	}
	st, err := state.Open(home)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	for _, id := range []string{"a_bad", "b_good"} {
		m := runtime.SessionMetaData{Name: "Original", Backend: "claude", Interface: "chat", CreatedAt: "2026-10-22T10:00:00Z", Cwd: home, SessionID: "old-native"}
		if err := index.New(st.DB()).UpsertSessionMeta(id, m); err != nil {
			t.Fatal(err)
		}
		m.Name = "Repaired"
		w, err := transcript.Open(home, id, &m)
		if err != nil {
			t.Fatal(err)
		}
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
	}
	badPath := filepath.Join(home, "sessions", "a_bad", "transcript.ndjson")
	backup, err := os.ReadFile(badPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(badPath); err != nil {
		t.Fatal(err)
	}
	run := func() (string, error) {
		root := NewRootCmd()
		root.SetArgs([]string{"reindex"})
		var out bytes.Buffer
		root.SetOut(&out)
		root.SetErr(&out)
		err := root.Execute()
		return out.String(), err
	}
	out, err := run()
	if err == nil || !strings.Contains(err.Error(), "a_bad") || !strings.Contains(err.Error(), "preserv") || strings.Contains(out, "reindexed archive") {
		t.Fatalf("partial repair: output=%q err=%v", out, err)
	}
	for id, want := range map[string]string{"a_bad": "Original", "b_good": "Repaired"} {
		snap, err := st.ReadSession(id)
		if err != nil || snap.Name != want || snap.LastSessionID != "old-native" {
			t.Fatalf("partial repair %s = %+v, %v", id, snap, err)
		}
	}
	if err := os.WriteFile(badPath, backup, 0o600); err != nil {
		t.Fatal(err)
	}
	out, err = run()
	if err != nil || !strings.Contains(out, "reindexed archive") {
		t.Fatalf("retry: output=%q err=%v", out, err)
	}
	snap, err := st.ReadSession("a_bad")
	if err != nil || snap.Name != "Repaired" {
		t.Fatalf("restored source = %+v, %v", snap, err)
	}
}

// ARCH-1: a corrupt or unreadable dashboard pidfile cannot prove that the
// server is absent. The command must refuse before opening its SQLite writer.
func TestReindexCommandRefusesUninspectablePidfile(t *testing.T) {
	tests := []struct {
		name       string
		unreadable bool
		setup      func(string) error
	}{
		{
			name: "malformed JSON",
			setup: func(home string) error {
				return os.WriteFile(pidfilePath(home), []byte("{"), 0o600)
			},
		},
		{
			name:       "unreadable file",
			unreadable: true,
			setup: func(home string) error {
				path := pidfilePath(home)
				if err := os.WriteFile(path, []byte(`{"pid":2000000000}`), 0o600); err != nil {
					return err
				}
				return os.Chmod(path, 0)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("CHUCK_HOME", home)
			t.Setenv("CHUCK_RUNTIME_ROOT", "")
			cfg := config.NewWithHome(home)
			if err := cfg.EnsureLayout(); err != nil {
				t.Fatal(err)
			}
			if err := cfg.SeedIfAbsent(); err != nil {
				t.Fatal(err)
			}
			if err := tt.setup(home); err != nil {
				t.Fatal(err)
			}
			if tt.unreadable {
				if _, err := os.ReadFile(pidfilePath(home)); err == nil {
					t.Skip("the test process can read a mode-000 file")
				}
			}

			root := NewRootCmd()
			root.SetArgs([]string{"reindex"})
			var out bytes.Buffer
			root.SetOut(&out)
			root.SetErr(&out)
			err := root.Execute()
			if err == nil || !strings.Contains(err.Error(), "dashboard pidfile") {
				t.Fatalf("reindex error = %v, want actionable pidfile inspection error; output=%q", err, out.String())
			}
			if strings.Contains(out.String(), "reindexed archive") {
				t.Fatalf("reindex reported success: %q", out.String())
			}
			if _, err := os.Stat(filepath.Join(home, "state.db")); !os.IsNotExist(err) {
				t.Fatalf("state.db stat error = %v, want no database writer opened", err)
			}
		})
	}
}
