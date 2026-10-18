package runtime

import (
	"errors"
	"time"
)

var errProcessStartUnknown = errors.New("runtime: process start time unavailable")

// processStartSlack absorbs stored-timestamp rounding and small clock steps.
const processStartSlack = 2 * time.Second

// ProcessIsRecorded reports whether live process pid is the one a record
// written at recordedAt names. That process was already running when the
// record was written, while any process that later reused its PID started
// after it; an unknown start time is never treated as a match (INV §9).
func ProcessIsRecorded(pid int, recordedAt time.Time) bool {
	started, err := ProcessStartTime(pid)
	return err == nil && !recordedAt.IsZero() && !started.After(recordedAt.Add(processStartSlack))
}
