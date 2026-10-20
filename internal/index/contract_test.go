package index

import (
	"testing"

	"github.com/AsaphNoam/Chuck/internal/state"
)

// TS-02.R53: the execution contract survives session_meta persistence; a
// later record without one (an older writer or transcript) keeps the stored
// value, and a legacy row reads as unknown.
func TestExecutionContractPersistsThroughSessionMeta(t *testing.T) {
	st, _ := openTestDB(t)
	ix := New(st.DB())

	legacy := meta()
	if err := ix.UpsertSessionMeta("a_legacy", legacy); err != nil {
		t.Fatal(err)
	}
	snap, err := st.ReadSession("a_legacy")
	if err != nil {
		t.Fatal(err)
	}
	if snap.ExecutionContract != nil {
		t.Fatalf("legacy row contract = %+v, want unknown", snap.ExecutionContract)
	}

	m := meta()
	m.ExecutionContract = &state.ExecutionContract{
		Version: 1, Backend: "claude", Model: "sonnet-4-6", BackendType: "claude-acp",
		ModelChoice: state.ModelChoice{Kind: "explicit", ID: "native-sentinel"},
		Setup:       state.SetupChoice{Source: "bound", Provider: "claude-code", Mode: "linked"},
		AuthOrigin:  "personal", NativeHome: "personal",
	}
	if err := ix.UpsertSessionMeta("a_contract", m); err != nil {
		t.Fatal(err)
	}
	if err := ix.UpsertSessionMeta("a_contract", meta()); err != nil {
		t.Fatal(err)
	}
	snap, err = st.ReadSession("a_contract")
	if err != nil {
		t.Fatal(err)
	}
	c := snap.ExecutionContract
	if c == nil || c.ModelChoice.Kind != "explicit" || c.ModelChoice.ID != "native-sentinel" || c.Setup.Provider != "claude-code" {
		t.Fatalf("stored contract = %+v", c)
	}

	bad := meta()
	bad.ExecutionContract = &state.ExecutionContract{Version: 1, ModelChoice: state.ModelChoice{Kind: "explicit"}}
	if err := ix.UpsertSessionMeta("a_bad", bad); err == nil {
		t.Fatal("explicit choice without an id was stored")
	}
}
