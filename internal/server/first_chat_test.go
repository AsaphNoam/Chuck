package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/AsaphNoam/Chuck/internal/backend/credcheck"
	"github.com/AsaphNoam/Chuck/internal/config"
)

func TestFirstChatReadDoesNotProbeAndProjectsNeedDirectory(t *testing.T) {
	server, handler, _ := refreshFixture(t, true)
	settings := config.DefaultConfig()
	settings.FirstChatEligible = true
	if err := server.configStore.WriteConfig(settings); err != nil {
		t.Fatal(err)
	}
	if err := server.configStore.WriteProject("missing", config.Project{Title: "Missing", Cwd: t.TempDir() + "/missing"}); err != nil {
		t.Fatal(err)
	}
	if err := server.configStore.WriteProject("ready", config.Project{Title: "Ready", Cwd: t.TempDir()}); err != nil {
		t.Fatal(err)
	}
	server.credCheck = func(context.Context, config.Backend, config.Model, map[string]string) credcheck.CredResult {
		t.Fatal("config read ran credential probe")
		return credcheck.CredResult{}
	}
	recorder := doGET(t, handler, "/api/config")
	var response configResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if !response.Onboarding.FirstChat || response.Onboarding.Steps.Backend.Done || len(response.Onboarding.LaunchableProjects) != 1 || response.Onboarding.LaunchableProjects[0] != "ready" {
		t.Fatalf("onboarding = %+v", response.Onboarding)
	}
}

func TestProviderCheckOnlySkipsModelImportAndSerializesTarget(t *testing.T) {
	server, handler, _ := refreshFixture(t, true)
	_, etag := backendsBody(t, handler)
	started, release := make(chan struct{}), make(chan struct{})
	server.credCheck = func(context.Context, config.Backend, config.Model, map[string]string) credcheck.CredResult {
		close(started)
		<-release
		return credcheck.CredResult{Status: "ok"}
	}
	finished := make(chan *httptest.ResponseRecorder, 1)
	go func() { finished <- refresh(t, handler, etag, `{"check_only":true}`) }()
	<-started
	second := refresh(t, handler, etag, `{"check_only":true}`)
	if second.Code != http.StatusTooManyRequests && second.Code != http.StatusConflict {
		close(release)
		t.Fatalf("duplicate check = %d %s", second.Code, second.Body)
	}
	close(release)
	first := <-finished
	var response refreshProviderResponse
	if err := json.Unmarshal(first.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if first.Code != http.StatusOK || first.Header().Get("ETag") != etag || response.Catalog == nil || response.Catalog.Status != "disabled" {
		t.Fatalf("check = %d %s", first.Code, first.Body)
	}
}

func TestSaveAndCheckReturnsCommittedSettingsWhenCheckBusy(t *testing.T) {
	server, handler, _ := refreshFixture(t, false)
	_, etag := backendsBody(t, handler)
	catalog, _ := server.configStore.ReadBackends()
	target := catalog.Backends["claude"]
	target.Name = "Saved Claude"
	body, _ := json.Marshal(map[string]any{"backend": target})
	server.providerChecks = map[string]bool{providerObsKey("claude", "sonnet"): true}
	response := refresh(t, handler, etag, string(body))
	var result refreshProviderResponse
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusOK || result.Backend == nil || result.CheckError == nil || result.Runtime != nil || result.Credentials != nil || result.Catalog != nil {
		t.Fatalf("partial = %d %s", response.Code, response.Body)
	}
	saved, _ := server.configStore.ReadBackends()
	if saved.Backends["claude"].Name != "Saved Claude" || response.Header().Get("ETag") == etag {
		t.Fatalf("saved = %+v", saved)
	}
	if strings.Contains(response.Body.String(), `"credentials"`) {
		t.Fatal("unperformed check claimed credentials")
	}
}

func TestFirstChatCompletionRetryRetainsPromotedDefault(t *testing.T) {
	server := testServer(t, false)
	if err := server.configStore.SeedIfAbsent(); err != nil {
		t.Fatal(err)
	}
	catalog, _ := server.configStore.ReadBackends()
	selected := ""
	for backendID, backend := range catalog.Backends {
		if backend.Type == "codex-acp" {
			selected = backendID
		}
	}
	if selected == "" {
		t.Fatal("seed has no Codex backend")
	}
	originalETag := backendCatalogETag(catalog)
	handler := server.routes()
	complete := func() *httptest.ResponseRecorder {
		payload, _ := json.Marshal(map[string]any{"onboarding_complete": true, "onboarding_backend": selected, "onboarding_catalog_etag": originalETag})
		request := newLocalRequest(http.MethodPut, "/api/config", strings.NewReader(string(payload)))
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		return recorder
	}
	write := server.writeConfig
	server.writeConfig = func(config.Config) error { return errors.New("injected config failure") }
	first := complete()
	if first.Code != http.StatusInternalServerError {
		t.Fatalf("completion = %d %s", first.Code, first.Body)
	}
	promoted, _ := server.configStore.ReadBackends()
	settings, _ := server.configStore.ReadConfig()
	if !promoted.Backends[selected].Default || settings.OnboardingComplete {
		t.Fatalf("default/completion lost after partial failure")
	}
	server.writeConfig = write
	second := complete()
	if second.Code != http.StatusOK {
		t.Fatalf("retry = %d %s", second.Code, second.Body)
	}
	settled, _ := server.configStore.ReadBackends()
	settings, _ = server.configStore.ReadConfig()
	if !settings.OnboardingComplete || backendCatalogETag(settled) != backendCatalogETag(promoted) {
		t.Fatal("retry rewrote promoted catalog or failed to complete")
	}
	if final := complete(); final.Code != http.StatusOK {
		t.Fatalf("replay = %d", final.Code)
	}
}

func TestSaveAndCheckFencesCatalogChangeDuringProbe(t *testing.T) {
	server, handler, _ := refreshFixture(t, false)
	_, etag := backendsBody(t, handler)
	catalog, _ := server.configStore.ReadBackends()
	target := catalog.Backends["claude"]
	target.Name = "Committed Claude"
	server.credCheck = func(context.Context, config.Backend, config.Model, map[string]string) credcheck.CredResult {
		latest, err := server.configStore.ReadBackends()
		if err != nil {
			t.Fatal(err)
		}
		changed := latest.Backends["claude"]
		changed.Name = "Newer Claude"
		latest.Backends["claude"] = changed
		if err := server.configStore.WriteBackends(latest); err != nil {
			t.Fatal(err)
		}
		return credcheck.CredResult{Status: "ok"}
	}
	payload, _ := json.Marshal(map[string]any{"backend": target})
	recorder := refresh(t, handler, etag, string(payload))
	var response refreshProviderResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if recorder.Code != http.StatusOK || response.Backend == nil || response.Backend.Name != "Committed Claude" || response.CheckError == nil || response.CheckError.Code != "backend_catalog_changed" || response.Runtime == nil || response.Credentials == nil {
		t.Fatalf("partial = %d %s", recorder.Code, recorder.Body)
	}
	latest, _ := server.configStore.ReadBackends()
	if latest.Backends["claude"].Name != "Newer Claude" || recorder.Header().Get("ETag") != backendCatalogETag(latest) {
		t.Fatal("late check replaced newer settings or returned old ETag")
	}
	if len(server.providerObs) != 0 {
		t.Fatal("stale check published a readiness observation")
	}
}
