package runtime

import (
	"testing"

	"github.com/AsaphNoam/Chuck/internal/state"
)

// EDIT-EDGE-2 (TS-01.R51): a staged Edit candidate's live usage stays off
// the source's status row until PublishStaged.
func TestStagedEditCandidateDoesNotRepublishLiveUsage(t *testing.T) {
	st, err := state.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	if err := st.WriteAgent(state.Agent{AgentID: "a_edit", Name: "edit", Project: "p", Role: "impl"}); err != nil {
		t.Fatal(err)
	}
	if err := st.WriteStatus(state.Status{AgentID: "a_edit", State: "stopped"}); err != nil {
		t.Fatal(err)
	}
	c := NewChatRuntime(st)
	as := &agentState{agentID: "a_edit", staged: &stagedStart{}, openTasks: 2,
		context: contextReading{pct: 0.5}}

	c.republishLive(as)
	if got, _ := st.ReadStatus("a_edit"); got.BackgroundRunning != 0 || got.ContextPct != 0 {
		t.Fatalf("staged candidate wrote the source's status: %+v", got)
	}

	as.staged = nil // published
	c.republishLive(as)
	if got, _ := st.ReadStatus("a_edit"); got.BackgroundRunning != 2 || got.ContextPct != 0.5 {
		t.Fatalf("published candidate did not republish: %+v", got)
	}
}
