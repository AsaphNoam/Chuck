package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSeedDefaultPersonaIsEmptyAwareAndDefaulted(t *testing.T) {
	s := newTestStore(t)
	if err := s.SeedIfAbsent(); err != nil {
		t.Fatal(err)
	}
	role, err := s.ReadRole("default")
	if err != nil {
		t.Fatal(err)
	}
	if role.SystemPrompt != "" || role.SkipPermissions != nil || !role.IsChuckAware() || role.ChuckAware == nil || !*role.ChuckAware {
		t.Fatalf("default role = %+v, want empty inherited aware role", role)
	}
	if got := DefaultConfig().DefaultRole; got != "default" {
		t.Fatalf("DefaultConfig().DefaultRole = %q, want default", got)
	}
}

func TestPrepareDefaultPersonaUpgradeOnceAndPreservesConfig(t *testing.T) {
	s := newTestStore(t)
	if err := s.SeedIfAbsent(); err != nil {
		t.Fatal(err)
	}
	cfg, err := s.ReadConfig()
	if err != nil {
		t.Fatal(err)
	}
	cfg.DefaultRole = "implementer"
	cfg.DefaultProject = "custom-project"
	cfg.SkipPermissions = true
	if err := s.WriteConfig(cfg); err != nil {
		t.Fatal(err)
	}
	got, err := s.PrepareDefaultPersonaUpgrade(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got.DefaultRole != "default" || got.DefaultProject != cfg.DefaultProject || !got.SkipPermissions {
		t.Fatalf("upgraded config = %+v, did not preserve fields", got)
	}
	marker, err := os.ReadFile(s.defaultPersonaUpgradePath())
	if err != nil || string(marker) != "{\n  \"version\": 1\n}\n" {
		t.Fatalf("marker = %q, err=%v", marker, err)
	}
	// A later deliberate choice survives because the marker makes this a no-op.
	cfg.DefaultRole = "reviewer"
	if err := s.WriteConfig(cfg); err != nil {
		t.Fatal(err)
	}
	got, err = s.PrepareDefaultPersonaUpgrade(cfg)
	if err != nil || got.DefaultRole != "reviewer" {
		t.Fatalf("second upgrade = %+v, err=%v", got, err)
	}
}

func TestPrepareDefaultPersonaUpgradeRejectsCustomizedDefaultAndRetriesFailures(t *testing.T) {
	t.Run("customized default", func(t *testing.T) {
		s := newTestStore(t)
		if err := s.SeedIfAbsent(); err != nil {
			t.Fatal(err)
		}
		if err := s.WriteRole("default", Role{Title: "Mine", SystemPrompt: "custom"}); err != nil {
			t.Fatal(err)
		}
		cfg, _ := s.ReadConfig()
		cfg.DefaultRole = "implementer"
		if _, err := s.PrepareDefaultPersonaUpgrade(cfg); err == nil {
			t.Fatal("customized default upgrade succeeded")
		}
		if _, err := os.Stat(s.defaultPersonaUpgradePath()); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("marker after conflict: err=%v", err)
		}
	})

	t.Run("awareness opt out is customized", func(t *testing.T) {
		s := newTestStore(t)
		if err := s.SeedIfAbsent(); err != nil {
			t.Fatal(err)
		}
		aware := false
		if err := s.WriteRole("default", Role{Title: "Default", ChuckAware: &aware}); err != nil {
			t.Fatal(err)
		}
		cfg, _ := s.ReadConfig()
		cfg.DefaultRole = "implementer"
		if _, err := s.PrepareDefaultPersonaUpgrade(cfg); err == nil {
			t.Fatal("unaware default upgrade succeeded")
		}
	})

	t.Run("marker write failure retries", func(t *testing.T) {
		s := newTestStore(t)
		if err := s.SeedIfAbsent(); err != nil {
			t.Fatal(err)
		}
		cfg, _ := s.ReadConfig()
		originalWriter := writeDefaultPersonaUpgradeMarker
		writeDefaultPersonaUpgradeMarker = func(string) error { return errors.New("injected marker write failure") }
		defer func() { writeDefaultPersonaUpgradeMarker = originalWriter }()
		if _, err := s.PrepareDefaultPersonaUpgrade(cfg); err == nil {
			t.Fatal("upgrade succeeded with marker write failure")
		}
		writeDefaultPersonaUpgradeMarker = originalWriter
		if _, err := s.PrepareDefaultPersonaUpgrade(cfg); err != nil {
			t.Fatalf("retry failed: %v", err)
		}
	})

	t.Run("config write failure retries", func(t *testing.T) {
		s := newTestStore(t)
		if err := s.SeedIfAbsent(); err != nil {
			t.Fatal(err)
		}
		cfg, _ := s.ReadConfig()
		cfg.DefaultRole = "implementer"
		if err := os.Remove(s.configPath()); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(s.configPath(), 0o700); err != nil {
			t.Fatal(err)
		}
		if _, err := s.PrepareDefaultPersonaUpgrade(cfg); err == nil {
			t.Fatal("upgrade succeeded with config directory")
		}
		if _, err := os.Stat(s.defaultPersonaUpgradePath()); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("marker after config failure: err=%v", err)
		}
		if err := os.Remove(s.configPath()); err != nil {
			t.Fatal(err)
		}
		if err := s.WriteConfig(cfg); err != nil {
			t.Fatal(err)
		}
		if _, err := s.PrepareDefaultPersonaUpgrade(cfg); err != nil {
			t.Fatalf("retry failed: %v", err)
		}
	})

	t.Run("invalid marker is actionable", func(t *testing.T) {
		s := newTestStore(t)
		if err := s.SeedIfAbsent(); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(s.Home(), fileDefaultPersonaUpgrade), []byte(`{"version":2}`), 0o600); err != nil {
			t.Fatal(err)
		}
		cfg, _ := s.ReadConfig()
		if _, err := s.PrepareDefaultPersonaUpgrade(cfg); err == nil {
			t.Fatal("invalid marker accepted")
		}
	})
}
