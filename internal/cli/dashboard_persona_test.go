package cli

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/AsaphNoam/Chuck/internal/config"
)

func TestResolvePreparedConfigUpgradesLegacyDefault(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CHUCK_HOME", home)
	s := config.NewWithHome(home)
	if err := s.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := s.SeedIfAbsent(); err != nil {
		t.Fatal(err)
	}
	cfg, err := s.ReadConfig()
	if err != nil {
		t.Fatal(err)
	}
	cfg.DefaultRole = "implementer"
	if err := s.WriteConfig(cfg); err != nil {
		t.Fatal(err)
	}
	got, err := resolvePreparedConfig(s, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	if got.DefaultRole != "default" {
		t.Fatalf("default role = %q, want default", got.DefaultRole)
	}
	if _, err := os.Stat(filepath.Join(home, "default-persona-upgrade.json")); err != nil {
		t.Fatalf("upgrade marker missing: %v", err)
	}
}

func TestResolvePreparedConfigLeavesCorruptConfigAndMarkerPending(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CHUCK_HOME", home)
	s := config.NewWithHome(home)
	if err := s.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := s.SeedIfAbsent(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "config.json"), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := resolvePreparedConfig(s, slog.New(slog.NewTextHandler(io.Discard, nil))); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, "default-persona-upgrade.json")); !os.IsNotExist(err) {
		t.Fatalf("marker status after corrupt config: %v", err)
	}
}
