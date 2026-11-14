package runtime

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

func TestHeldSuccessorCannotLeaveExitedProcessBusy(t *testing.T) {
	c, spec := newChatTest(t, "hold_turn")
	dir := t.TempDir()
	hold := dir + "/hold"
	spec.Generation = "generation"
	spec.Env = append(spec.Env, "FAKEACP_HOLD_FILE="+hold)
	history := recordHistory(c)
	c.SetPersistence(t.TempDir(), func(string, string, *SessionMetaData) (TranscriptWriter, error) { return &recordingWriter{}, nil }, &removalIndexer{})

	var killed atomic.Bool
	sinkEntered := make(chan struct{})
	sinkRelease := make(chan struct{})
	var once sync.Once
	var releaseOnce sync.Once
	releaseSink := func() { releaseOnce.Do(func() { close(sinkRelease) }) }
	defer releaseSink()
	bindReturned := make(chan struct{})
	c.SetEventSink(func(obs Observation) {
		if killed.Load() && obs.Event.Type == EvTurnEnd {
			once.Do(func() { close(sinkEntered) })
			<-sinkRelease
		}
	})
	ctx := context.Background()
	h, err := c.Start(ctx, spec)
	if err != nil {
		t.Fatal(err)
	}
	ch, unsub, _ := c.Subscribe(h.AgentID)
	defer unsub()
	first := ""
	c.SetQuotaManualBind(func(agentID, gen, turnID string) error {
		if first == "" {
			first = turnID
			return nil
		}
		killed.Store(true)
		_ = syscall.Kill(-h.Pid, syscall.SIGKILL)
		select {
		case <-sinkEntered: // onTransportClosed is emitting its turn_end
		case <-time.After(5 * time.Second):
			t.Error("exit teardown never reached turn_end")
		}
		close(bindReturned)
		return nil
	})
	if err := c.SendPrompt(ctx, h.AgentID, "first"); err != nil {
		t.Fatal(err)
	}
	waitForChunk(t, ch, "working")
	if held, err := c.SendPromptOrHold(ctx, h.AgentID, "second"); err != nil || !held {
		t.Fatalf("hold: %v %v", held, err)
	}
	if err := os.WriteFile(hold, []byte("go"), 0o600); err != nil {
		t.Fatal(err)
	}
	select {
	case <-bindReturned:
	case <-time.After(5 * time.Second):
		t.Fatal("held successor never raced the process exit")
	}
	releaseSink()
	for {
		ev := waitForEvent(t, ch, EvTurnEnd)
		if outcome, _, _ := DecodeTurnEnd(ev); outcome == TurnProcessEnded {
			row, err := c.store.ReadConversationTurn(h.AgentID, spec.Generation)
			if err != nil || row.TurnID != ev.TurnID || row.State != "ended" || row.Outcome != "process_ended" {
				t.Fatalf("exit event %s, turn row %+v, err %v", ev.TurnID, row, err)
			}
			break
		}
	}
	if st, err := c.store.ReadStatus(h.AgentID); err != nil || st.State != "error" {
		t.Fatalf("exited process status = %+v, err %v", st, err)
	}
	for _, ev := range history() {
		if ev.Type == EvUserPrompt && ev.TurnID != first {
			t.Errorf("retained an unaccepted prompt after process exit: %s", ev.Data)
		}
	}
}
