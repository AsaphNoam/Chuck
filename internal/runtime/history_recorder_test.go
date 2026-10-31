package runtime

import "sync"

// recordHistory observes the same committed publication boundary used by
// transcript consumers, without requiring the runtime to retain event bodies.
func recordHistory(c *ChatRuntime) func() []Event {
	var mu sync.Mutex
	var events []Event
	c.SetHistoryPublisher(func(ev Event) {
		mu.Lock()
		events = append(events, ev)
		mu.Unlock()
	})
	return func() []Event {
		mu.Lock()
		defer mu.Unlock()
		return append([]Event(nil), events...)
	}
}
