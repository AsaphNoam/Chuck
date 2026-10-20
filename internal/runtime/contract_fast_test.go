package runtime

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/AsaphNoam/Chuck/internal/state"
)

// TS-02.R53: the persisted contract records the applied fast value, not the
// request an adapter did not honor.
func TestContractRecordsAppliedFast(t *testing.T) {
	c, spec := newChatTest(t, "stream_text")
	f, err := os.Create(filepath.Join(t.TempDir(), "fast.ndjson"))
	if err != nil {
		t.Fatal(err)
	}
	var meta *SessionMetaData
	c.SetPersistence(t.TempDir(), func(_ string, _ string, m *SessionMetaData) (TranscriptWriter, error) {
		meta = m
		return &messageTestWriter{file: f}, nil
	}, &removalIndexer{})
	spec.Fast, spec.Agent.Fast = true, true
	spec.ExecutionContract = &state.ExecutionContract{Version: 1, Fast: true, ModelChoice: state.ModelChoice{Kind: state.ModelChoiceNative}}
	h, err := c.Start(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Stop(context.Background(), h.AgentID) })
	if h.Fast {
		t.Fatal("fixture adapter unexpectedly applied fast")
	}
	if meta.ExecutionContract.Fast != h.Fast {
		t.Fatalf("persisted contract fast=%v, applied fast=%v", meta.ExecutionContract.Fast, h.Fast)
	}
}
