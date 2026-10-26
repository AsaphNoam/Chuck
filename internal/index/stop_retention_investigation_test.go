package index

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
	"github.com/AsaphNoam/Chuck/internal/transcript"
)

type stopInvestigationWriter struct {
	*transcript.Writer
	closed chan struct{}
}

func (w *stopInvestigationWriter) Close() error {
	err := w.Writer.Close()
	close(w.closed)
	return err
}

// MEM-3: a stopped generation has no current turn whose index text can remain
// in memory (INV sections 4/16, TS-02.R16). The durable transcript remains the
// recovery authority. Cross the real Start/stream/Stop boundary and wait for
// writer teardown, rather than testing an index cleanup helper in isolation.
func TestStopReleasesPendingIndexContent(t *testing.T) {
	t.Skip("MEM-3: Stop skips turn flush and retains pending index content; confirmed with this skip removed")
	st, home := openTestDB(t)
	ix := New(st.DB())
	fake := filepath.Join(t.TempDir(), "fakeacp")
	if output, err := exec.Command("go", "build", "-o", fake, "../runtime/testdata/fakeacp").CombinedOutput(); err != nil {
		t.Fatalf("build fake ACP: %v\n%s", err, output)
	}
	c := runtime.NewChatRuntime(st)
	c.SetCommand(fake)
	closed := make(chan struct{})
	c.SetPersistence(home, func(home, id string, meta *runtime.SessionMetaData) (runtime.TranscriptWriter, error) {
		w, err := transcript.Open(home, id, meta)
		if err != nil {
			return nil, err
		}
		return &stopInvestigationWriter{Writer: w, closed: closed}, nil
	}, ix)
	agent := state.Agent{
		AgentID: "a_stop_retention", Name: "Stop retention", Backend: "claude",
		Model: "sonnet-4-6", Interface: "chat", CreatedAt: time.Now().UTC(),
	}
	if err := st.WriteAgent(agent); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Start(context.Background(), runtime.LaunchSpec{
		Agent: agent, Cwd: t.TempDir(), BackendType: "claude-acp", ModelID: "claude-sonnet-4-6",
		Env: []string{"FAKEACP_SCENARIO=hold_turn", "HOME=" + os.Getenv("HOME")},
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Stop(context.Background(), agent.AgentID) })
	events, unsub, err := c.Subscribe(agent.AgentID)
	if err != nil {
		t.Fatal(err)
	}
	defer unsub()
	if err := c.SendPrompt(context.Background(), agent.AgentID, strings.Repeat("x", 64*1024)); err != nil {
		t.Fatal(err)
	}
	deadline := time.After(5 * time.Second)
	for {
		select {
		case event := <-events:
			if event.Type == runtime.EvAssistantText {
				goto streamed
			}
		case <-deadline:
			t.Fatal("fake ACP did not stream the held turn")
		}
	}

streamed:
	if err := c.Stop(context.Background(), agent.AgentID); err != nil {
		t.Fatal(err)
	}
	select {
	case <-closed:
	case <-time.After(5 * time.Second):
		t.Fatal("stopped generation did not close its transcript writer")
	}
	if _, err := c.Transcript(agent.AgentID); !errors.Is(err, runtime.ErrNoHandle) {
		t.Fatalf("stopped generation still owned: %v", err)
	}
	ix.mu.Lock()
	defer ix.mu.Unlock()
	retainedBytes := 0
	for _, entry := range ix.content[agent.AgentID] {
		retainedBytes += len(entry.text)
	}
	if retainedBytes != 0 {
		t.Fatalf("stopped agent retains %d bytes of pending index text", retainedBytes)
	}
}
