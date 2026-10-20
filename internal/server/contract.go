package server

import (
	"encoding/json"

	"github.com/AsaphNoam/Chuck/internal/config"
	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
)

// launchModelChoice is the one precedence seam for a newly selected provider
// model (TS-01.R40, TS-07.R19): an explicit launch model, then a bound source
// override, then native inheritance for a bound source, then the catalogue.
func launchModelChoice(model config.Model, explicit bool, fed *federationModel) state.ModelChoice {
	if !explicit && fed != nil {
		switch {
		case fed.override != nil:
			return explicitModelChoice(*fed.override)
		case fed.inherit:
			return state.ModelChoice{Kind: state.ModelChoiceNative}
		}
	}
	return explicitModelChoice(model.Model)
}

func explicitModelChoice(id string) state.ModelChoice {
	if id == "" {
		return state.ModelChoice{Kind: state.ModelChoiceNative}
	}
	return state.ModelChoice{Kind: state.ModelChoiceExplicit, ID: id}
}

// newExecutionContract freezes a newly derived selection for agent.
func newExecutionContract(agent state.Agent, backendType string, choice state.ModelChoice, skip bool, setup state.SetupChoice) *state.ExecutionContract {
	if setup.Source == "" {
		setup.Source = "native"
	}
	home := "personal"
	if backendType == "codex-acp" {
		home = "chuck_private" // codexHomeEnv's owned private store
	}
	return &state.ExecutionContract{
		Version: state.ExecutionContractVersion, Backend: agent.Backend, Model: agent.Model,
		BackendType: backendType, ModelChoice: choice, Effort: agent.Effort, Fast: agent.Fast,
		SkipPerms: skip, Setup: setup, AuthOrigin: "personal", NativeHome: home,
	}
}

// continuedContract is the selection an ordinary continuation (resume, wake,
// same-identity switch, rollback) realizes: the frozen contract unchanged
// except the explicitly applied effort/fast controls. explicit (a caller-named
// model) or a changed backend/model identity derives a new explicit catalogue
// choice. A legacy row is derived only from unambiguous frozen evidence; an
// undecidable one refuses so the caller uses explicit refresh (TS-02.R53).
func continuedContract(snap state.SessionSnapshot, agent state.Agent, be config.Backend, model config.Model, explicit bool) (*state.ExecutionContract, *runtime.APIError) {
	frozen := snap.ExecutionContract
	backend, modelKey := snap.Backend, snap.Model
	setup := legacySetup(snap.LaunchConfig)
	if frozen != nil {
		backend, modelKey, setup = frozen.Backend, frozen.Model, frozen.Setup
	}
	if !explicit && backend == agent.Backend && modelKey == agent.Model {
		if frozen != nil {
			c := *frozen
			c.BackendType, c.Effort, c.Fast = be.Type, agent.Effort, agent.Fast
			return &c, nil
		}
		choice, ok := legacyModelChoice(snap.LaunchConfig, model)
		if !ok {
			return nil, apiError(runtime.CodeSourceInvalid,
				"this conversation predates recorded model choices and its source model cannot be determined; resume with the latest setup or start a new conversation")
		}
		return newExecutionContract(agent, be.Type, choice, snap.SkipPermissions, setup), nil
	}
	if backend != agent.Backend {
		setup = state.SetupChoice{}
	}
	return newExecutionContract(agent, be.Type, explicitModelChoice(model.Model), snap.SkipPermissions, setup), nil
}

// legacyModelChoice derives a pre-contract row's choice from its frozen launch
// object. An unbound launch and an explicit switch both chose the catalogue
// key, and native inheritance is recorded; a bound launch that did neither may
// have sent a source override or an explicit request, which nothing proves.
func legacyModelChoice(launchConfig json.RawMessage, model config.Model) (state.ModelChoice, bool) {
	if len(launchConfig) == 0 {
		return explicitModelChoice(model.Model), true
	}
	var doc launchConfigDoc
	if err := json.Unmarshal(launchConfig, &doc); err != nil {
		return state.ModelChoice{}, false
	}
	switch {
	case doc.CurrentModelExplicit:
		return explicitModelChoice(model.Model), true
	case doc.NativeInherited:
		return state.ModelChoice{Kind: state.ModelChoiceNative}, true
	}
	return state.ModelChoice{}, false
}

func legacySetup(launchConfig json.RawMessage) state.SetupChoice {
	var doc launchConfigDoc
	if len(launchConfig) == 0 || json.Unmarshal(launchConfig, &doc) != nil {
		return state.SetupChoice{}
	}
	return boundSetup(doc.Binding)
}

func boundSetup(b launchConfigBinding) state.SetupChoice {
	return state.SetupChoice{Source: "bound", Provider: b.Provider, Profile: b.Profile, Mode: b.Mode}
}
