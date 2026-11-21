package runtime

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/state"
)

// beginPathAgent is one idle chat agent on the hold_turn scenario with a launch
// generation, so every begin writes its per-turn row (a spec with no generation
// writes none). A turn it starts stays open until release is called; hold
// closes the gate again for the next turn.
type beginPathAgent struct {
	c          *ChatRuntime
	r          *Registry
	id         string
	ch         <-chan Event
	holdFile   string
	promptLog  string
	release    func()
	hold       func()
	generation string
}

// stubTaskTurn stands in for the server's assignment preparer: it supplies a
// fixed prompt and confirms nothing durable.
func stubTaskTurn(prompt string) TaskTurnPreparer {
	return func(string, string, string, bool) (TaskTurn, error) {
		return TaskTurn{Prompt: prompt, Confirm: func() error { return nil }}, nil
	}
}

func newBeginPathAgent(t *testing.T, env ...string) *beginPathAgent {
	t.Helper()
	c, spec := newChatTest(t, "hold_turn")
	c.SetTaskTurnPreparer(stubTaskTurn("do the work"))
	spec.Generation = "g"
	dir := t.TempDir()
	a := &beginPathAgent{c: c, id: spec.Agent.AgentID, generation: "g",
		holdFile: filepath.Join(dir, "hold"), promptLog: filepath.Join(dir, "prompts.log")}
	spec.Env = append(spec.Env, "FAKEACP_HOLD_FILE="+a.holdFile, "FAKEACP_PROMPT_LOG="+a.promptLog)
	spec.Env = append(spec.Env, env...)
	ctx := context.Background()
	h, err := c.Start(ctx, spec)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	a.release = func() { _ = os.WriteFile(a.holdFile, []byte("go"), 0o644) }
	a.hold = func() { _ = os.Remove(a.holdFile) }
	t.Cleanup(func() {
		a.release()
		_ = c.Stop(ctx, h.AgentID)
	})
	ch, unsub, err := c.Subscribe(h.AgentID)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	t.Cleanup(unsub)
	a.ch = ch
	// The registry fronts the same runtime so the launched-task and settled-chat
	// entries, which live on Registry, reach the real begin function.
	a.r = NewRegistry(c.store)
	a.r.chat = c
	a.r.rtByAgent[a.id] = c
	a.r.generationByAgent[a.id] = a.generation
	return a
}

func (a *beginPathAgent) turn(t *testing.T) state.ConversationTurn {
	t.Helper()
	row, err := a.c.store.ReadConversationTurn(a.id, a.generation)
	if err != nil {
		t.Fatalf("ReadConversationTurn: %v", err)
	}
	return row
}

// completeFirstTurn runs one person turn to completion and returns its row, so a
// path that needs a settled or busy conversation starts from a real one.
func (a *beginPathAgent) completeFirstTurn(t *testing.T) state.ConversationTurn {
	t.Helper()
	if err := a.c.SendPrompt(context.Background(), a.id, "first"); err != nil {
		t.Fatalf("SendPrompt first: %v", err)
	}
	waitForChunk(t, a.ch, "working")
	first := a.turn(t)
	a.release()
	waitForRuntimeStatus(t, a.c, a.id, "idle")
	return first
}

