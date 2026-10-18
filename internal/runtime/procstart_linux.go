package runtime

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// linuxClockTicks is USER_HZ, fixed at 100 on every supported Linux ABI.
const linuxClockTicks = 100

// ProcessStartTime reports when the live process pid started, so a caller can
// tell a recorded process from a later one that reused its PID (INV §9).
func ProcessStartTime(pid int) (time.Time, error) {
	stat, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return time.Time{}, err
	}
	// Fields after the parenthesized command name; starttime is field 22.
	end := bytes.LastIndexByte(stat, ')')
	if end < 0 {
		return time.Time{}, errProcessStartUnknown
	}
	fields := strings.Fields(string(stat[end+1:]))
	if len(fields) < 20 {
		return time.Time{}, errProcessStartUnknown
	}
	ticks, err := strconv.ParseInt(fields[19], 10, 64)
	if err != nil {
		return time.Time{}, errProcessStartUnknown
	}
	boot, err := linuxBootTime()
	if err != nil {
		return time.Time{}, err
	}
	return boot.Add(time.Duration(ticks) * time.Second / linuxClockTicks), nil
}

func linuxBootTime() (time.Time, error) {
	raw, err := os.ReadFile("/proc/stat")
	if err != nil {
		return time.Time{}, err
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if v, ok := strings.CutPrefix(line, "btime "); ok {
			sec, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
			if err != nil {
				break
			}
			return time.Unix(sec, 0), nil
		}
	}
	return time.Time{}, errProcessStartUnknown
}
