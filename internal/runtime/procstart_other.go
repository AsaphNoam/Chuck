//go:build !darwin && !linux

package runtime

import "time"

// ProcessStartTime is unavailable here, so callers treat identity as
// unverifiable and refuse destructive action (INV §9).
func ProcessStartTime(int) (time.Time, error) {
	return time.Time{}, errProcessStartUnknown
}
