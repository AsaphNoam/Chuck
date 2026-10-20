package server

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/AsaphNoam/Chuck/internal/config"
	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
)

// bindClaudeSource binds the federation fixture's Claude root with overrides.
func bindClaudeSource(t *testing.T, srv *Server, root, projectDir string, overrides config.SourceOverrides) {
	t.Helper()
	sources, err := srv.readConfigSources()
	if err != nil {
		t.Fatal(err)
	}
	sources.Sources["claude"] = config.SourceBinding{Provider: "claude-code", Mode: "linked", Root: root,
		Claims: []string{"launch_defaults"}, Approved: []string{root, projectDir}, Overrides: overrides}
	if err := srv.configStore.WriteConfigSources(sources); err != nil {
		t.Fatal(err)
	}
}

// editCatalogueModel changes the provider id behind agent's catalogue key, so
// a continuation rebuilt from the catalogue would visibly differ.
func editCatalogueModel(t *testing.T, srv *Server, backendID, key, id string) (config.Backend, config.Model) {
	t.Helper()
	backends, err := srv.configStore.ReadBackends()
	if err != nil {
		t.Fatal(err)
	}
	be := backends.Backends[backendID]
	m := be.Models[key]
	m.Model = id
	be.Models[key] = m
	backends.Backends[backendID] = be
	if err := srv.configStore.WriteBackends(backends); err != nil {
		t.Fatal(err)
	}
	return be, m
}

func snapshotOf(agent state.Agent, spec runtime.LaunchSpec) state.SessionSnapshot {
	return state.SessionSnapshot{AgentID: agent.AgentID, Name: agent.Name, Role: agent.Role, Project: agent.Project,
		Backend: agent.Backend, Model: agent.Model, Effort: agent.Effort, Interface: agent.Interface, Cwd: spec.Cwd,
		SystemPrompt: spec.SystemPrompt, LaunchConfig: spec.LaunchConfig, ExecutionContract: spec.ExecutionContract,
		LastSessionID: "native-1"}
}

// FS-08.A14 / FS-01.A35: a nonempty source model override is the frozen
// choice through resume, same-identity switch and rollback, even after the
// catalogue and the binding change; a new backend/model is explicit.
func TestContractKeepsSourceModelOverrideAcrossContinuation(t *testing.T) {
	srv, root, projectDir := federationServer(t)
	sentinel := "native-sentinel"
	bindClaudeSource(t, srv, root, projectDir, config.SourceOverrides{Model: &sentinel})

	fresh, agent, ae := srv.composeLaunch(context.Background(), launchRequest{Role: "implementer", Project: "fed"})
	if ae != nil {
		t.Fatal(ae)
	}
	if fresh.ModelID != sentinel || agent.Model == sentinel {
		t.Fatalf("fresh ModelID=%q identity=%q", fresh.ModelID, agent.Model)
	}
	snap := snapshotOf(agent, fresh)

	later := "later-override"
	bindClaudeSource(t, srv, root, projectDir, config.SourceOverrides{Model: &later})
	be, model := editCatalogueModel(t, srv, agent.Backend, agent.Model, "catalogue-edited")

	resumed, ae := srv.composeResumeSpec(agent, snap, be, model)
	if ae != nil {
		t.Fatal(ae)
	}
	if resumed.ModelID != sentinel {
		t.Fatalf("resume ModelID=%q, want frozen %q", resumed.ModelID, sentinel)
	}
	if resumed.ExecutionContract == nil || resumed.ExecutionContract.ModelChoice != (state.ModelChoice{Kind: "explicit", ID: sentinel}) {
		t.Fatalf("resume contract=%+v", resumed.ExecutionContract)
	}

	terminal := agent
	terminal.Interface = "terminal"
	switched, ae := srv.composeSwitchSpecFrom(context.Background(), terminal, "native-1", snap, nil)
	if ae != nil {
		t.Fatal(ae)
	}
	if switched.ModelID != sentinel {
		t.Fatalf("same-identity switch ModelID=%q, want %q", switched.ModelID, sentinel)
	}

	other := agent
	for key := range be.Models {
		if key != agent.Model {
			other.Model = key
			break
		}
	}
	if other.Model == agent.Model {
		t.Skip("fixture backend has a single model")
	}
	changed, ae := srv.composeSwitchSpecFrom(context.Background(), other, "native-1", snap, nil)
	if ae != nil {
		t.Fatal(ae)
	}
	if want := be.Models[other.Model].Model; changed.ModelID != want || changed.ExecutionContract.ModelChoice.Kind != "explicit" {
		t.Fatalf("model switch ModelID=%q contract=%+v, want explicit %q", changed.ModelID, changed.ExecutionContract, want)
	}

	// Rollback composes the previous identity from its pre-switch snapshot.
	rolled, ae := srv.composeSwitchSpecFrom(context.Background(), agent, "native-1", snap, nil)
	if ae != nil {
		t.Fatal(ae)
	}
	if rolled.ModelID != sentinel {
		t.Fatalf("rollback ModelID=%q, want %q", rolled.ModelID, sentinel)
	}
}

// FS-08.R37: an explicit empty model override is native inheritance, frozen
// as such and kept on resume.
func TestContractEmptyModelOverrideIsNative(t *testing.T) {
	srv, root, projectDir := federationServer(t)
	empty := ""
	bindClaudeSource(t, srv, root, projectDir, config.SourceOverrides{Model: &empty})

	fresh, agent, ae := srv.composeLaunch(context.Background(), launchRequest{Role: "implementer", Project: "fed"})
	if ae != nil {
		t.Fatal(ae)
	}
	if fresh.ModelID != "" || fresh.ExecutionContract.ModelChoice.Kind != "native" {
		t.Fatalf("fresh ModelID=%q contract=%+v", fresh.ModelID, fresh.ExecutionContract)
	}
	be, model := editCatalogueModel(t, srv, agent.Backend, agent.Model, "catalogue-edited")
	resumed, ae := srv.composeResumeSpec(agent, snapshotOf(agent, fresh), be, model)
	if ae != nil {
		t.Fatal(ae)
	}
	if resumed.ModelID != "" {
		t.Fatalf("resume ModelID=%q, want native", resumed.ModelID)
	}
}

