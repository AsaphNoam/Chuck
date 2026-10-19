package runtime

import (
	"errors"
	"sync"
	"testing"
	"time"
)

// recordingWriter records the durable append order. A short sleep inside
// Append widens the window in which an unserialized commit would reorder.
type recordingWriter struct {
	mu       sync.Mutex
	seqs     []int64
	failSeq  int64
	syncFail bool
}

func (w *recordingWriter) Append(ev Event) error {
	time.Sleep(50 * time.Microsecond)
	w.mu.Lock()
	defer w.mu.Unlock()
	if ev.Seq == w.failSeq {
		return errors.New("disk full")
	}
	w.seqs = append(w.seqs, ev.Seq)
	return nil
}
func (w *recordingWriter) Sync() error {
	if w.syncFail {
		return errors.New("sync failed")
	}
	return nil
}
func (w *recordingWriter) Close() error   { return nil }
func (w *recordingWriter) NextSeq() int64 { return 1 }
func (w *recordingWriter) Discard() error { return nil }

func newChatJournalState(t *testing.T, w TranscriptWriter) (*ChatRuntime, *agentState, *[]int64) {
	t.Helper()
	c := NewChatRuntime(nil)
	var pubMu sync.Mutex
	published := &[]int64{}
	c.SetHistoryPublisher(func(ev Event) {
		pubMu.Lock()
		*published = append(*published, ev.Seq)
		pubMu.Unlock()
	})
	as := &agentState{agentID: "a_journal", hub: NewHub(), generation: "g1"}
	as.journal = NewJournal(as.agentID, c.publishHistory)
	as.journal.Attach(w, &removalIndexer{})
	return c, as, published
}

// ASRD-2 / FS-03.A78: concurrent provider output, annotations and notices
// commit with unique increasing sequences whose durable order equals both the
// hub and history-publisher order and the in-memory history.
func TestJournalSerializesConcurrentCommits(t *testing.T) {
	w := &recordingWriter{}
	c, as, published := newChatJournalState(t, w)
	ch, cancel := as.hub.Subscribe()
	defer cancel()

	const per = 40
	var wg sync.WaitGroup
	start := make(chan struct{})
	for g := 0; g < 3; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			<-start
			for i := 0; i < per; i++ {
				switch g {
				case 0:
					c.emit(as, EvAssistantText, AssistantTextData{Delta: "x"})
				case 1:
					if _, err := c.commitNotice(as, CommitRequest{Event: Event{Type: EvNotice}}, NoticeData{Severity: "info", Title: "n"}, "notice"); err != nil {
						t.Errorf("notice: %v", err)
					}
				case 2:
					c.emit(as, EvAnnotation, AnnotationData{})
				}
			}
		}(g)
	}
	close(start)
	wg.Wait()

	var hubSeqs []int64
	for len(hubSeqs) < 3*per {
		select {
		case ev := <-ch:
			hubSeqs = append(hubSeqs, ev.Seq)
		case <-time.After(time.Second):
			t.Fatalf("hub delivered %d of %d events", len(hubSeqs), 3*per)
		}
	}
	as.mu.Lock()
	var memSeqs []int64
	for _, ev := range as.transcript {
		memSeqs = append(memSeqs, ev.Seq)
	}
	as.mu.Unlock()
	for name, got := range map[string][]int64{"durable": w.seqs, "hub": hubSeqs, "history": *published, "memory": memSeqs} {
		if len(got) != 3*per {
			t.Fatalf("%s order has %d events, want %d", name, len(got), 3*per)
		}
		for i, seq := range got {
			if seq != int64(i+1) {
				t.Fatalf("%s order[%d] = %d, want %d (%v)", name, i, seq, i+1, got)
			}
		}
	}
}

