package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/AsaphNoam/Chuck/internal/backend/credcheck"
	"github.com/AsaphNoam/Chuck/internal/backend/providerexec"
	"github.com/AsaphNoam/Chuck/internal/config"
	"github.com/AsaphNoam/Chuck/internal/runtime"
)

// providerRuntime is one backend/model's next-start provider metadata
// (TS-03.R52). Version and CheckedAt appear only for a fresh observation of the
// same executable; an available path with no version is unverified, not
// incompatible.
type providerRuntime struct {
	Source          string `json:"source"`
	State           string `json:"state"`
	Path            string `json:"path,omitempty"`
	Version         string `json:"version,omitempty"`
	CheckedAt       string `json:"checked_at,omitempty"`
	BundleAvailable bool   `json:"bundle_available"`
}

// providerObservation is a version a Refresh provider accepted for one
// executable identity.
type providerObservation struct {
	identity    string
	version     string
	checkedAt   time.Time
	catalog     string
	credentials credcheck.CredResult
}

// providerObservationTTL bounds how long a GET reuses an observation (TS-04.R72).
const providerObservationTTL = 60 * time.Second

// providerRefreshSlots caps concurrent refreshes process-wide, with no waiting
// queue (TS-03.R53).
var providerRefreshSlots = make(chan struct{}, 2)

func providerObsKey(backendID, modelID string) string { return backendID + "\x00" + modelID }

// executableIdentity names the selection and the file it reaches, so an
// observation is never reused after a symlink retarget or in-place update.
func executableIdentity(sel providerexec.Selection) string {
	id := sel.Source + "\x00" + sel.Path
	if target, err := filepath.EvalSymlinks(sel.Path); err == nil {
		id += "\x00" + target
		if info, err := os.Stat(target); err == nil {
			id += fmt.Sprintf("\x00%d\x00%d", info.Size(), info.ModTime().UnixNano())
		}
	}
	return id
}

func (s *Server) discardProviderObservations() {
	s.providerObsMu.Lock()
	s.providerObs = nil
	s.providerObsMu.Unlock()
}

func (s *Server) publishProviderObservation(backendID, modelID string, obs providerObservation) {
	s.providerObsMu.Lock()
	if s.providerObs == nil {
		s.providerObs = map[string]providerObservation{}
	}
	for key := range s.providerObs {
		if strings.HasPrefix(key, backendID+"\x00") {
			delete(s.providerObs, key)
		}
	}
	s.providerObs[providerObsKey(backendID, modelID)] = obs
	s.providerObsMu.Unlock()
}

// providerRuntimeFor projects one selection, attaching a fresh matching
// observation. It only resolves and stats: no provider command runs.
func (s *Server) providerRuntimeFor(backendID, modelID string, sel providerexec.Selection) providerRuntime {
	out := providerRuntime{Source: sel.Source, State: sel.State, Path: sel.Path}
	bundle, _ := providerexec.Resolve(providerexec.Input{BackendType: map[string]string{"claude": "claude-acp", "codex": "codex-acp"}[sel.Provider.ID], Mode: config.ProviderModeBundled, ProcessEnv: os.Environ()})
	out.BundleAvailable = bundle.Available()
	if !sel.Available() {
		return out
	}
	s.providerObsMu.Lock()
	obs, ok := s.providerObs[providerObsKey(backendID, modelID)]
	s.providerObsMu.Unlock()
	if ok && time.Since(obs.checkedAt) <= providerObservationTTL && obs.identity == executableIdentity(sel) {
		out.Version, out.CheckedAt = obs.version, obs.checkedAt.UTC().Format(time.RFC3339)
	}
	return out
}

// providerRuntimes is the GET/PUT projection for every Claude/Codex
// backend/model. Empty maps are {} (INV §11).
func (s *Server) providerRuntimes(b config.BackendsConfig) map[string]map[string]providerRuntime {
	out := map[string]map[string]providerRuntime{}
	for backendID, be := range b.Backends {
		if _, ok := providerexec.ForBackendType(be.Type); !ok {
			continue
		}
		models := map[string]providerRuntime{}
		for modelID, model := range be.Models {
			sel, _ := providerexec.ForBackend(be, model)
			models[modelID] = s.providerRuntimeFor(backendID, modelID, sel)
		}
		out[backendID] = models
	}
	return out
}

