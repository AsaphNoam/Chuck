package server

import (
	"context"
	"testing"

	"github.com/AsaphNoam/Chuck/internal/config"
	"github.com/AsaphNoam/Chuck/internal/index"
	"github.com/AsaphNoam/Chuck/internal/runtime"
)

// FS-01.R53 / TS-04.R97: a stopped agent's quota check probes the frozen
// continuation selection, not the catalogue key's current provider id.
func TestStoppedQuotaCheckUsesFrozenModel(t *testing.T) {
	srv, root, projectDir := federationServer(t)
	sentinel := "frozen-provider-model"
	bindClaudeSource(t, srv, root, projectDir, config.SourceOverrides{Model: &sentinel})
	spec, agent, ae := srv.composeLaunch(context.Background(), launchRequest{Role: "implementer", Project: "fed"})
	if ae != nil {
		t.Fatal(ae)
	}
	srv.teardownAgentRegistration(agent.AgentID)
	if err := srv.stateStore.WriteAgent(agent); err != nil {
		t.Fatal(err)
	}
	ix := index.New(srv.stateStore.DB())
	if err := ix.UpsertSessionMeta(agent.AgentID, runtime.NewSessionMeta(spec, "native-1")); err != nil {
		t.Fatal(err)
	}
	check, err := srv.quotaCheckSpec(agent.AgentID)
	if err != nil {
		t.Fatal(err)
	}
	if check.ModelID != sentinel {
		t.Fatalf("stopped quota check model=%q; continuation model=%q", check.ModelID, sentinel)
	}
}

// FS-01.R53 / TS-02.R53: a clone copies its source's frozen permission choice
// even after the persona default changes.
func TestForkKeepsSourcePermissions(t *testing.T) {
	srv, _, _ := federationServer(t)
	spec, agent, ae := srv.composeLaunch(context.Background(), launchRequest{Role: "implementer", Project: "fed"})
	if ae != nil {
		t.Fatal(ae)
	}
	srv.teardownAgentRegistration(agent.AgentID)
	snap := snapshotOf(agent, spec)
	snap.SkipPermissions = spec.SkipPerms
	role, err := srv.configStore.ReadRole("implementer")
	if err != nil {
		t.Fatal(err)
	}
	changed := !spec.SkipPerms
	role.SkipPermissions = &changed
	if err := srv.configStore.WriteRole("implementer", role); err != nil {
		t.Fatal(err)
	}
	clone, _, ae := srv.composeLaunchWithOptions(context.Background(),
		launchRequest{Role: "implementer", Project: "fed", Backend: agent.Backend, Model: agent.Model},
		launchOptions{Source: &snap, Fork: &runtime.ForkPlan{SourceSessionID: "native-1"}})
	if ae != nil {
		t.Fatal(ae)
	}
	if clone.SkipPerms != spec.SkipPerms || clone.ExecutionContract.SkipPerms != spec.ExecutionContract.SkipPerms {
		t.Fatalf("clone permission choice=%v contract=%v; source=%v", clone.SkipPerms, clone.ExecutionContract.SkipPerms, spec.SkipPerms)
	}
}
