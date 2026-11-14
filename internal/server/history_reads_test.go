package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
)

// The history route and point reads must not queue behind each other's pool.
// Holding a connection models a long scan without requiring a large fixture.
func TestHistorySearchDoesNotShareWriterConnection(t *testing.T) {
	s := testServer(t, true)
	const agent = "a_history_pool"
	if err := s.indexer.UpsertSessionMeta(agent, runtime.SessionMetaData{
		Name: "Needle", Project: "my-app", Interface: "chat",
	}); err != nil {
		t.Fatal(err)
	}
	writer := s.stateStore.DB()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	conn, err := writer.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	readDone := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		rec := httptest.NewRecorder()
		s.routes().ServeHTTP(rec, newLocalRequest(http.MethodGet, "/api/archive/projects/my-app?q=Needle", nil))
		readDone <- rec
	}()
	select {
	case rec := <-readDone:
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), agent) {
			t.Fatalf("archive search: %d %s", rec.Code, rec.Body)
		}
	case <-ctx.Done():
		// Unblock the legacy route before allowing test server cleanup.
		_ = conn.Close()
		<-readDone
		t.Fatal("archive search waited for the writer connection")
	}
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}

	reader := s.stateStore.HistoryDB()
	if _, err := reader.Exec("CREATE TABLE must_not_be_created (id INTEGER)"); err == nil {
		t.Fatal("history connection accepted a write")
	}
	ctx, cancelRead := context.WithTimeout(context.Background(), time.Second)
	defer cancelRead()
	held, err := reader.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer held.Close()
	var name string
	if err := writer.QueryRowContext(ctx, "SELECT name FROM sessions WHERE agent_id = ?", agent).Scan(&name); err != nil || name != "Needle" {
		t.Fatalf("point read while history connection is occupied: name=%q err=%v", name, err)
	}
	if err := s.stateStore.WriteAgent(state.Agent{
		AgentID: agent, Name: "Needle", Project: "my-app", Interface: "chat", CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("write while history connection is occupied: %v", err)
	}
}
