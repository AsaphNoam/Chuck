package runtime

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// CommitOutcome is a journal append's canonical result (TS-02.R52).
type CommitOutcome uint8

const (
	// CommitNotWritten means no byte of the record reached the log.
	CommitNotWritten CommitOutcome = iota
	// CommitUncertain means the append or its required sync failed after
	// bytes may have reached the log. Its sequence stays reserved.
	CommitUncertain
	// CommitCommitted means the complete record and any required sync landed.
	CommitCommitted
)

func (o CommitOutcome) String() string {
	switch o {
	case CommitCommitted:
		return "committed"
	case CommitUncertain:
		return "uncertain"
	default:
		return "not_written"
	}
}

// Observation is the typed control view of one commit attempt (TS-04.R100).
// Control consumers check Outcome against their own authority: room capture
// needs a committed source, while task cleanup has its own durable record.
type Observation struct {
	Event   Event
	Outcome CommitOutcome
	// IndexOK is false when the canonical record committed but its derived
	// index projection failed; that never changes Outcome (TS-01.R39).
	IndexOK bool
}

// Committed reports whether the event is retained canonical history.
func (o Observation) Committed() bool { return o.Outcome == CommitCommitted }

// Journal is the single commit owner for one conversation (TS-01.R39). It
// serializes sequence allocation, canonical append and required sync, index
// projection and ordered history publication, so durable order equals
// observed order and a failed record never becomes visible history.
//
// Lock order: Journal.mu may be followed by agentState.mu inside a commit's
// post-commit hook; never take Journal.mu while holding agentState.mu. No
// provider call or control callback runs under Journal.mu.
type Journal struct {
	agentID string

	mu sync.Mutex
	w  TranscriptWriter
	ix PersistenceIndexer
	// next is the next sequence to allocate. Allocated identities are never
	// reused, including those of failed or uncertain appends.
	next int64
	// broken latches an uncertain append: the log may hold a torn or
	// unsynced record, so later appends are refused rather than fused onto it.
	broken error
	// indexDegraded records a committed event whose projection failed.
	indexDegraded bool
	publish       func(Event)
	// attachment fences Detach to the generation that attached the writer.
	attachment uint64
	// persistent marks a journal whose conversation is durable: once a
	// writer has been attached, a commit without one is not written rather
	// than published as non-durable history.
	persistent bool
	refs       int // guarded by Journals.mu
}

// NewJournal returns an unattached journal for agentID. Until a writer is
// attached it allocates sequences and publishes without durability, which is
// the runtime's behavior when persistence is not configured.
func NewJournal(agentID string, publish func(Event)) *Journal {
	return &Journal{agentID: agentID, next: 1, publish: publish}
}

// ErrJournalLive refuses an inactive append while a live generation owns the
// conversation's writer; the caller uses the live runtime path instead.
var ErrJournalLive = errors.New("runtime: conversation has a live transcript writer")

// Open opens the conversation's writer under the journal lock, so the
// writer's torn-tail repair can never cut a concurrent append, and attaches it
// for the returned attachment. A writer still attached by an older generation
// is closed and replaced; that generation's later Detach is a no-op.
func (j *Journal) Open(open func() (TranscriptWriter, error), ix PersistenceIndexer) (uint64, TranscriptWriter, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	w, err := open()
	if err != nil {
		return 0, nil, err
	}
	if j.w != nil {
		if err := j.w.Close(); err != nil {
			slog.Error("runtime: close replaced transcript writer", "agent", j.agentID, "err", err)
		}
	}
	if j.broken != nil {
		// Open truncated any torn tail; FS-03.R99 gap recovery is not yet durable.
		slog.Warn("runtime: reopening transcript after a failed append", "agent", j.agentID, "err", j.broken)
		j.broken = nil
	}
	j.attachLocked(w, ix)
	j.persistent = true
	j.attachment++
	return j.attachment, w, nil
}

// Attach makes w the canonical writer without fencing (tests and callers
// that own the journal outright).
func (j *Journal) Attach(w TranscriptWriter, ix PersistenceIndexer) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.attachLocked(w, ix)
}

// attachLocked continues allocation after both the writer's recovered tail
// and every identity this journal already reserved.
func (j *Journal) attachLocked(w TranscriptWriter, ix PersistenceIndexer) {
	j.w, j.ix = w, ix
	if n := w.NextSeq(); n > j.next {
		j.next = n
	}
}

