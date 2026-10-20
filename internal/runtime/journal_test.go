package runtime

import (
	"encoding/json"
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

type memHealth struct {
	mu       sync.Mutex
	h        map[string]HistoryHealth
	failOpen bool
}

func (m *memHealth) ReadHistoryHealth(id string) (HistoryHealth, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.h[id], nil
}
func (m *memHealth) WriteHistoryHealth(id string, h HistoryHealth) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failOpen && h.OpenWriter && h.State == HistoryOK {
		return errors.New("db locked")
	}
	if m.h == nil {
		m.h = map[string]HistoryHealth{}
	}
	m.h[id] = h
	return nil
}

func noticeTitles(t *testing.T, w *recordingWriter, events []Event) []string {
	t.Helper()
	var out []string
	for _, ev := range events {
		var n NoticeData
		if ev.Type == EvNotice && json.Unmarshal(ev.Data, &n) == nil {
			out = append(out, n.Title)
		}
	}
	return out
}

// eventWriter records whole events so a test can inspect gap notices.
type eventWriter struct {
	recordingWriter
	events []Event
}

func (w *eventWriter) Append(ev Event) error {
	if err := w.recordingWriter.Append(ev); err != nil {
		return err
	}
	w.events = append(w.events, ev)
	return nil
}

// FS-03.R99 / TS-02.R52: an uncertain append degrades durable health outside
// the log and refuses new work; once storage works again, Admit reopens the log,
// records one gap notice at a fresh sequence and reports healthy.
func TestJournalDegradesThenRecoversWithGapNotice(t *testing.T) {
	health := &memHealth{}
	var notified []HistoryHealth
	s := NewJournals(nil)
	s.SetHealth(health, func(_ string, h HistoryHealth) { notified = append(notified, h) })
	j := s.Acquire("a_h")
	first := &eventWriter{recordingWriter: recordingWriter{failSeq: 2}}
	reopenFails := true
	second := &eventWriter{}
	opens := 0
	open := func() (TranscriptWriter, error) {
		opens++
		if opens == 1 {
			return first, nil
		}
		if reopenFails {
			return nil, errors.New("disk still full")
		}
		return second, nil
	}
	if _, _, err := j.Open(open, &removalIndexer{}); err != nil {
		t.Fatal(err)
	}
	if h := health.h["a_h"]; h.State != HistoryOK || !h.OpenWriter {
		t.Fatalf("open marker = %+v", h)
	}
	j.Commit(CommitRequest{Event: Event{Type: EvAssistantText}})
	if obs := j.Commit(CommitRequest{Event: Event{Type: EvAssistantText}}); obs.Outcome != CommitUncertain {
		t.Fatalf("outcome = %v", obs.Outcome)
	}
	if h := health.h["a_h"]; h.State != HistoryDegraded || h.Reason != "append_failed" || h.ReservedSeq != 2 || !h.OpenWriter {
		t.Fatalf("degraded health = %+v", h)
	}
	if len(notified) != 1 || notified[0].State != HistoryDegraded {
		t.Fatalf("notifications = %+v", notified)
	}
	if err := j.Admit(); !errors.Is(err, ErrHistoryUnavailable) {
		t.Fatalf("Admit while storage still fails = %v", err)
	}
	if h := health.h["a_h"]; h.State != HistoryDegraded {
		t.Fatalf("health after failed repair = %+v", h)
	}
	reopenFails = false
	if err := j.Admit(); err != nil {
		t.Fatalf("Admit after storage recovered: %v", err)
	}
	if titles := noticeTitles(t, &second.recordingWriter, second.events); len(titles) != 1 || titles[0] != "Some conversation history could not be saved" {
		t.Fatalf("gap notices = %v", titles)
	}
	if second.events[0].Seq <= 2 {
		t.Fatalf("gap notice reused reserved sequence %d", second.events[0].Seq)
	}
	if h := health.h["a_h"]; h.State != HistoryOK || !h.OpenWriter {
		t.Fatalf("health after recovery = %+v", h)
	}
	if obs := j.Commit(CommitRequest{Event: Event{Type: EvAssistantText}}); !obs.Committed() {
		t.Fatalf("commit after recovery = %+v", obs)
	}
}

// TS-02.R52: a writer that never closed cleanly (crash) leaves its marker;
// the next open records a gap notice. Legacy unknown health gets none, a clean
// close clears the marker, and failure to record the marker refuses the writer.
func TestJournalOpenMarkerRecovery(t *testing.T) {
	health := &memHealth{h: map[string]HistoryHealth{"a_crash": {State: HistoryOK, OpenWriter: true}}}
	s := NewJournals(nil)
	s.SetHealth(health, nil)

	crashed := &eventWriter{}
	j := s.Acquire("a_crash")
	att, _, err := j.Open(func() (TranscriptWriter, error) { return crashed, nil }, &removalIndexer{})
	if err != nil {
		t.Fatal(err)
	}
	if titles := noticeTitles(t, &crashed.recordingWriter, crashed.events); len(titles) != 1 || titles[0] != "Conversation history may be incomplete" {
		t.Fatalf("unclean-stop gap notices = %v", titles)
	}
	if err := j.Close(att, false); err != nil {
		t.Fatal(err)
	}
	if h := health.h["a_crash"]; h.State != HistoryOK || h.OpenWriter {
		t.Fatalf("health after clean close = %+v", h)
	}

	legacy := &eventWriter{}
	if _, _, err := s.Acquire("a_legacy").Open(func() (TranscriptWriter, error) { return legacy, nil }, &removalIndexer{}); err != nil {
		t.Fatal(err)
	}
	if len(legacy.events) != 0 {
		t.Fatalf("legacy unknown health wrote %d records; want validation only", len(legacy.events))
	}

	health.failOpen = true
	if _, _, err := s.Acquire("a_refused").Open(func() (TranscriptWriter, error) { return &eventWriter{}, nil }, &removalIndexer{}); err == nil {
		t.Fatal("writer admitted without a recorded open marker")
	}
}