func waitForTurnState(t *testing.T, a *beginPathAgent, turnID, want string) state.ConversationTurn {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		row, err := a.c.store.ReadConversationTurn(a.id, a.generation)
		if err == nil && row.TurnID == turnID && row.State == want {
			return row
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for turn %q state %q (last %+v err %v)", turnID, want, row, err)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// TS-01.R45 / EXEC-6(b) — every entry path that sends a conversation turn to the
// provider reaches the one begin function, which records the turn's generation
// and origin in the per-turn row, and the one end function settles that row.
// Expected origins are the documented ones: a person's prompt (including a held
// successor and a promoted Steer fallback) is "person"; the launched-task first
// prompt and dependency activation are "task" carrying the task id; mail is
// "mail" carrying the activation id; the settled-chat continuation is
// "owner_continuation" carrying the settled turn; quota and room activations are
// "quota" / "room".
//
// Quota and room activation origins are written by their domain's own
// pre-side-effect callback (state.AdmitQuotaTurn, state.BeginThinkTankAttempt),
// whose setup is a server/state concern. Those rows drive the real
// StartActivation with a callback that records the documented origin and prove
// the runtime half: the begin function keeps that row, runs it, and ends it.
func TestBeginTurnRecordsOriginForEveryEntryPath(t *testing.T) {
	ctx := context.Background()
	type outcome struct {
		kind state.TurnOriginKind
		id   string
	}
	rows := []struct {
		name string
		env  []string
		// drive starts the path under test and returns the expected origin and the
		// id of the turn it began (empty means "the newest row").
		drive func(t *testing.T, a *beginPathAgent) (want outcome, turnID string)
	}{
		{
			name: "send",
			drive: func(t *testing.T, a *beginPathAgent) (outcome, string) {
				if err := a.c.SendPrompt(ctx, a.id, "hello"); err != nil {
					t.Fatalf("SendPrompt: %v", err)
				}
				return outcome{kind: state.TurnOriginPerson}, ""
			},
		},
		{
			name: "send_or_hold_idle",
			drive: func(t *testing.T, a *beginPathAgent) (outcome, string) {
				if held, err := a.c.SendPromptOrHold(ctx, a.id, "hello"); err != nil || held {
					t.Fatalf("SendPromptOrHold idle = held %v err %v, want sent", held, err)
				}
				return outcome{kind: state.TurnOriginPerson}, ""
			},
		},
		{
			name: "held_successor",
			drive: func(t *testing.T, a *beginPathAgent) (outcome, string) {
				if err := a.c.SendPrompt(ctx, a.id, "first"); err != nil {
					t.Fatalf("SendPrompt first: %v", err)
				}
				waitForChunk(t, a.ch, "working")
				first := a.turn(t)
				if held, err := a.c.SendPromptOrHold(ctx, a.id, "second"); err != nil || !held {
					t.Fatalf("SendPromptOrHold while busy = held %v err %v, want held", held, err)
				}
				a.release()
				waitForPrompts(t, a.promptLog, 2)
				// The successor replaces the first turn's row; wait for it.
				deadline := time.Now().Add(5 * time.Second)
				for {
					if row := a.turn(t); row.TurnID != first.TurnID {
						return outcome{kind: state.TurnOriginPerson}, row.TurnID
					}
					if time.Now().After(deadline) {
						t.Fatal("held successor never began a new turn row")
					}
					time.Sleep(5 * time.Millisecond)
				}
			},
		},
		{
			name: "steer_prompt_required_fallback",
			env: []string{
				"FAKEACP_STEERING=1", "FAKEACP_STEER_OUTCOME=promptRequired",
			},
			drive: func(t *testing.T, a *beginPathAgent) (outcome, string) {
				if err := a.c.SendPrompt(ctx, a.id, "first"); err != nil {
					t.Fatalf("SendPrompt first: %v", err)
				}
				waitForChunk(t, a.ch, "working")
				first := a.turn(t)
				got, err := a.c.Steer(ctx, a.id, "late correction")
				if err != nil || got != SteerNewTurn {
					t.Fatalf("Steer = %q err %v, want %q", got, err, SteerNewTurn)
				}
				waitForPrompts(t, a.promptLog, 2)
				row := a.turn(t)
				if row.TurnID == first.TurnID {
					t.Fatalf("Steer fallback reused the first turn row %+v", row)
				}
				return outcome{kind: state.TurnOriginPerson}, row.TurnID
			},
		},
		{
			name: "launched_task_first_prompt",
			drive: func(t *testing.T, a *beginPathAgent) (outcome, string) {
				if err := a.r.StartLaunchedTask(a.id, "task_launch"); err != nil {
					t.Fatalf("StartLaunchedTask: %v", err)
				}
				return outcome{kind: state.TurnOriginTask, id: "task_launch"}, ""
			},
		},
		{
			name: "settled_chat_continuation",
			drive: func(t *testing.T, a *beginPathAgent) (outcome, string) {
				first := a.completeFirstTurn(t)
				a.hold()
				err := a.r.ContinueSettledChat(a.id, a.generation, func(string) (string, error) { return "continue", nil })
				if err != nil {
					t.Fatalf("ContinueSettledChat: %v", err)
				}
				return outcome{kind: state.TurnOriginOwnerContinuation, id: first.TurnID}, ""
			},
		},
		{
			name: "activation_mail",
			drive: func(t *testing.T, a *beginPathAgent) (outcome, string) {
				if _, err := a.c.store.InsertMessage(state.Message{
					FromAgent: "a_sender", FromAddress: "peer@test", FromName: "Nova",
					ToAgent: a.id, Subject: "Review", Body: "please look", Wake: true,
				}); err != nil {
					t.Fatalf("InsertMessage: %v", err)
				}
				pending, err := a.c.store.PendingActivations(state.ActivationKindMail, a.id, 5)
				if err != nil || len(pending) != 1 {
					t.Fatalf("pending mail activations = %v err %v, want one", pending, err)
				}
				token, ok, err := a.c.store.ClaimMailActivation(pending[0].ActivationID)
				if err != nil || !ok {
					t.Fatalf("ClaimMailActivation = %v err %v", ok, err)
				}
				started, err := a.c.StartActivation(ctx, a.id, state.ActivationKindMail, func(turnID string) error {
					_, err := a.c.store.AttemptActivation(state.ActivationKindMail, pending[0].ActivationID, token, turnID, a.generation)
					return err
				})
				if err != nil || !started {
					t.Fatalf("StartActivation mail = %v err %v, want started", started, err)
				}
				return outcome{kind: state.TurnOriginMail, id: pending[0].ActivationID}, ""
			},
		},
		{
			name: "activation_dependency",
			drive: func(t *testing.T, a *beginPathAgent) (outcome, string) {
				act, err := a.c.store.EnsurePendingDependencyActivation(a.id, "task_dep")
				if err != nil {
					t.Fatalf("EnsurePendingDependencyActivation: %v", err)
				}
				token, ok, err := a.c.store.ClaimActivation(state.ActivationKindDependency, act.ActivationID)
				if err != nil || !ok {
					t.Fatalf("ClaimActivation = %v err %v", ok, err)
				}
				started, err := a.c.StartActivation(ctx, a.id, state.ActivationKindDependency, func(turnID string) error {
					_, err := a.c.store.AttemptActivation(state.ActivationKindDependency, act.ActivationID, token, turnID, a.generation)
					return err
				})
				if err != nil || !started {
					t.Fatalf("StartActivation dependency = %v err %v, want started", started, err)
				}
				return outcome{kind: state.TurnOriginTask, id: "task_dep"}, ""
			},
		},
	}
	// Quota and room kinds: the callback owns the origin (see the test comment).
	for _, k := range []struct {
		name, kind string
		origin     outcome
	}{
		{"activation_quota_continue", state.ActivationKindQuotaContinue, outcome{kind: state.TurnOriginQuota, id: "episode_1"}},
		{"activation_quota_think_tank", state.ActivationKindQuotaThinkTank, outcome{kind: state.TurnOriginQuota, id: "episode_2"}},
		{"activation_think_tank", state.ActivationKindThinkTank, outcome{kind: state.TurnOriginRoom, id: "attempt_1"}},
	} {
		k := k
		rows = append(rows, struct {
			name  string
			env   []string
			drive func(t *testing.T, a *beginPathAgent) (outcome, string)
		}{
			name: k.name,
			drive: func(t *testing.T, a *beginPathAgent) (outcome, string) {
				started, err := a.c.StartActivation(ctx, a.id, k.kind, func(turnID string) error {
					return a.c.store.BeginConversationTurn(a.id, turnID, state.TurnOrigin{Generation: a.generation, Kind: k.origin.kind, ID: k.origin.id})
				})
				if err != nil || !started {
					t.Fatalf("StartActivation %s = %v err %v, want started", k.kind, started, err)
				}
				return k.origin, ""
			},
		})
	}

	for _, tc := range rows {
		t.Run(tc.name, func(t *testing.T) {
			a := newBeginPathAgent(t, tc.env...)
			want, turnID := tc.drive(t, a)
			row := a.turn(t)
			if turnID != "" && row.TurnID != turnID {
				t.Fatalf("newest row = %+v, want turn %q", row, turnID)
			}
			if row.Generation != "g" || row.Kind != want.kind || row.ID != want.id {
				t.Fatalf("row origin = {%q %q %q}, want {g %q %q}", row.Generation, row.Kind, row.ID, want.kind, want.id)
			}
			// A held successor and a Steer fallback run after the original
			// prompt was released, so they can end before they are observable;
			// every other path holds its turn open until released.
			if tc.name != "held_successor" && tc.name != "steer_prompt_required_fallback" {
				if row.State != "running" || row.Outcome != "" {
					t.Fatalf("in-flight row = %+v, want running with no outcome", row)
				}
			}
			a.release()
			ended := waitForTurnState(t, a, row.TurnID, "ended")
			if ended.Outcome != "completed" {
				t.Fatalf("ended row = %+v, want outcome completed", ended)
			}
			if ended.Kind != want.kind || ended.ID != want.id || ended.Generation != "g" {
				t.Fatalf("ending changed the origin: %+v, want {g %q %q}", ended, want.kind, want.id)
			}
		})
	}
}
