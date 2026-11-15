package runtime

import (
	"context"
	"encoding/json"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/state"
)

// turnCountingIndexer counts the turn boundaries the journal hands the archive.
type turnCountingIndexer struct {
	removalIndexer
	turns atomic.Int64
}

func (ix *turnCountingIndexer) OnEventAndTurnEnd(string, Event, TurnRollup) error {
	ix.turns.Add(1)
	return nil
}

// collectUntilClosed drains ch until the hub closes it at process teardown.
func collectUntilClosed(t *testing.T, ch <-chan Event) []Event {
	t.Helper()
	var got []Event
	deadline := time.After(5 * time.Second)
	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				return got
			}
			got = append(got, ev)
		case <-deadline:
			t.Fatalf("event stream never closed; collected %d events", len(got))
		}
	}
}

func turnEndEvents(events []Event) []Event {
	var ends []Event
	for _, ev := range events {
		if ev.Type == EvTurnEnd {
			ends = append(ends, ev)
		}
	}
	return ends
}

// OUTCOME-1 (FS-03.R117/A92, TS-01.R42, INV §5/§20): the provider answers the
// prompt completely and exits. The result reached the transport before EOF, so
// it is the turn's ending even when the prompt goroutine has not resumed by the
// time the exit is handled. The later process error leaves that one outcome,
// its quota settlement and its inline-mail receipt in place.
func TestResultDeliveredBeforeExitSettlesTheTurnOnce(t *testing.T) {
	c, spec := newChatTest(t, "stream_text")
	spec.Generation = "g"
	spec.Env = append(spec.Env, "FAKEACP_EXIT_AFTER_RESULT=1")
	mu, calls := recordQuotaSettlement(c)
	release := make(chan struct{})
	var releaseOnce sync.Once
	releasePrompt := func() { releaseOnce.Do(func() { close(release) }) }
	defer releasePrompt()
	c.afterPromptCall = func() { <-release }

	ctx := context.Background()
	h, err := c.Start(ctx, spec)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	as, err := c.lookup(h.AgentID)
	if err != nil {
		t.Fatal(err)
	}
	ch, unsub, err := c.Subscribe(h.AgentID)
	if err != nil {
		t.Fatal(err)
	}
	defer unsub()
	if _, err := c.store.InsertMessage(state.Message{
		FromAgent: "a_sender", FromAddress: "reviewer@test", FromName: "Nova",
		ToAgent: h.AgentID, Subject: "Review", Body: "Please review.",
	}); err != nil {
		t.Fatal(err)
	}
	started, err := c.StartActivation(ctx, h.AgentID, "mail", func(turnID string) error {
		return c.store.BeginConversationTurn(h.AgentID, turnID, state.TurnOrigin{Generation: "g", Kind: state.TurnOriginMail})
	})
	if err != nil || !started {
		t.Fatalf("StartActivation = %v, %v; want started", started, err)
	}

	// The provider has answered and exited; the prompt goroutine is held with
	// the delivered result in hand. The exit must not settle the turn first.
	select {
	case <-as.transport.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("provider never exited")
	}
	var events []Event
	early := time.After(300 * time.Millisecond)
held:
	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				t.Fatal("exit tore the agent down while the delivered result was unsettled")
			}
			if ev.Type == EvTurnEnd {
				outcome, data, _ := DecodeTurnEnd(ev)
				t.Fatalf("exit settled the turn before its delivered result: outcome %v, data %+v", outcome, data)
			}
			events = append(events, ev)
		case <-early:
			break held
		}
	}
	releasePrompt()
	events = append(events, collectUntilClosed(t, ch)...)

	ends := turnEndEvents(events)
	if len(ends) != 1 {
		t.Fatalf("turn_end events = %d, want exactly one", len(ends))
	}
	outcome, data, err := DecodeTurnEnd(ends[0])
	if err != nil || outcome != TurnCompleted || data.StopReason != "end_turn" || data.ProcessEnded {
		t.Fatalf("turn end = %v %+v err=%v, want completed end_turn", outcome, data, err)
	}
	fatal := false
	for _, ev := range events {
		if ev.Type != EvError {
			continue
		}
		var ed ErrorData
		_ = json.Unmarshal(ev.Data, &ed)
		if ed.Fatal {
			fatal = true
			if ev.Seq < ends[0].Seq {
				t.Fatalf("fatal process error seq %d precedes the turn end seq %d", ev.Seq, ends[0].Seq)
			}
		}
	}
	if !fatal {
		t.Fatal("process exit published no fatal error")
	}
	row, err := c.store.ReadConversationTurn(h.AgentID, "g")
	if err != nil || row.TurnID != ends[0].TurnID || row.State != "ended" || row.Outcome != "completed" {
		t.Fatalf("turn row = %+v err=%v, want %s ended completed", row, err, ends[0].TurnID)
	}
	mu.Lock()
	if len(*calls) != 1 || (*calls)[0].turnID != ends[0].TurnID || !(*calls)[0].successful {
		t.Errorf("quota settlement = %+v, want one successful settlement for %s", *calls, ends[0].TurnID)
	}
	mu.Unlock()
	if unread, err := c.store.UnreadCount(h.AgentID); err != nil || unread != 0 {
		t.Fatalf("unread after the completed turn = %d, %v; want the inline receipt confirmed", unread, err)
	}
	if status, _ := c.store.ReadStatus(h.AgentID); status.State != "error" {
		t.Fatalf("status after exit = %q, want error", status.State)
	}
}