type refreshCatalogResult struct {
	Status     string `json:"status"` // added | unchanged | disabled | unavailable
	AddedCount int    `json:"added_count"`
}

type refreshProviderResponse struct {
	Runtime     *providerRuntime      `json:"runtime,omitempty"`
	Credentials *credcheck.CredResult `json:"credentials,omitempty"`
	Catalog     *refreshCatalogResult `json:"catalog,omitempty"`
	Backend     *config.Backend       `json:"backend,omitempty"`
	CheckError  *runtime.APIError     `json:"check_error,omitempty"`
}

// handleRefreshProvider implements POST /api/backends/{id}/refresh-provider
// (FS-09.R72, TS-03.R53): recheck one saved backend/model's selected provider,
// its readiness and, when autosync is on, the add-only local model import.
// Provider execution happens outside the catalog lock; under it the original
// catalog is revalidated before anything is written or published.
func (s *Server) handleRefreshProvider(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ModelID   string          `json:"model_id"`
		CheckOnly bool            `json:"check_only"`
		Backend   *config.Backend `json:"backend"`
	}
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		writeAPIError(w, apiError(runtime.CodeValidation, "request body too large or unreadable"))
		return
	}
	if len(raw) > 0 {
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&req); err != nil {
			writeAPIError(w, apiError(runtime.CodeValidation, "invalid provider check request"))
			return
		}
	}
	if (req.CheckOnly && req.Backend != nil) || (req.Backend == nil && len(raw) > 4<<10) {
		writeAPIError(w, apiError(runtime.CodeValidation, "check_only and backend are mutually exclusive; checks are limited to 4 KiB"))
		return
	}

	backendID := r.PathValue("id")
	current, err := s.configStore.ReadBackends()
	if err != nil {
		if errors.Is(err, config.ErrNotFound) || errors.Is(err, config.ErrCorrupt) {
			current = config.DefaultBackends()
		} else {
			writeAPIError(w, apiError(runtime.CodeInternal, "internal error"))
			return
		}
	}
	be, ok := current.Backends[backendID]
	if !ok {
		writeAPIError(w, apiError(runtime.CodeNotFound, "unknown backend: "+backendID))
		return
	}
	if _, supported := providerexec.ForBackendType(be.Type); !supported {
		writeAPIError(w, apiError(runtime.CodeValidation, "provider refresh is available for Claude and Codex backends"))
		return
	}
	var committed *config.Backend
	if req.Backend != nil {
		s.catalogMu.Lock()
		latest, readErr := s.configStore.ReadBackends()
		if readErr != nil || backendCatalogETag(latest) != r.Header.Get("If-Match") {
			s.catalogMu.Unlock()
			writeAPIError(w, apiError(runtime.CodeBackendCatalogChanged, "backend catalog changed; reload before saving"))
			return
		}
		current = latest
		latest.Backends = maps.Clone(latest.Backends)
		for targetID, target := range latest.Backends {
			target.Models = maps.Clone(target.Models)
			latest.Backends[targetID] = target
		}
		be, ok = latest.Backends[backendID]
		if !ok {
			s.catalogMu.Unlock()
			writeAPIError(w, apiError(runtime.CodeNotFound, "backend was removed"))
			return
		}
		if req.Backend.Type != be.Type || req.Backend.Default != be.Default {
			s.catalogMu.Unlock()
			writeAPIError(w, apiError(runtime.CodeValidation, "item save cannot change backend type or default"))
			return
		}
		latest.Backends[backendID] = *req.Backend
		if validation := config.ValidateBackendsConfig(&latest); validation != nil {
			s.catalogMu.Unlock()
			writeValidationError(w, validation)
			return
		}
		if fieldErr := config.BackendNameError("backend.name", req.Backend.Name); fieldErr != nil {
			s.catalogMu.Unlock()
			writeValidationError(w, &config.ValidationErrors{Errors: []config.FieldError{*fieldErr}})
			return
		}
		writeErr := s.configStore.WriteBackends(latest)
		if writeErr == nil {
			if pruneErr := s.pruneIncompatibleConfigSources(latest); pruneErr != nil {
				writeErr = pruneErr
				if restoreErr := s.configStore.WriteBackends(current); restoreErr != nil {
					s.log.Error("backends: item restore", "err", restoreErr)
				}
			}
		}
		s.catalogMu.Unlock()
		if writeErr != nil {
			if durable, readErr := s.configStore.ReadBackends(); readErr == nil && backendCatalogETag(durable) != backendCatalogETag(current) {
				saved := durable.Backends[backendID]
				s.invalidateOnboardingCache()
				w.Header().Set("ETag", backendCatalogETag(durable))
				writeJSON(w, http.StatusOK, refreshProviderResponse{Backend: &saved, CheckError: apiError(runtime.CodeInternal, "backend settings changed but the save could not finish; reload before checking")})
				return
			}
			writeAPIError(w, apiError(runtime.CodeInternal, "could not save backend"))
			return
		}
		s.invalidateOnboardingCache()
		current, be = latest, latest.Backends[backendID]
		committed = &be
		req.ModelID = be.DefaultModel
	}
	var checkedRuntime *providerRuntime
	var checkedCredentials *credcheck.CredResult
	failCheck := func(code, message string) {
		if committed == nil {
			writeAPIError(w, apiError(code, message))
			return
		}
		responseETag := backendCatalogETag(current)
		if latest, readErr := s.configStore.ReadBackends(); readErr == nil {
			responseETag = backendCatalogETag(latest)
		}
		w.Header().Set("ETag", responseETag)
		writeJSON(w, http.StatusOK, refreshProviderResponse{Backend: committed, CheckError: apiError(code, message), Runtime: checkedRuntime, Credentials: checkedCredentials})
	}
	modelID := req.ModelID
	if modelID == "" {
		modelID = be.DefaultModel
	}
	model, ok := be.Models[modelID]
	if !ok {
		writeAPIError(w, apiError(runtime.CodeNotFound, "unknown model: "+modelID))
		return
	}
	if _, ok := providerexec.ForBackendType(be.Type); !ok {
		writeAPIError(w, apiError(runtime.CodeValidation, "provider refresh is available for Claude and Codex backends"))
		return
	}
	original := backendCatalogETag(current)
	if committed == nil && r.Header.Get("If-Match") != original {
		writeAPIError(w, apiError(runtime.CodeBackendCatalogChanged, "backend catalog changed; reload it before refreshing"))
		return
	}
	claim := providerObsKey(backendID, modelID)
	s.providerObsMu.Lock()
	if s.providerChecks == nil {
		s.providerChecks = map[string]bool{}
	}
	if s.providerChecks[claim] {
		s.providerObsMu.Unlock()
		failCheck(runtime.CodeProviderCheckBusy, "this provider is already checking; retry in a moment")
		return
	}
	s.providerChecks[claim] = true
	s.providerObsMu.Unlock()
	defer func() { s.providerObsMu.Lock(); delete(s.providerChecks, claim); s.providerObsMu.Unlock() }()
	select {
	case providerRefreshSlots <- struct{}{}:
		defer func() { <-providerRefreshSlots }()
	default:
		failCheck(runtime.CodeProviderCheckBusy, "provider checks are already running; retry in a moment")
		return
	}

	// Bounded checks, outside the lock, sharing readiness's deadline.
	ctx, cancel := context.WithTimeout(r.Context(), credcheck.DefaultTimeout)
	defer cancel()
	sel, _ := providerexec.ForBackend(be, model)
	obs := providerObservation{identity: executableIdentity(sel), checkedAt: time.Now()}
	if sel.Available() {
		obs.version = providerexec.ProbeVersion(ctx, sel.Path, composeEnv(os.Environ(), be.Env, model.Env))
	}
	credentials := s.credCheck(ctx, be, model, credcheck.MergeEnv(be.Env, model.Env))
	observedRuntime := s.providerRuntimeFor(backendID, modelID, sel)
	observedRuntime.Version, observedRuntime.CheckedAt = obs.version, obs.checkedAt.UTC().Format(time.RFC3339)
	checkedRuntime, checkedCredentials = &observedRuntime, &credentials

	s.catalogMu.Lock()
	latest, err := s.configStore.ReadBackends()
	if errors.Is(err, config.ErrNotFound) || errors.Is(err, config.ErrCorrupt) {
		latest, err = config.DefaultBackends(), nil
	}
	latestSelection, _ := providerexec.ForBackend(latest.Backends[backendID], latest.Backends[backendID].Models[modelID])
	if err != nil || backendCatalogETag(latest) != original || executableIdentity(latestSelection) != obs.identity {
		s.catalogMu.Unlock()
		failCheck(runtime.CodeBackendCatalogChanged, "backend catalog changed during the check; reload and retry")
		return
	}
	catalog := refreshCatalogResult{Status: "disabled"}
	enabled, available, added := false, false, 0
	if !req.CheckOnly {
		enabled, available, added = config.RefreshBackendModels(&latest, backendID)
	}
	switch {
	case !enabled:
	case !available:
		catalog.Status = "unavailable"
	case added == 0:
		catalog.Status = "unchanged"
	default:
		if err := s.configStore.WriteBackends(latest); err != nil {
			s.catalogMu.Unlock()
			s.log.Error("backends: refresh import", "err", err)
			failCheck(runtime.CodeInternal, "provider checked; model import failed")
			return
		}
		catalog = refreshCatalogResult{Status: "added", AddedCount: added}
	}
	if added > 0 {
		s.invalidateOnboardingCache() // a catalog write: also discards observations
	} else {
		s.onboardingCacheMu.Lock()
		s.onboardingCache = nil // readiness was rechecked
		s.onboardingCacheMu.Unlock()
	}
	etag := backendCatalogETag(latest)
	obs.catalog, obs.credentials = etag, credentials
	s.publishProviderObservation(backendID, modelID, obs)
	s.catalogMu.Unlock()

	w.Header().Set("ETag", etag)
	projected := s.providerRuntimeFor(backendID, modelID, sel)
	writeJSON(w, http.StatusOK, refreshProviderResponse{
		Runtime:     &projected,
		Credentials: &credentials,
		Catalog:     &catalog,
		Backend:     committed,
	})
}

