package server

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/AsaphNoam/Chuck/internal/config"
	"github.com/AsaphNoam/Chuck/internal/state"
)

func TestChatStopPublishesStoppedCard(t *testing.T) {
	fake := buildFakeACP(t)
	t.Setenv("FAKEACP_SCENARIO", "stream_text")

	for _, action := range []string{"agent", "group"} {
		t.Run(action, func(t *testing.T) {
			srv := testServer(t, true)
			srv.registry.Chat().SetCommand(fake)
			t.Cleanup(func() { srv.registry.Shutdown(context.Background()) })
			if err := srv.configStore.WriteProject("tmpproj", config.Project{Title: "Tmp", Cwd: t.TempDir()}); err != nil {
				t.Fatal(err)
			}
			if err := srv.configStore.WriteRole("impl", config.Role{Title: "Impl", SystemPrompt: "be helpful"}); err != nil {
				t.Fatal(err)
			}

			handler := srv.routes()
			rec := doJSON(t, handler, http.MethodPost, "/api/sessions", `{"project":"tmpproj","role":"impl","group":"review"}`)
			var launched sessionResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &launched); err != nil || rec.Code != http.StatusCreated || launched.Running == nil {
				t.Fatalf("launch = %d %s, decode error %v", rec.Code, rec.Body, err)
			}
			id := launched.Agent.AgentID
			updates, unsubscribe := srv.eventBus.Subscribe()
			defer unsubscribe()

			if action == "agent" {
				rec = doJSON(t, handler, http.MethodPost, "/api/sessions/"+id+"/stop", `{}`)
			} else {
				rec = doJSON(t, handler, http.MethodPost, "/api/projects/tmpproj/groups/stop", `{"group":"review"}`)
				var result groupActionResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil || len(result.Results) != 1 || !result.Results[0].OK {
					t.Fatalf("group stop = %d %s, decode error %v", rec.Code, rec.Body, err)
				}
			}
			if rec.Code != http.StatusOK {
				t.Fatalf("stop = %d %s", rec.Code, rec.Body)
			}
			if _, err := srv.stateStore.ReadRunning(id); err != state.ErrNotFound {
				t.Fatalf("ReadRunning after stop = %v, want no running row", err)
			}

			var latest state.AgentStateUpdate
			for len(updates) > 0 {
				event := <-updates
				if update, ok := event.Data.(state.AgentStateUpdate); event.Type == "state_update" && ok && update.AgentID == id {
					latest = update
				}
			}
			if latest.AgentID != id || latest.Running || latest.State != "done" {
				t.Errorf("last card update after successful stop = agent %q, running=%t, state=%q; want %q, running=false, state=done",
					latest.AgentID, latest.Running, latest.State, id)
			}
		})
	}
}
