package runtime

import (
	"errors"

	"github.com/AsaphNoam/Chuck/internal/state"
)

// ActivationKind is the code-owned contract for one host-owned turn kind: the
// fixed instruction the provider receives and the status the dashboard shows
// while that turn runs. It is the only activation data that reaches the model
// (TS-01.R21), and it lives in one table rather than as literals at the call
// site so a new kind is a row here instead of a branch inside the runtime, and
// so no kind can inherit another's instruction by accident (TS-10.R5, FS-16.R26,
// INV §2).
type ActivationKind struct {
	// Instruction is the entire prompt. It names the tool the agent should call
	// and carries no payload from the source domain.
	Instruction string
	// StatusDetail and LastTrace are the bounded, in-vocabulary status the card
	// shows for the turn (INV §8).
	StatusDetail string
	LastTrace    string
}

// TaskTurn is a task assignment prepared for one admitted turn. Prompt is the
// whole provider text; Confirm binds the task to this execution and runs after
// every runtime admission check but before the provider goroutine exists, so no
// tool callback can see an unbound execution (FS-16.R61, TS-10.R64). A Confirm
// error sends nothing.
type TaskTurn struct {
	Prompt  string
	Confirm func() error
}

// TaskTurnPreparer builds the current assignment for the task this agent's
// admitted turn executes, including a quota or stalled-owner continuation.
// launched is true only for a task's own first launch prompt; every other route
// is an existing-agent assignment or continuation.
type TaskTurnPreparer func(agentID, generation, turnID string, launched bool) (TaskTurn, error)

// errNoTaskPreparer refuses a task turn rather than sending an assignment
// nobody built or confirmed (INV §24).
var errNoTaskPreparer = errors.New("runtime: no task turn preparer")

// activationKinds is the closed registry. An activation whose kind is absent
// here cannot start, so an unregistered kind fails loudly at its first use
// rather than prompting a model with another kind's instruction.
var activationKinds = map[string]ActivationKind{
	state.ActivationKindQuotaThinkTank: {
		Instruction:  "Continue your quota-interrupted Think Tank opportunity. Call read_think_tank to read the current room, following its cursor until the read is complete. Submit once using this turn's new read_receipt and authority; do not reuse or publish an earlier failed turn's staged submission.",
		StatusDetail: "continuing a Think Tank turn", LastTrace: "QuotaContinuation",
	},
	state.ActivationKindQuotaContinue: {
		Instruction:  "Continue the work interrupted by the provider quota from this conversation's existing history. Read the current state of any owned work first. Do not replay already executed operations or repeat accepted results.",
		StatusDetail: "continuing after quota interruption", LastTrace: "QuotaContinuation",
	},
	state.ActivationKindMail: {
		Instruction:  "Handle the attributed peer mail supplied with this turn.",
		StatusDetail: "checking messages",
		LastTrace:    "MailActivation",
	},
	// Dependent work has its own status. Its prompt is not a fixed instruction:
	// the task turn preparer supplies the current assignment for the admitted
	// execution, while the activation row stays a payload-free signal
	// (FS-16.R60, TS-01.R52).
	state.ActivationKindDependency: {
		StatusDetail: "starting assigned task",
		LastTrace:    "TaskActivation",
	},
	// A room turn names only the room tools. Goal, peer contributions, role and
	// remaining ceiling are pulled through read_think_tank as data, never
	// carried in the prompt (TS-14.R3, FS-21.R16).
	state.ActivationKindThinkTank: {
		Instruction: "It is your turn in a Think Tank. Call read_think_tank to read the goal, your role, " +
			"its guidance and the new discussion, following its cursor until the read is complete. Then call " +
			"submit_think_tank_turn once with the read_receipt and the disposition the guidance offers.",
		StatusDetail: "taking a Think Tank turn",
		LastTrace:    "ThinkTankActivation",
	},
}

// LookupActivationKind returns the contract for a kind and whether it exists.
func LookupActivationKind(kind string) (ActivationKind, bool) {
	k, ok := activationKinds[kind]
	return k, ok
}

// ActivationKinds returns every registered kind name, so a test can assert the
// registry and the state layer's kind vocabulary have not drifted apart.
func ActivationKinds() []string {
	kinds := make([]string, 0, len(activationKinds))
	for kind := range activationKinds {
		kinds = append(kinds, kind)
	}
	return kinds
}