// FS-08.R37 / TS-07.R19: an empty effort override falls through to the
// model's default; absent and nonempty overrides keep their meaning.
func TestContractEmptyEffortOverrideFallsThrough(t *testing.T) {
	srv, root, projectDir := federationServer(t)
	model := config.Model{DefaultEffort: "medium", Efforts: []string{"low", "medium", "high"}}
	backends, err := srv.configStore.ReadBackends()
	if err != nil {
		t.Fatal(err)
	}
	empty, high := "", "high"
	for name, tc := range map[string]struct {
		override *string
		explicit string
		want     string
	}{
		"empty":    {&empty, "", "medium"},
		"absent":   {nil, "", "medium"},
		"nonempty": {&high, "", "high"},
		"explicit": {&high, "low", "low"},
	} {
		bindClaudeSource(t, srv, root, projectDir, config.SourceOverrides{Effort: tc.override})
		_, fed, ae := srv.composeFederation(context.Background(), "claude", launchRequest{Project: "fed", Effort: tc.explicit},
			backends.Backends["claude"], config.Project{Cwd: projectDir}, "sonnet")
		if ae != nil {
			t.Fatal(ae)
		}
		got, ae := resolveEffort(tc.explicit, model, fed)
		if ae != nil || got != tc.want {
			t.Fatalf("%s: effort=%q (%v), want %q", name, got, ae, tc.want)
		}
	}
}

// TS-02.R53: a legacy row derives its choice only from unambiguous frozen
// evidence and otherwise refuses before any registration.
func TestContractLegacySnapshots(t *testing.T) {
	srv, _, projectDir := federationServer(t)
	backends, err := srv.configStore.ReadBackends()
	if err != nil {
		t.Fatal(err)
	}
	be := backends.Backends["claude"]
	key := be.DefaultModel
	model := be.Models[key]
	agent := state.Agent{AgentID: "a_legacy", Name: "Legacy", Role: "implementer", Project: "fed", Backend: "claude", Model: key, Interface: "chat"}
	doc := func(inherited, explicit bool) json.RawMessage {
		raw, _ := json.Marshal(launchConfigDoc{Version: 1, Binding: launchConfigBinding{BackendID: "claude", Provider: "claude-code", Mode: "linked"},
			NativeInherited: inherited, CurrentModelExplicit: explicit})
		return raw
	}
	for name, tc := range map[string]struct {
		launch json.RawMessage
		want   string
		refuse bool
	}{
		"unbound":   {nil, model.Model, false},
		"inherited": {doc(true, false), "", false},
		"switched":  {doc(true, true), model.Model, false},
		"ambiguous": {doc(false, false), "", true},
	} {
		snap := state.SessionSnapshot{AgentID: agent.AgentID, Backend: "claude", Model: key, Interface: "chat", Cwd: projectDir,
			LaunchConfig: tc.launch, LastSessionID: "native-1"}
		spec, ae := srv.composeResumeSpec(agent, snap, be, model)
		if tc.refuse {
			if ae == nil || ae.Code != runtime.CodeSourceInvalid {
				t.Fatalf("%s: want source_invalid refusal, got %v", name, ae)
			}
			if _, ok := srv.hookTokens[agent.AgentID]; ok {
				t.Fatalf("%s: refusal left a hook registration", name)
			}
			continue
		}
		if ae != nil {
			t.Fatalf("%s: %v", name, ae)
		}
		if spec.ModelID != tc.want || spec.ExecutionContract == nil {
			t.Fatalf("%s: ModelID=%q contract=%+v, want %q", name, spec.ModelID, spec.ExecutionContract, tc.want)
		}
		srv.teardownAgentRegistration(agent.AgentID)
	}
}

// FS-01.R53: a clone inherits its source's frozen choice rather than
// re-resolving the catalogue key it names.
func TestContractForkInheritsSourceChoice(t *testing.T) {
	srv, root, projectDir := federationServer(t)
	empty := ""
	bindClaudeSource(t, srv, root, projectDir, config.SourceOverrides{Model: &empty})
	fresh, agent, ae := srv.composeLaunch(context.Background(), launchRequest{Role: "implementer", Project: "fed"})
	if ae != nil {
		t.Fatal(ae)
	}
	snap := snapshotOf(agent, fresh)
	snap.SystemPrompt = "SOURCE_PERSONA"
	sentinel := "later-override"
	bindClaudeSource(t, srv, root, projectDir, config.SourceOverrides{Model: &sentinel})

	clone, _, ae := srv.composeLaunchWithOptions(context.Background(),
		launchRequest{Role: "implementer", Project: "fed", Backend: agent.Backend, Model: agent.Model},
		launchOptions{Source: &snap})
	if ae != nil {
		t.Fatal(ae)
	}
	if clone.ModelID != "" || clone.ExecutionContract.ModelChoice.Kind != "native" {
		t.Fatalf("clone ModelID=%q contract=%+v, want inherited native", clone.ModelID, clone.ExecutionContract)
	}
	if clone.SystemPrompt != "SOURCE_PERSONA" {
		t.Fatalf("clone persona = %q, want the source's frozen persona", clone.SystemPrompt)
	}
}