// ASRD-3 / FS-03.R99: an append or required-sync failure never becomes live
// history; the control observer still sees the attempt and its outcome, the
// sequence is not reused, and the latch refuses later appends.
func TestJournalFailedAppendIsNotPublished(t *testing.T) {
	for _, tc := range []struct {
		name string
		w    *recordingWriter
		typ  string
		data any
	}{
		{"append", &recordingWriter{failSeq: 2}, EvAssistantText, AssistantTextData{Delta: "lost"}},
		{"sync", &recordingWriter{}, EvTurnEnd, TurnEndData{StopReason: "end_turn"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, as, published := newChatJournalState(t, tc.w)
			observedCh := make(chan Observation, 8)
			c.SetEventSink(func(obs Observation) { observedCh <- obs })
			ch, cancel := as.hub.Subscribe()
			defer cancel()

			if _, ok := c.emitChecked(as, activityScope{}, EvAssistantText, AssistantTextData{Delta: "kept"}, false); !ok {
				t.Fatal("first event did not commit")
			}
			tc.w.syncFail = tc.name == "sync"
			if _, ok := c.emitChecked(as, activityScope{}, tc.typ, tc.data, false); ok {
				t.Fatal("failed event reported committed")
			}
			tc.w.syncFail = false
			if _, ok := c.emitChecked(as, activityScope{}, EvAssistantText, AssistantTextData{Delta: "after"}, false); ok {
				t.Fatal("append after an uncertain failure was accepted without repair")
			}

			if got := <-ch; got.Seq != 1 {
				t.Fatalf("hub first event = %d", got.Seq)
			}
			select {
			case ev := <-ch:
				t.Fatalf("failed event reached the hub: %+v", ev)
			case <-time.After(20 * time.Millisecond):
			}
			if len(*published) != 1 || (*published)[0] != 1 {
				t.Fatalf("history publisher saw %v, want only seq 1", *published)
			}
			if txt, _ := c.transcriptOf(as); len(txt) != 1 {
				t.Fatalf("in-memory history kept %d events, want 1", len(txt))
			}
			var observed []Observation
			for len(observed) < 3 {
				select {
				case obs := <-observedCh:
					observed = append(observed, obs)
				case <-time.After(time.Second):
					t.Fatalf("control observations = %+v, want 3 in commit order", observed)
				}
			}
			if !observed[0].Committed() || observed[1].Outcome != CommitUncertain || observed[2].Outcome != CommitNotWritten {
				t.Fatalf("observations = %+v", observed)
			}
			if observed[1].Event.Seq != 2 || observed[2].Event.Seq != 3 {
				t.Fatalf("failed attempts reused sequences: %d, %d", observed[1].Event.Seq, observed[2].Event.Seq)
			}
			if broken, _ := as.journal.Health(); broken == nil {
				t.Fatal("journal health did not latch the failure")
			}
		})
	}
}

// Index projection failure does not change whether the log accepted the event.
func TestJournalIndexFailureStillCommits(t *testing.T) {
	j := NewJournal("a_ix", nil)
	j.Attach(&recordingWriter{}, &failingIndexer{})
	obs := j.Commit(CommitRequest{Event: Event{Type: EvAnnotation}})
	if !obs.Committed() || obs.IndexOK {
		t.Fatalf("observation = %+v, want committed with index degraded", obs)
	}
	if _, degraded := j.Health(); !degraded {
		t.Fatal("index degradation not recorded")
	}
}

type failingIndexer struct{ removalIndexer }

func (*failingIndexer) OnEventAndFlushContent(string, Event, int64, string) error {
	return errors.New("index down")
}

func (c *ChatRuntime) transcriptOf(as *agentState) ([]Event, error) {
	as.mu.Lock()
	defer as.mu.Unlock()
	return append([]Event(nil), as.transcript...), nil
}

// TS-01.R39 / FS-03.A78: concurrent emitters' control observations reach the
// sink in commit order, one at a time, outside the journal lock; a turn_end
// emitter returns only after its own observation settled.
func TestControlObservationsSettleInCommitOrder(t *testing.T) {
	c, as, _ := newChatJournalState(t, &recordingWriter{})
	var mu sync.Mutex
	var seqs []int64
	inFlight := 0
	c.SetEventSink(func(obs Observation) {
		mu.Lock()
		inFlight++
		if inFlight > 1 {
			t.Error("control observations delivered concurrently")
		}
		mu.Unlock()
		// A history-only notice from inside a callback must not deadlock.
		if _, err := c.commitNotice(as, CommitRequest{Event: Event{Type: EvNotice}}, NoticeData{Severity: "info", Title: "reentrant"}, "notice"); err != nil {
			t.Errorf("reentrant notice: %v", err)
		}
		mu.Lock()
		seqs = append(seqs, obs.Event.Seq)
		inFlight--
		mu.Unlock()
	})
	var wg sync.WaitGroup
	for g := 0; g < 4; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 25; i++ {
				c.emit(as, EvAssistantText, AssistantTextData{Delta: "x"})
			}
		}()
	}
	wg.Wait()
	end := c.emit(as, EvTurnEnd, TurnEndData{StopReason: "end_turn"})
	mu.Lock()
	defer mu.Unlock()
	if len(seqs) != 101 || seqs[len(seqs)-1] != end.Seq {
		t.Fatalf("turn_end returned before its observation settled: %d observed, last %v, turn_end %d", len(seqs), seqs[len(seqs)-1:], end.Seq)
	}
	for i := 1; i < len(seqs); i++ {
		if seqs[i] <= seqs[i-1] {
			t.Fatalf("control order %d after %d: observations must follow commit order", seqs[i], seqs[i-1])
		}
	}
}
