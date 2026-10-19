package runtime

import (
	"context"
	"errors"
	"sync"
	"testing"
)

// exitingRuntime simulates a terminal runtime whose process exits after the
// handshake but before Registry.Launch commits: Start reports the exit
// through the registry's onExit callback, then returns success.
type exitingRuntime struct {
	notImplementedRuntime
	mu        sync.Mutex
	onExit    func(string, string)
	exitFirst bool
	stopped   []string
}

func (e *exitingRuntime) SetOnExit(fn func(string, string)) { e.onExit = fn }

func (e *exitingRuntime) Start(_ context.Context, spec LaunchSpec) (*Handle, error) {
	e.mu.Lock()
	exit := e.exitFirst
	e.mu.Unlock()
	if exit {
		e.onExit(spec.Agent.AgentID, spec.Generation)
	}
	return &Handle{AgentID: spec.Agent.AgentID}, nil
}

func (e *exitingRuntime) Resume(ctx context.Context, spec LaunchSpec, _ string) (*Handle, error) {
	return e.Start(ctx, spec)
}

func (e *exitingRuntime) Stop(_ context.Context, agentID string) error {
	e.mu.Lock()
	e.stopped = append(e.stopped, agentID)
	e.mu.Unlock()
	return nil
}

// ASRC-4 / TS-01.R41: an exit inside the launch window wins against the later
// commit. The launch fails, ownership and generation stay clear, the runtime
// drops its dead handle, and the launcher (not the exit hook) owns rollback,
// so the same agent can start again. A late exit of an older generation never
// tears down the newer committed claim.
func TestExitBeforeCommitFailsLaunchAndFreesClaim(t *testing.T) {
	for _, op := range []string{"launch", "resume"} {
		t.Run(op, func(t *testing.T) {
			r := NewRegistry(nil)
			term := &exitingRuntime{notImplementedRuntime: notImplementedRuntime{name: "terminal"}, exitFirst: true}
			r.SetTerminalRuntime(term)
			hooked := 0
			r.SetExitHook(func(string, string, string) { hooked++ })
			start := r.Launch
			if op == "resume" {
				start = r.Resume
			}
			spec := LaunchSpec{Generation: "g1"}
			spec.Agent.AgentID, spec.Agent.Interface = "a_window", "terminal"

			if _, err := start(context.Background(), spec); !errors.Is(err, ErrExitedDuringStart) {
				t.Fatalf("%s = %v, want ErrExitedDuringStart", op, err)
			}
			if r.Owns("a_window") || r.Generation("a_window") != "" {
				t.Fatal("dead runtime was committed as owned")
			}
			if len(term.stopped) != 1 {
				t.Fatalf("runtime residue not dropped: stops = %v", term.stopped)
			}
			if hooked != 0 {
				t.Fatalf("exit hook tore down the in-flight launch %d times", hooked)
			}

			term.mu.Lock()
			term.exitFirst = false
			term.mu.Unlock()
			spec.Generation = "g2"
			if _, err := start(context.Background(), spec); err != nil {
				t.Fatalf("restart after exit: %v", err)
			}
			term.onExit("a_window", "g1")
			if !r.Owns("a_window") || r.Generation("a_window") != "g2" || hooked != 0 {
				t.Fatal("late prior-generation exit tore down the new claim")
			}
		})
	}
}