// Detach removes and returns the writer attached by attachment, or nil when a
// newer generation has replaced it. The caller closes or discards it.
func (j *Journal) Detach(attachment uint64) TranscriptWriter {
	j.mu.Lock()
	defer j.mu.Unlock()
	if attachment != j.attachment {
		return nil
	}
	w := j.w
	j.w, j.ix = nil, nil
	return w
}

// CommitDetached appends one event to a conversation with no live writer
// (an inactive annotation, notice or switch marker): it opens the writer,
// commits and closes it under the journal lock, so it cannot interleave with
// a resume's attach or another inactive append (TS-01.R39). The record is
// synced before publication; an outcome other than committed is an error.
func (j *Journal) CommitDetached(open func() (TranscriptWriter, error), ix PersistenceIndexer, req CommitRequest) (Observation, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.w != nil {
		return Observation{}, ErrJournalLive
	}
	w, err := open()
	if err != nil {
		return Observation{}, err
	}
	j.attachLocked(w, ix)
	req.requireSync = true // the record is synced before it is published
	obs := j.commitLocked(req)
	j.w, j.ix = nil, nil
	if err := w.Close(); err != nil {
		slog.Error("runtime: close inactive transcript writer", "agent", j.agentID, "err", err)
	}
	if !obs.Committed() {
		return obs, fmt.Errorf("transcript append %s", obs.Outcome)
	}
	return obs, nil
}

// Durable reports whether a writer is attached.
func (j *Journal) Durable() bool {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.w != nil && j.ix != nil
}

// Health returns the latched append failure, if any, and whether a committed
// event's index projection failed.
func (j *Journal) Health() (broken error, indexDegraded bool) {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.broken, j.indexDegraded
}

// CommitRequest is one event offered to the journal.
type CommitRequest struct {
	// Event carries the producer's captured generation, turn and scope. The
	// journal always allocates Seq; session_meta (seq 0) is written by Open.
	Event Event
	// Rollup is the turn_end context rollup; LastSeq/UpdatedAt are filled in.
	Rollup *TurnRollup
	// Committed runs under the journal lock after a successful commit and
	// before publication: in-memory history bookkeeping only. It may take
	// agentState.mu but must not block on subscribers or call controllers.
	Committed func(Event)
	// Unpublished commits history that is not live activity (a fork prefix):
	// it is retained but never sent to the history publisher.
	Unpublished bool
	// IndexedByCaller skips projection: the caller indexes the event in its
	// own receipt transaction (a quota continuation notice).
	IndexedByCaller bool
	// Observe receives every attempt's observation, whatever its outcome,
	// under the journal lock in reserved-sequence order. It only enqueues
	// (the control dispatcher); it must not block or call controllers.
	Observe func(Observation)

	requireSync bool
}

// Commit runs one serialized commit and returns its observation. Publication
// happens only for committed events, in allocation order.
func (j *Journal) Commit(req CommitRequest) Observation {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.commitLocked(req)
}

func (j *Journal) commitLocked(req CommitRequest) Observation {
	ev := req.Event
	ev.AgentID, ev.Seq = j.agentID, j.next
	j.next++
	if ev.Ts == "" {
		ev.Ts = time.Now().UTC().Format(time.RFC3339)
	}
	obs := Observation{Event: ev, Outcome: CommitCommitted, IndexOK: true}
	switch {
	case j.w != nil && j.ix != nil:
		obs.Outcome, obs.IndexOK = j.writeLocked(ev, req)
		if !obs.IndexOK {
			j.indexDegraded = true
		}
	case j.persistent:
		// The generation's writer is closed: nothing durable can accept it.
		obs.Outcome = CommitNotWritten
	}
	if obs.Committed() {
		if req.Committed != nil {
			req.Committed(ev)
		}
		if j.publish != nil && !req.Unpublished {
			j.publish(ev)
		}
	}
	if req.Observe != nil {
		req.Observe(obs)
	}
	return obs
}

// requiresSync lists the event types whose append is not accepted until
// synced. Streaming deltas are never fsynced individually (TS-01.R39).
func requiresSync(typ string) bool {
	switch typ {
	case EvTurnEnd, EvError, EvAnnotation, EvNotice, EvUserPrompt, EvPermissionResolved:
		return true
	}
	return false
}

