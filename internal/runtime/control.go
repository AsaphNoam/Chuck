package runtime

import (
	"context"
	"log/slog"
	"sync"
)

// controlQueueLimit bounds one conversation's pending control observations;
// one more may be in flight (TS-01.R39).
const controlQueueLimit = 256

// controlQueue delivers one conversation's control observations to the
// control sink in commit order, outside the journal and agent locks
// (TS-01.R39, TS-04.R100). It belongs to the conversation's Journal, so every
// generation of that conversation shares one FIFO and one drainer. Capacity is
// reserved before a commit enters the journal, so enqueueing under the journal
// lock never blocks. A single drainer runs while items are pending and exits
// when the queue empties, so a conversation that stops leaves no goroutine
// behind. Pending entries are not restart authority: task, quota and room rows
// own recovery.
type controlQueue struct {
	slots   chan struct{}
	deliver func(Observation)
	// idle runs after the drainer exits, outside every lock (journal reclaim).
	idle func()

	mu      sync.Mutex
	items   []controlItem
	running bool
	// shed holds generations whose shutdown found the queue full. Their later
	// observations are never delivered, so no consumer settles from a gapped
	// stream; the generation's exit teardown owns their cleanup.
	shed map[string]struct{}
}

type controlItem struct {
	obs      Observation
	reserved bool
	done     chan struct{} // closed after delivery or shedding, for a waiting terminal emitter
}

func newControlQueue(deliver func(Observation)) *controlQueue {
	return &controlQueue{slots: make(chan struct{}, controlQueueLimit+1), deliver: deliver}
}

// reserve takes a capacity slot for generation, waiting while the queue is
// full. It returns false when ctx ends first (the generation is shutting
// down) or the generation was already shed; the observation is then not
// delivered and capacity stays bounded through shutdown.
func (q *controlQueue) reserve(ctx context.Context, generation string) bool {
	if q.isShed(generation) {
		return false
	}
	select {
	case q.slots <- struct{}{}:
		return true
	default:
	}
	select {
	case q.slots <- struct{}{}:
		return true
	case <-ctx.Done():
		q.mu.Lock()
		if q.shed == nil {
			q.shed = map[string]struct{}{}
		}
		q.shed[generation] = struct{}{}
		q.mu.Unlock()
		slog.Warn("runtime: control queue full at shutdown; exit teardown settles the generation", "generation", generation)
		return false
	}
}

// forget drops generation's shed marker at its exit teardown, once it can emit
// nothing more, so markers live no longer than their generation (INV §4/§16).
func (q *controlQueue) forget(generation string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	delete(q.shed, generation)
}

func (q *controlQueue) isShed(generation string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	_, ok := q.shed[generation]
	return ok
}

// enqueue appends one observation in commit order. It runs under the journal
// lock and never blocks or calls the sink. An unreserved observation, or one
// of a generation shed since it reserved, is dropped and its waiter released.
func (q *controlQueue) enqueue(item controlItem) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if _, shed := q.shed[item.obs.Event.Generation]; shed || !item.reserved {
		if item.reserved {
			<-q.slots
		}
		if item.done != nil {
			close(item.done)
		}
		return
	}
	q.items = append(q.items, item)
	if !q.running {
		q.running = true
		go q.drain()
	}
}

// busy reports whether observations are pending or in flight.
func (q *controlQueue) busy() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.running
}

func (q *controlQueue) drain() {
	for {
		q.mu.Lock()
		if len(q.items) == 0 {
			q.running = false
			q.mu.Unlock()
			if q.idle != nil {
				q.idle()
			}
			return
		}
		item := q.items[0]
		q.items[0] = controlItem{}
		q.items = q.items[1:]
		q.mu.Unlock()
		q.deliver(item.obs)
		<-q.slots
		if item.done != nil {
			close(item.done)
		}
	}
}
