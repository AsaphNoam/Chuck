package server

import (
	"context"
	"errors"
	"github.com/AsaphNoam/Chuck/internal/backend/providerexec"
	"github.com/AsaphNoam/Chuck/internal/config"
	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
)

// recoverIncompleteLaunches runs before request handling and background starts.
func (s *Server) recoverIncompleteLaunches() {
	if err := s.stateStore.RestoreInterruptedSwitches(); err != nil {
		s.log.Warn("recover interrupted switches", "err", err)
	}
	ids, err := s.stateStore.AgentsWithoutSession()
	if err != nil {
		s.log.Warn("read incomplete launches", "err", err)
		return
	}
	for _, id := range ids {
		if row, err := s.stateStore.ReadRunning(id); err == nil {
			// A legacy live process can predate session snapshots. Preserve its
			// ownership and any process whose identity cannot be proved.
			if runtime.ClassifyProcess(row.PID, row.ProcessBirth) != runtime.ProcessExited || runtime.LeaderlessGroup(row.PID) {
				continue
			}
		} else if !errors.Is(err, state.ErrNotFound) {
			s.log.Warn("read incomplete launch process", "agent_id", id, "err", err)
			continue
		}
		s.teardownAgentRegistration(id)
		if err := s.stateStore.DeleteAgent(id); err != nil {
			s.log.Warn("delete incomplete launch", "agent_id", id, "err", err)
		}
	}
}

// startRuntimeClaimed owns registration through the applied-field write.
// The origin retains its admission checks and holds its claim across this step
// and any origin-specific work before or after it.
func (s *Server) startRuntimeClaimed(ctx context.Context, fresh bool, compose func() (runtime.LaunchSpec, *runtime.APIError), prepare func(*runtime.LaunchSpec) *runtime.APIError) (*runtime.APIError, error) {
	spec, ae := compose()
	if ae != nil {
		return ae, nil // the composer owns partial registration
	}
	id := spec.Agent.AgentID
	if prepare != nil {
		if ae := prepare(&spec); ae != nil {
			s.teardownAgentRegistration(id)
			return ae, nil
		}
	}
	start, startError := s.registry.Resume, resumeStartError
	if fresh {
		start, startError = s.registry.Launch, launchStartError
	}
	handle, err := start(ctx, spec)
	if err != nil {
		if fresh {
			_ = s.stateStore.DeleteRunning(id)
			_ = s.stateStore.DeleteStatus(id)
			_ = s.stateStore.DeleteAgent(id)
		}
		s.teardownAgentRegistration(id)
		return startError(err), nil
	}
	spec.Agent.Fast = handle.Fast
	if err := s.stateStore.WriteAgentRuntimeIdentity(spec.Agent); err != nil {
		return apiError(runtime.CodeInternal, "write applied session configuration: "+err.Error()), s.stopFailedStart(id)
	}
	return nil, nil
}

type startSpec struct {
	agent       state.Agent
	session     state.SessionSnapshot
	backend     config.Backend
	model       config.Model
	provider    providerexec.Selection
	providerEnv map[string]string
	resourceDir string
	generation  string
	driver      string
	fork        *runtime.ForkPlan
}

// buildLaunchSpec registers the process and assembles the fresh or frozen
// inputs selected by its origin.
func (s *Server) buildLaunchSpec(in startSpec) (runtime.LaunchSpec, *runtime.APIError) {
	agent, snap, backend, model := in.agent, in.session, in.backend, in.model
	token := mintHookToken()
	generation := in.generation
	if generation == "" {
		generation = token
	}
	s.rememberHookToken(agent.AgentID, token)
	mcp, err := s.registerMessagingMCP(agent, generation)
	if err != nil {
		s.teardownAgentRegistration(agent.AgentID)
		return runtime.LaunchSpec{}, apiError(runtime.CodeInternal, err.Error())
	}
	args, err := s.composeHookRegistration(agent, backend.Type)
	if err != nil {
		s.teardownAgentRegistration(agent.AgentID)
		return runtime.LaunchSpec{}, apiError(runtime.CodeInternal, err.Error())
	}
	spec := runtime.LaunchSpec{
		Agent:              agent,
		Generation:         generation,
		Cwd:                snap.Cwd,
		WorkspaceID:        snap.WorkspaceID,
		AddDirs:            snap.AddDirs,
		SystemPrompt:       snap.SystemPrompt,
		HistoryPrimer:      snap.HistoryPrimer,
		ChuckAware:         &snap.ChuckAware,
		BackendType:        backend.Type,
		ModelID:            snap.ExecutionContract.ProviderModelID(),
		Effort:             snap.Effort,
		Fast:               snap.Fast,
		Driver:             in.driver,
		Env:                composeChildEnv(backend.Type, s.configStore.Home(), backend.Env, model.Env, s.hookEnv(agent, token), projectResourcesEnv(in.resourceDir), in.providerEnv),
		QuotaCheckEnv:      s.quotaCheckEnv(backend, model, in.providerEnv),
		ProviderExecutable: in.provider.Path,
		ProviderSource:     in.provider.Source,
		SkipPerms:          snap.SkipPermissions,
		HookToken:          token,
		MCPServers:         []runtime.MCPServerSpec{mcp},
		ExtraArgs:          args,
		LastSessionID:      snap.LastSessionID,
		LastContextPct:     snap.LastContextPct,
		LastContext:        snap.LastContext,
		LaunchConfig:       snap.LaunchConfig,
		ExecutionContract:  snap.ExecutionContract,
		Fork:               in.fork,
	}
	return s.applyKnowledgeOverlay(spec), nil
}