func (j *Journal) writeLocked(ev Event, req CommitRequest) (CommitOutcome, bool) {
	if j.broken != nil {
		return CommitNotWritten, true
	}
	if err := j.w.Append(ev); err != nil {
		j.broken = fmt.Errorf("append seq %d: %w", ev.Seq, err)
		slog.Error("runtime: append transcript", "agent", j.agentID, "seq", ev.Seq, "err", err)
		return CommitUncertain, true
	}
	if requiresSync(ev.Type) || req.requireSync {
		if err := j.w.Sync(); err != nil {
			j.broken = fmt.Errorf("sync seq %d: %w", ev.Seq, err)
			slog.Error("runtime: sync transcript event", "agent", j.agentID, "seq", ev.Seq, "type", ev.Type, "err", err)
			return CommitUncertain, true
		}
	}
	var err error
	switch {
	case req.IndexedByCaller:
	case ev.Type == EvTurnEnd:
		r := TurnRollup{}
		if req.Rollup != nil {
			r = *req.Rollup
		}
		r.LastSeq, r.UpdatedAt = ev.Seq, ev.Ts
		var td TurnEndData
		if json.Unmarshal(ev.Data, &td) == nil {
			r.LastContextPct, r.LastContext = td.ContextPct, td.ContextCounts
		}
		err = j.ix.OnEventAndTurnEnd(j.agentID, ev, r)
	case ev.Type == EvAnnotation:
		err = j.ix.OnEventAndFlushContent(j.agentID, ev, ev.Seq, ev.Ts)
	default:
		err = j.ix.OnEvent(j.agentID, ev)
	}
	if err != nil {
		slog.Error("runtime: index event", "agent", j.agentID, "seq", ev.Seq, "type", ev.Type, "err", err)
		return CommitCommitted, false
	}
	return CommitCommitted, true
}

// Journals owns one ref-counted Journal per stable agent ID, shared by every
// chat generation, the terminal runtime and server-side inactive appends
// (TS-01.R39). A journal is reclaimed only once no reference remains and no
// writer is attached. Lock order: Journals.mu → Journal.mu.
type Journals struct {
	mu      sync.Mutex
	m       map[string]*Journal
	publish func(Event)
}

// NewJournals returns an empty set whose journals publish committed history
// through publish.
func NewJournals(publish func(Event)) *Journals {
	return &Journals{m: map[string]*Journal{}, publish: publish}
}

// Acquire returns agentID's journal, creating it if needed, and takes a reference.
func (s *Journals) Acquire(agentID string) *Journal {
	s.mu.Lock()
	defer s.mu.Unlock()
	j := s.m[agentID]
	if j == nil {
		j = NewJournal(agentID, s.publish)
		s.m[agentID] = j
	}
	j.refs++
	return j
}

// Release drops one reference taken by Acquire.
func (s *Journals) Release(j *Journal) {
	if s == nil || j == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.m[j.agentID] != j {
		return
	}
	if j.refs > 0 {
		j.refs--
	}
	if j.refs == 0 && !j.Durable() {
		delete(s.m, j.agentID)
	}
}

// CommitDetached commits one event to agentID's conversation when no live
// generation holds its writer, taking a reference for the operation.
func (s *Journals) CommitDetached(agentID string, open func() (TranscriptWriter, error), ix PersistenceIndexer, req CommitRequest) (Observation, error) {
	j := s.Acquire(agentID)
	defer s.Release(j)
	return j.CommitDetached(open, ix, req)
}

// reclaim drops an unreferenced journal whose writer has since detached; a
// generation that released its reference before detaching calls it after.
func (s *Journals) reclaim(j *Journal) {
	if s == nil || j == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.m[j.agentID] == j && j.refs == 0 && !j.Durable() {
		delete(s.m, j.agentID)
	}
}

// Record opens and closes agentID's writer under its journal lock, for a
// runtime whose only record is the session_meta Open writes (terminal). It
// refuses while another generation's writer is attached.
func (s *Journals) Record(agentID string, open func() (TranscriptWriter, error)) error {
	j := s.Acquire(agentID)
	defer s.Release(j)
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.w != nil {
		return ErrJournalLive
	}
	w, err := open()
	if err != nil {
		return err
	}
	if n := w.NextSeq(); n > j.next {
		j.next = n
	}
	return w.Close()
}
