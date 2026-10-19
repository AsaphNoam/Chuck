package runtime

import (
	"encoding/json"
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
}

// NewJournal returns an unattached journal for agentID. Without an attached
// writer it allocates sequences and publishes without durability, which is
// the runtime's behavior when persistence is not configured.
func NewJournal(agentID string, publish func(Event)) *Journal {
	return &Journal{agentID: agentID, next: 1, publish: publish}
}

// Attach makes w the canonical writer. Allocation continues after both the
// writer's recovered tail and every identity this journal already reserved.
func (j *Journal) Attach(w TranscriptWriter, ix PersistenceIndexer) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.w, j.ix = w, ix
	if n := w.NextSeq(); n > j.next {
		j.next = n
	}
}

// Detach removes and returns the attached writer, if any. The caller closes
// or discards it; commits after Detach are not durable.
func (j *Journal) Detach() TranscriptWriter {
	j.mu.Lock()
	defer j.mu.Unlock()
	w := j.w
	j.w, j.ix = nil, nil
	return w
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
}

// Commit runs one serialized commit and returns its observation. Publication
// happens only for committed events, in allocation order.
func (j *Journal) Commit(req CommitRequest) Observation {
	j.mu.Lock()
	defer j.mu.Unlock()
	ev := req.Event
	ev.AgentID, ev.Seq = j.agentID, j.next
	j.next++
	if ev.Ts == "" {
		ev.Ts = time.Now().UTC().Format(time.RFC3339)
	}
	obs := Observation{Event: ev, Outcome: CommitCommitted, IndexOK: true}
	if j.w != nil && j.ix != nil {
		obs.Outcome, obs.IndexOK = j.writeLocked(ev, req)
		if !obs.IndexOK {
			j.indexDegraded = true
		}
	}
	if !obs.Committed() {
		return obs
	}
	if req.Committed != nil {
		req.Committed(ev)
	}
	if j.publish != nil && !req.Unpublished {
		j.publish(ev)
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
	if requiresSync(ev.Type) {
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