func (s *Server) providerReady(backendID string, catalog config.BackendsConfig, selection providerexec.Selection) bool {
	target := catalog.Backends[backendID]
	s.providerObsMu.Lock()
	defer s.providerObsMu.Unlock()
	observation, exists := s.providerObs[providerObsKey(backendID, target.DefaultModel)]
	return exists && selection.Available() && observation.catalog == backendCatalogETag(catalog) && observation.identity == executableIdentity(selection) && time.Since(observation.checkedAt) <= providerObservationTTL && observation.credentials.Status == "ok"
}

func (s *Server) promoteFirstChatBackend(backendID, etag string) error {
	s.catalogMu.Lock()
	defer s.catalogMu.Unlock()
	catalog, err := s.configStore.ReadBackends()
	if err != nil {
		return err
	}
	if backendCatalogETag(catalog) != etag {
		return nil
	}
	target, exists := catalog.Backends[backendID]
	if !exists || target.Default {
		return nil
	}
	seeded := config.DefaultBackends()
	for currentID, backend := range catalog.Backends {
		if backend.Default && !seeded.Backends[currentID].Default {
			return nil
		}
	}
	for currentID, backend := range catalog.Backends {
		backend.Default = currentID == backendID
		catalog.Backends[currentID] = backend
	}
	if err := s.configStore.WriteBackends(catalog); err != nil {
		return err
	}
	s.invalidateOnboardingCache()
	return nil
}