// TS-02.R52 (ASW1-5): a failed append's reserved identity survives a
// degraded clean close and a restart whose log tail no longer shows it; the
// recovery gap notice is allocated after it, never at the reused identity.
func TestJournalReservedSequenceSurvivesCloseAndRestart(t *testing.T) {
	health := &memHealth{}
	s := NewJournals(nil)
	s.SetHealth(health, nil)
	j := s.Acquire("a_res")
	failing := &eventWriter{recordingWriter: recordingWriter{failSeq: 2}}
	att, _, err := j.Open(func() (TranscriptWriter, error) { return failing, nil }, &removalIndexer{})
	if err != nil {
		t.Fatal(err)
	}
	j.Commit(CommitRequest{Event: Event{Type: EvAssistantText}})
	if obs := j.Commit(CommitRequest{Event: Event{Type: EvAssistantText}}); obs.Outcome != CommitUncertain {
		t.Fatalf("outcome = %v", obs.Outcome)
	}
	if err := j.Close(att, false); err != nil {
		t.Fatal(err)
	}
	if h := health.h["a_res"]; h.State != HistoryDegraded || h.ReservedSeq != 2 || h.OpenWriter {
		t.Fatalf("degraded close health = %+v; want reservation 2 kept", h)
	}

	// A restart: a fresh journal set whose writer's tail ends before seq 2.
	health.h["a_res"] = HistoryHealth{State: HistoryDegraded, Reason: "append_failed", ReservedSeq: 42, OpenWriter: true}
	restarted := NewJournals(nil)
	restarted.SetHealth(health, nil)
	w := &eventWriter{}
	if _, _, err := restarted.Acquire("a_res").Open(func() (TranscriptWriter, error) { return w, nil }, &removalIndexer{}); err != nil {
		t.Fatal(err)
	}
	if len(w.events) != 1 || w.events[0].Seq <= 42 {
		t.Fatalf("recovery records = %+v; want one gap notice after reserved 42", w.events)
	}
}

// FS-03.A78: a newer generation's open supersedes a writer the older one has
// not yet closed; the older generation's late commit continues the same
// sequence space and its late Close leaves the new writer attached.
func TestJournalGenerationReplacement(t *testing.T) {
	s := NewJournals(nil)
	j := s.Acquire("a_gen")
	oldW, newW := &eventWriter{}, &eventWriter{}
	oldAtt, _, err := j.Open(func() (TranscriptWriter, error) { return oldW, nil }, &removalIndexer{})
	if err != nil {
		t.Fatal(err)
	}
	j.Commit(CommitRequest{Event: Event{Type: EvAssistantText, Generation: "g1"}})
	if _, _, err := j.Open(func() (TranscriptWriter, error) { return newW, nil }, &removalIndexer{}); err != nil {
		t.Fatal(err)
	}
	late := j.Commit(CommitRequest{Event: Event{Type: EvTurnEnd, Generation: "g1"}})
	if err := j.Close(oldAtt, false); err != nil {
		t.Fatal(err)
	}
	next := j.Commit(CommitRequest{Event: Event{Type: EvUserPrompt, Generation: "g2"}})
	if !late.Committed() || !next.Committed() || late.Event.Seq != 2 || next.Event.Seq != 3 {
		t.Fatalf("late=%+v next=%+v; want seqs 2 and 3 committed", late, next)
	}
	if len(oldW.events) != 1 || len(newW.events) != 2 {
		t.Fatalf("old writer kept %d, new writer %d; want 1 and 2", len(oldW.events), len(newW.events))
	}
}

type blockedMetaIndexer struct {
	removalIndexer
	entered chan struct{}
	release chan struct{}
}

func (ix *blockedMetaIndexer) UpsertSessionMeta(string, SessionMetaData) error {
	close(ix.entered)
	<-ix.release
	return nil
}

// TS-01.R39 / INV §4 (ASW1-3): an exit teardown that runs after the journal
// attached a writer but before startup finished must not strand that writer:
// later inactive appends still reach the conversation.
func TestExitDuringPersistenceOpenClosesJournal(t *testing.T) {
	c := NewChatRuntime(nil)
	ix := &blockedMetaIndexer{entered: make(chan struct{}), release: make(chan struct{})}
	c.SetPersistence(t.TempDir(), func(string, string, *SessionMetaData) (TranscriptWriter, error) { return &recordingWriter{}, nil }, ix)
	as := &agentState{agentID: "a_attach", journals: c.journals}
	as.journal = c.journals.Acquire(as.agentID)
	opened := make(chan error, 1)
	go func() { opened <- c.openPersistence(as, LaunchSpec{}, "s") }()
	<-ix.entered
	as.closePersistence() // the transport-closed teardown
	close(ix.release)
	<-opened
	if _, err := c.CommitInactive(as.agentID, CommitRequest{Event: Event{Type: EvAnnotation}}); errors.Is(err, ErrJournalLive) {
		t.Fatal("exited startup left a live writer with no owner; inactive history is wedged")
	}
}
