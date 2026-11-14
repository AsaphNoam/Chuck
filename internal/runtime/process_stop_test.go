package runtime

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestStopEndsTermIgnoringDescendant(t *testing.T) {
	c, spec := newChatTest(t, "text")
	fake := c.command
	dir := t.TempDir()
	pidfile := filepath.Join(dir, "child.pid")
	wrapper := filepath.Join(dir, "wrap.sh")
	script := fmt.Sprintf("#!/bin/sh\n(trap '' TERM; exec sleep 300) &\necho $! > %q\nexec %q\n", pidfile, fake)
	if err := os.WriteFile(wrapper, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	c.command = wrapper
	if _, err := c.Start(context.Background(), spec); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(pidfile)
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil || pid <= 0 {
		t.Fatalf("descendant pid = %q: %v", raw, err)
	}
	t.Cleanup(func() { _ = syscall.Kill(pid, syscall.SIGKILL) })
	if err := c.Stop(context.Background(), spec.Agent.AgentID); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for PidAlive(pid) && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if PidAlive(pid) {
		t.Fatal("TERM-ignoring descendant survived Stop")
	}
}
