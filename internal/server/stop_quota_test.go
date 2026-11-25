package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
	"github.com/AsaphNoam/Chuck/internal/transcript"
)

type stopQuotaSlowCloseWriter struct {
	runtime.TranscriptWriter
	closing chan struct{}
	release chan struct{}
	closed  chan struct{}
}

func (w stopQuotaSlowCloseWriter) Close() error {
	close(w.closing)
	<-w.release
	defer close(w.closed)
	return w.TranscriptWriter.Close()
}

// ARCH-POST025-1 (FS-01.R6/R43, TS-10.R43, INV §15): once Stop or Archive ends
// the process, a client that disconnects during the journal-release wait cannot
// leave the captured quota continuation due to restart the conversation.
func TestStopDisconnectDuringJournalCloseCancelsQuota(t *testing.T) {
	for _, action := range []string{"stop", "archive"} {
		t.Run(action, func(t *testing.T) { testStopDisconnectDuringJournalClose(t, action) })
	}
}

func testStopDisconnectDuringJournalClose(t *testing.T, action string) {
	s := testServer(t, true)
	q := seedQuotaHTTP(t, s, "quota-stop-disconnect")
	a, err := s.stateStore.ReadAgent(q.AgentID)
	if err != nil {
		t.Fatal(err)
	}
	s.registry.Chat().SetCommand(buildServerFakeACP(t))
	closing, release, closed := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var once sync.Once
	releaseWriter := func() { once.Do(func() { close(release) }) }
	defer releaseWriter()
	s.registry.SetPersistence(s.configStore.Home(), func(home, id string, meta *runtime.SessionMetaData) (runtime.TranscriptWriter, error) {
		w, err := transcript.Open(home, id, meta)
		if err != nil {
			return nil, err
		}
		return stopQuotaSlowCloseWriter{w, closing, release, closed}, nil
	}, s.indexer)
	if _, err := s.registry.Launch(context.Background(), runtime.LaunchSpec{
		Agent: a, Generation: q.SourceGeneration, Cwd: t.TempDir(), BackendType: "claude-acp", ModelID: "claude-sonnet-4-6",
	}); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r := newLocalRequest(http.MethodPost, "/api/sessions/"+q.AgentID+"/"+action, nil).WithContext(ctx)
	w := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		s.routes().ServeHTTP(w, r)
		close(done)
	}()
	select {
	case <-closing:
		cancel()
	case <-time.After(10 * time.Second):
		t.Fatal("Stop did not reach journal close")
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Stop did not return after disconnect")
	}
	releaseWriter()
	select {
	case <-closed:
	case <-time.After(5 * time.Second):
		t.Fatal("slow close did not complete after release")
	}

	if _, err := s.stateStore.ReadRunning(q.AgentID); !errors.Is(err, state.ErrNotFound) {
		t.Fatalf("want removed process row after Stop, got %v", err)
	}
	latest, err := s.stateStore.GetQuota(q.AgentID)
	if err != nil {
		t.Fatal(err)
	}
	due, err := s.stateStore.DueQuotaIntents(time.Now().Add(time.Second), state.MaxQuotaDueBatch)
	if err != nil {
		t.Fatal(err)
	}
	if latest.State != state.QuotaCancelled || len(due) != 0 {
		t.Fatalf("%s killed the process but quota remains %s and %d intent(s) are due; HTTP %d %s", action, latest.State, len(due), w.Code, w.Body.String())
	}
	if action == "archive" {
		if a, err := s.stateStore.ReadAgent(q.AgentID); err != nil || !a.Archived {
			t.Fatalf("archive after disconnect: archived=%v err=%v; HTTP %d %s", a.Archived, err, w.Code, w.Body.String())
		}
	}
}
