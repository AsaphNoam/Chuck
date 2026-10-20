package server

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AsaphNoam/Chuck/internal/config"
	"github.com/AsaphNoam/Chuck/internal/configsource"
	"github.com/AsaphNoam/Chuck/internal/runtime"
)

// FS-08.A14 / TS-07.R19 (ASRA-3): a bound Codex root other than the personal
// setup, or a named profile, refuses before any registration instead of
// launching the personal setup under the bound source's provenance. The
// personal root itself still launches and mirrors the personal sentinel.
func TestSourceSetupPreflightRefusesUnrealizableBindings(t *testing.T) {
	srv := testServer(t, true)
	personal := t.TempDir()
	personalCodex := filepath.Join(personal, ".codex")
	if err := os.MkdirAll(personalCodex, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(personalCodex, "config.toml"), []byte(`model = "personal-sentinel"`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CODEX_HOME", personalCodex)
	boundRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(boundRoot, "config.toml"), []byte(`model = "bound-user-sentinel"`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(boundRoot, "work.config.toml"), []byte(`model = "bound-profile-sentinel"`), 0o600); err != nil {
		t.Fatal(err)
	}
	projectDir := t.TempDir()
	if err := srv.configStore.WriteProject("codex-fed", config.Project{Title: "Codex Fed", Cwd: projectDir}); err != nil {
		t.Fatal(err)
	}
	srv.sourceMgr = configsource.NewManager(srv.configStore, map[string]configsource.Resolver{
		configsource.ProviderCodex: configsource.NewCodexResolver(configsource.CodexOptions{UserHome: personal, CodexHome: personalCodex}),
	}, nil)
	bind := func(root, profile string) {
		t.Helper()
		sources, err := srv.readConfigSources()
		if err != nil {
			t.Fatal(err)
		}
		sources.Sources["codex"] = config.SourceBinding{Provider: configsource.ProviderCodex, Mode: configsource.ModeLinked,
			Root: root, Profile: profile, Claims: []string{"launch_defaults"}, Approved: []string{root, projectDir}}
		if err := srv.configStore.WriteConfigSources(sources); err != nil {
			t.Fatal(err)
		}
	}
	launch := func() (runtime.LaunchSpec, *runtime.APIError) {
		spec, _, ae := srv.composeLaunch(context.Background(), launchRequest{Role: "implementer", Project: "codex-fed", Backend: "codex"})
		return spec, ae
	}

	for name, tc := range map[string]struct{ root, profile string }{
		"distinct root":    {boundRoot, ""},
		"selected profile": {boundRoot, "work"},
		"personal profile": {personalCodex, "work"},
	} {
		bind(tc.root, tc.profile)
		_, ae := launch()
		if ae == nil || ae.Code != runtime.CodeSourceInvalid {
			t.Fatalf("%s: want source_invalid, got %v", name, ae)
		}
		if len(srv.hookTokens) != 0 {
			t.Fatalf("%s: refusal left a registration", name)
		}
		if _, err := srv.readConfigSources(); err != nil {
			t.Fatalf("%s: binding no longer inspectable: %v", name, err)
		}
	}

	bind(personalCodex, "")
	spec, ae := launch()
	if ae != nil {
		t.Fatalf("personal root: %v", ae)
	}
	if !strings.Contains(string(spec.LaunchConfig), "personal-sentinel") {
		t.Fatalf("personal launch provenance = %s", spec.LaunchConfig)
	}
}

// The same preflight covers Claude: a bound root other than ~/.claude refuses.
func TestSourceSetupPreflightRefusesDistinctClaudeRoot(t *testing.T) {
	srv, _, projectDir := federationServer(t)
	other := canonicalPath(t, t.TempDir())
	if err := os.WriteFile(filepath.Join(other, "settings.json"), []byte(`{"model":"bound-sentinel"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	bindClaudeSource(t, srv, other, projectDir, config.SourceOverrides{})
	_, _, ae := srv.composeLaunch(context.Background(), launchRequest{Role: "implementer", Project: "fed"})
	if ae == nil || ae.Code != runtime.CodeSourceInvalid {
		t.Fatalf("want source_invalid, got %v", ae)
	}
	if len(srv.hookTokens) != 0 {
		t.Fatal("refusal left a registration")
	}
}
