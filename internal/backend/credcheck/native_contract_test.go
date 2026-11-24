package credcheck

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/AsaphNoam/Chuck/internal/config"
)

func TestNativeStatusUnknownZeroExitDoesNotEstablishReadiness(t *testing.T) {
	for _, provider := range []string{"claude", "codex"} {
		for _, output := range []string{"", "Unexpected status format"} {
			t.Run(provider+"/"+output, func(t *testing.T) {
				directory := t.TempDir()
				executable := filepath.Join(directory, provider)
				if err := os.WriteFile(executable, []byte(fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n' %q\n", output)), 0o755); err != nil {
					t.Fatal(err)
				}
				isolateProviderPath(t, directory)
				var result CredResult
				if provider == "claude" {
					result = claudeProber{}.Check(context.Background(), config.Backend{}, config.Model{}, nil)
				} else {
					result = codexProber{}.Check(context.Background(), config.Backend{}, config.Model{}, nil)
				}
				if result.Status != "skipped" {
					t.Fatalf("unknown zero-exit status = %+v; want skipped", result)
				}
				if provider == "claude" && result.Detail != "status_check_unavailable" {
					t.Fatalf("unknown Claude status = %+v", result)
				}
			})
		}
	}
}

func TestClaudeNativeJSONStatusPreservesAuthMeaning(t *testing.T) {
	for _, testCase := range []struct{ name, output, status, detail string }{
		{"signed in", `{"loggedIn":true,"apiProvider":"firstParty","subscriptionType":"max"}`, "ok", ""},
		{"signed out", `{"loggedIn":false,"apiProvider":"firstParty"}`, "failed", "not_logged_in"},
		{"external provider", `{"loggedIn":false,"apiProvider":"bedrock"}`, "ok", ""},
		{"API key", `{"loggedIn":false,"apiProvider":"firstParty","apiKeySource":"apiKeyHelper"}`, "ok", ""},
		{"unknown shape", `{"status":"authenticated"}`, "skipped", "status_check_unavailable"},
		{"array", `["authenticated"]`, "skipped", "status_check_unavailable"},
		{"scalar", `"authenticated"`, "skipped", "status_check_unavailable"},
		{"malformed object", `{"loggedIn":true,"status":"authenticated"`, "skipped", "status_check_unavailable"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			directory := t.TempDir()
			executable := filepath.Join(directory, "claude")
			if err := os.WriteFile(executable, []byte(fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n' %q\n", testCase.output)), 0o755); err != nil {
				t.Fatal(err)
			}
			isolateProviderPath(t, directory)
			result := claudeProber{}.Check(context.Background(), config.Backend{}, config.Model{}, nil)
			if result.Status != testCase.status || result.Detail != testCase.detail {
				t.Fatalf("native status = %+v; want %s/%s", result, testCase.status, testCase.detail)
			}
		})
	}
}
