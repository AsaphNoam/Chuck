package runtime

import (
	"context"
	"sync"
)

// controlQueueLimit bounds one conversation's pending control observations;
// one more may be in flight (TS-01.R39).
const controlQueueLimit = 256

// controlQueue delivers one conversation's control observations to the
// control sink in commit order, outside the journal and agent locks
// (TS-01.R39, TS-04.R100). Capacity is reserved before a commit enters the
// journal, so enqueueing under the journal lock never blocks. A single
// drainer runs while items are pending and exits when the queue empties, so a
// conversation that stops leaves no goroutine behind. Pending entries are not
// restart authority: task, quota and room rows own recovery.
type controlQueue struct {
	slots   chan struct{}
	deliver func(Observation)

	mu      sync.Mutex
	items   []controlItem
	running bool
}

type controlItem struct {
	obs      Observation
	reserved bool
	done     chan struct{} // closed after delivery, for a waiting terminal emitter
}

func newControlQueue(deliver func(Observation)) *controlQueue {
	return &controlQueue{slots: make(chan struct{}, controlQueueLimit+1), deliver: deliver}
}

// reserve takes a capacity slot, waiting while the queue is full. It returns
// false when ctx ends first (the generation is shutting down); the caller then
// enqueues unreserved so its final observations are still delivered in order.
func (q *controlQueue) reserve(ctx context.Context) bool {
	select {
	case q.slots <- struct{}{}:
		return true
	default:
	}
	select {
	case q.slots <- struct{}{}:
		return true
	case <-ctx.Done():
		return false
	}
}

// enqueue appends one observation in commit order. It runs under the journal
// lock and never blocks or calls the sink.
func (q *controlQueue) enqueue(item controlItem) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.items = append(q.items, item)
	if !q.running {
		q.running = true
		go q.drain()
	}
}

func (q *controlQueue) drain() {
	for {
		q.mu.Lock()
		if len(q.items) == 0 {
			q.running = false
			q.mu.Unlock()
			return
		}
		item := q.items[0]
		q.items[0] = controlItem{}
		q.items = q.items[1:]
		q.mu.Unlock()
		q.deliver(item.obs)
		if item.reserved {
			<-q.slots
		}
		if item.done != nil {
			close(item.done)
		}
	}
}
