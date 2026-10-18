package runtime

import (
	"time"

	"golang.org/x/sys/unix"
)

// ProcessStartTime reports when the live process pid started, so a caller can
// tell a recorded process from a later one that reused its PID (INV §9).
func ProcessStartTime(pid int) (time.Time, error) {
	kp, err := unix.SysctlKinfoProc("kern.proc.pid", pid)
	if err != nil {
		return time.Time{}, err
	}
	tv := kp.Proc.P_starttime
	if tv.Sec == 0 {
		return time.Time{}, errProcessStartUnknown
	}
	return time.Unix(int64(tv.Sec), int64(tv.Usec)*1000), nil
}
