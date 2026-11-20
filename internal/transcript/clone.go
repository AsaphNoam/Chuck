package transcript

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/AsaphNoam/Chuck/internal/runtime"
)

// CloneCompletedPrefix is the sole clone builder (TS-02.R35): the source's
// active events through a root turn_end, without the source's session
// metadata, annotation or revision records, which belong to the source alone. The clone's
// own session metadata comes from its normal launch path. throughSeq names
// that turn_end (TS-03.R86); zero means the latest one. The returned boundary
// is the turn_end itself, zero-Seq when there is no such completed turn.
func CloneCompletedPrefix(home, sourceAgentID string, throughSeq int64) ([]runtime.Event, runtime.Event, error) {
	events, err := ReadFile(home, sourceAgentID, ReadOptions{})
	if err != nil {
		return nil, runtime.Event{}, err
	}
	var boundary runtime.Event
	for _, ev := range events {
		if ev.Type == runtime.EvTurnEnd && ev.ActivityID == "" && (throughSeq == 0 || ev.Seq == throughSeq) {
			boundary = ev
		}
	}
	if boundary.Seq == 0 {
		return nil, runtime.Event{}, nil
	}
	prefix := make([]runtime.Event, 0, len(events))
	for _, ev := range events {
		if ev.Seq <= 0 || ev.Seq > boundary.Seq {
			continue
		}
		// The read is already folded; a source revision record would name the
		// source's sequence range and hide the clone's renumbered history
		// (TS-02.R70).
		if ev.Type == runtime.EvSessionMeta || ev.Type == runtime.EvAnnotation || ev.Type == runtime.EvConversationRevision {
			continue
		}
		prefix = append(prefix, ev)
	}
	return prefix, boundary, nil
}

// Discard closes the log and removes it. It is the rollback for a clone whose
// launch fails after its copied prefix was written (TS-02.R35, INV §15).
func (w *Writer) Discard() error {
	errClose := w.Close()
	if err := os.Remove(w.path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("transcript: discard: %w", err)
	}
	_ = os.Remove(filepath.Dir(w.path)) // only when empty
	return errClose
}
