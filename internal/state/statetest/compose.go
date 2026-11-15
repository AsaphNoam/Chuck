// Package statetest holds read-only assertions shared by runtime and server
// tests. It is imported only from _test.go files.
package statetest

import (
	"errors"
	"fmt"

	"github.com/AsaphNoam/Chuck/internal/state"
)

// ChatStatusMismatches returns, for every chat agent whose stored status row
// differs from TS-01.R46's composition of its stored inputs, one description.
// The oracle is the durable process and turn rows, not the writer that chose
// the state. skip names agents whose in-memory turn is still settling; their
// inputs and status are not yet written together.
//
// Inputs: no running row means the process ended, which composes only `done`
// (Stop or reconciled exit) or `error` (unexpected exit). A running row is a
// live process; its generation's turn row (an older generation's row reads as
// none) and the pending-permission input, which only the runtime holds and is
// therefore taken as "the stored state says waiting_input", compose the rest.
func ChatStatusMismatches(st *state.Store, skip func(agentID string) bool) ([]string, error) {
	agents, err := st.ListAgents()
	if err != nil {
		return nil, err
	}
	var out []string
	for _, a := range agents {
		if a.Interface != "chat" || skip != nil && skip(a.AgentID) {
			continue
		}
		status, err := st.ReadStatus(a.AgentID)
		if errors.Is(err, state.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		running, err := st.ReadRunning(a.AgentID)
		if errors.Is(err, state.ErrNotFound) {
			if status.State != "done" && status.State != "error" {
				out = append(out, fmt.Sprintf("%s: stored %q with no live process, want done or error", a.AgentID, status.State))
			}
			continue
		}
		if err != nil {
			return nil, err
		}
		if running.Generation == "" {
			continue // a generation-less fixture launch records no turn row to compose from
		}
		turn, err := st.ReadConversationTurn(a.AgentID, running.Generation)
		if errors.Is(err, state.ErrNotFound) {
			turn = state.ConversationTurn{}
		} else if err != nil {
			return nil, err
		}
		want := state.ComposeChatStatus(state.ChatProcessLive, turn, status.State == "waiting_input")
		if status.State != want {
			out = append(out, fmt.Sprintf("%s: stored %q, composition of live generation %q turn %+v is %q", a.AgentID, status.State, running.Generation, turn, want))
		}
	}
	return out, nil
}