// OUTCOME-2 (FS-03.R8/R117, TS-01.R42/R45, INV §18/§20): a process exit always
// publishes its fatal error, but a turn boundary only for a turn it interrupts.
// An idle exit and an exit after a completed turn add no boundary to the
// retained events or the archive's turn count.
func TestProcessExitEndsOnlyAnUnfinishedTurn(t *testing.T) {
	for _, tc := range []struct {
		name       string
		scenario   string
		prompt     bool // send a prompt before the exit
		settle     bool // let that turn complete first
		wantEnds   int
		wantFinal  TurnOutcome
		wantTurnID bool
	}{
		{name: "idle", scenario: "stream_text"},
		{name: "after completion", scenario: "stream_text", prompt: true, settle: true, wantEnds: 1, wantFinal: TurnCompleted, wantTurnID: true},
		{name: "active turn", scenario: "hold_turn", prompt: true, wantEnds: 1, wantFinal: TurnProcessEnded, wantTurnID: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, spec := newChatTest(t, tc.scenario)
			spec.Generation = "g"
			spec.Env = append(spec.Env, "FAKEACP_HOLD_FILE="+filepath.Join(t.TempDir(), "never"))
			ix := &turnCountingIndexer{}
			c.SetPersistence(t.TempDir(), func(string, string, *SessionMetaData) (TranscriptWriter, error) { return &recordingWriter{}, nil }, ix)
			retained := recordHistory(c)
			ctx := context.Background()
			h, err := c.Start(ctx, spec)
			if err != nil {
				t.Fatalf("Start: %v", err)
			}
			ch, unsub, err := c.Subscribe(h.AgentID)
			if err != nil {
				t.Fatal(err)
			}
			defer unsub()
			if tc.prompt {
				if err := c.SendPrompt(ctx, h.AgentID, "go"); err != nil {
					t.Fatal(err)
				}
				if tc.settle {
					drainTurn(t, ch)
					waitTurnSettled(t, c, h.AgentID)
				} else {
					waitForEvent(t, ch, EvAssistantText)
				}
			}
			if err := syscall.Kill(-h.Pid, syscall.SIGKILL); err != nil {
				t.Fatalf("kill provider: %v", err)
			}
			collectUntilClosed(t, ch)

			events := retained()
			ends := turnEndEvents(events)
			if len(ends) != tc.wantEnds {
				t.Fatalf("retained turn_end events = %d, want %d", len(ends), tc.wantEnds)
			}
			if got := ix.turns.Load(); got != int64(tc.wantEnds) {
				t.Fatalf("archive turn boundaries = %d, want %d", got, tc.wantEnds)
			}
			fatal := false
			for _, ev := range events {
				var ed ErrorData
				if ev.Type == EvError && json.Unmarshal(ev.Data, &ed) == nil && ed.Fatal && ed.Scope == "process" {
					fatal = true
				}
			}
			if !fatal {
				t.Fatal("exit retained no fatal process error")
			}
			if tc.wantEnds == 0 {
				return
			}
			last := ends[len(ends)-1]
			if outcome, _, err := DecodeTurnEnd(last); err != nil || outcome != tc.wantFinal {
				t.Fatalf("last turn end = %v err=%v, want %v", outcome, err, tc.wantFinal)
			}
			if tc.wantTurnID && last.TurnID == "" {
				t.Fatal("turn end carries no turn identity")
			}
		})
	}
}
