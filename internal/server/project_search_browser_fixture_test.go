package server

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/attachment"
	"github.com/AsaphNoam/Chuck/internal/config"
	"github.com/AsaphNoam/Chuck/internal/index"
	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
	"github.com/AsaphNoam/Chuck/internal/transcript"
)

// Opt-in real-store browser fixture, never the operator's own home/providers.
func TestProjectSearchBrowserFixture(t *testing.T) {
	ready := os.Getenv("CHUCK_SEARCH_BROWSER_READY")
	if ready == "" {
		t.Skip("manual project-search browser fixture")
	}
	s := testServer(t, true)
	defer s.registry.Shutdown(context.Background())
	s.registry.Chat().SetCommand(buildFakeACP(t))
	cfg, err := s.configStore.ReadConfig()
	if err != nil {
		t.Fatal(err)
	}
	cfg.OnboardingComplete = true
	if err = s.configStore.WriteConfig(cfg); err != nil {
		t.Fatal(err)
	}
	project := "search-project"
	if err = s.configStore.WriteProject(project, config.Project{Title: "Search project with a long operational heading", Cwd: t.TempDir()}); err != nil {
		t.Fatal(err)
	}
	ix := index.New(s.stateStore.DB())
	type sourceCase struct {
		Name     string `json:"name"`
		Query    string `json:"query"`
		AgentID  string `json:"agent_id"`
		Expected string `json:"expected"`
		Key      string `json:"key"`
		Route    string `json:"route"`
	}
	cases := []sourceCase{}
	seed := func(id, name string, archived bool, events []runtime.Event) {
		t.Helper()
		now := time.Now().UTC()
		meta := runtime.SessionMetaData{Name: name, Role: "implementer", Project: project, Backend: "claude", Model: "sonnet", Interface: "chat", Cwd: "/tmp", CreatedAt: now.Format(time.RFC3339)}
		if err := s.stateStore.WriteAgent(state.Agent{AgentID: id, Name: name, Role: meta.Role, Project: project, Backend: meta.Backend, Model: meta.Model, Interface: "chat", CreatedAt: now, Archived: archived}); err != nil {
			t.Fatal(err)
		}
		if err := ix.UpsertSessionMeta(id, meta); err != nil {
			t.Fatal(err)
		}
		w, err := transcript.Open(s.configStore.Home(), id, &meta)
		if err != nil {
			t.Fatal(err)
		}
		defer w.Close()
		for i, event := range events {
			event.Seq = int64(i + 1)
			event.Ts = now.Format(time.RFC3339)
			if err = w.Append(event); err != nil {
				t.Fatal(err)
			}
			if event.Type == runtime.EvTurnEnd {
				err = ix.OnEventAndTurnEnd(id, event, runtime.TurnRollup{LastSeq: event.Seq, UpdatedAt: event.Ts})
			} else if event.Type == runtime.EvAnnotation {
				err = ix.OnEventAndFlushContent(id, event, event.Seq, event.Ts)
			} else {
				err = ix.OnEvent(id, event)
			}
			if err != nil {
				t.Fatal(err)
			}
		}
		if err = ix.FlushPendingContent(id, int64(len(events))); err != nil {
			t.Fatal(err)
		}
		if _, err = s.stateMgr.Touch(id); err != nil {
			t.Fatal(err)
		}
	}
	ev := func(kind string, data any) runtime.Event {
		raw, err := json.Marshal(data)
		if err != nil {
			t.Fatal(err)
		}
		return runtime.Event{Type: kind, Data: raw}
	}
	end := func() runtime.Event { return ev(runtime.EvTurnEnd, runtime.TurnEndData{StopReason: "end_turn"}) }
	// The hit is in a later delta of an old completed turn; newer turns force
	// genuine scroll distance and the normal collapsed-turn state.
	merged := []runtime.Event{ev(runtime.EvUserPrompt, runtime.UserPromptData{Text: "Discuss retention"}), ev(runtime.EvAssistantText, runtime.AssistantTextData{Delta: "Earlier same café.\n\n"}), ev(runtime.EvAssistantText, runtime.AssistantTextData{Delta: "Later café authenticationneedle is the retained finding."}), end()}
	merged = append(merged[:3], ev(runtime.EvToolCall, runtime.ToolCallData{ToolCallID: "search-read", Name: "Read", Args: json.RawMessage(`{"path":"example.go"}`)}), ev(runtime.EvToolResult, runtime.ToolResultData{ToolCallID: "search-read", Status: "completed"}), ev(runtime.EvAssistantText, runtime.AssistantTextData{Delta: "Final summary after reading."}), end())
	for i := 0; i < 8; i++ {
		merged = append(merged, ev(runtime.EvUserPrompt, runtime.UserPromptData{Text: "Later unrelated prompt"}), ev(runtime.EvAssistantText, runtime.AssistantTextData{Delta: strings.Repeat("Later unrelated discussion. ", 100)}), end())
	}
	seed("a_search_merged", "Retained merged discussion", false, merged)
	cases = append(cases, sourceCase{"Retained merged discussion", "authenticationn", "a_search_merged", "authenticationneedle", "merged", "agent"})
	seed("a_search_archived", "Archived prompt discussion", true, []runtime.Event{ev(runtime.EvUserPrompt, runtime.UserPromptData{Text: "Find archivedpromptneedle in this original prompt"}), ev(runtime.EvAssistantText, runtime.AssistantTextData{Delta: "Response"}), end()})
	cases = append(cases, sourceCase{"Archived prompt discussion", "archivedpromptn", "a_search_archived", "archivedpromptneedle", "archived-prompt", "archive"})
	seed("a_search_annotation", "Annotation discussion", false, []runtime.Event{ev(runtime.EvAnnotation, runtime.AnnotationData{Annotations: []runtime.Annotation{{Seq: 1, Excerpt: "Original selected text", Instruction: "Review annotationneedle carefully"}}, Target: runtime.AnnotationTarget{Kind: "self"}})})
	cases = append(cases, sourceCase{"Annotation discussion", "annotationn", "a_search_annotation", "annotationneedle", "annotation", "agent"})
	seed("a_search_code", "Code discussion", true, []runtime.Event{ev(runtime.EvAssistantText, runtime.AssistantTextData{Delta: "```go\nvar codeneedle = 1\n```"}), end()})
	cases = append(cases, sourceCase{"Code discussion", "coden", "a_search_code", "codeneedle", "code", "archive"})
	seed("a_search_attachment", "Attachment discussion", false, []runtime.Event{ev(runtime.EvUserPrompt, runtime.UserPromptData{Text: "Review this file", Attachments: []attachment.Descriptor{{Name: "attachmentneedle.txt", Kind: "file", MIMEType: "text/plain"}}}), end()})
	cases = append(cases, sourceCase{"Attachment discussion", "attachmentn", "a_search_attachment", "attachmentneedle", "attachment", "agent"})
	seed("a_search_cross", "Cross-event discussion", true, []runtime.Event{ev(runtime.EvAssistantText, runtime.AssistantTextData{Delta: "crosssource "}), ev(runtime.EvAssistantText, runtime.AssistantTextData{Delta: "needles"}), end()})
	cases = append(cases, sourceCase{"Cross-event discussion", `"crosssource needles"`, "a_search_cross", "crosssource needles", "cross-event", "archive"})
	desktop := httptest.NewServer(s.routes())
	defer desktop.Close()
	live := launchAndWaitIdle(t, desktop, "implementer", project)
	data, _ := json.Marshal(map[string]any{"desktop": desktop.URL, "home": s.configStore.Home(), "project": project, "cases": cases, "live_agent": live})
	if err = os.WriteFile(ready, data, 0600); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 900; i++ {
		if _, err = os.Stat(ready + ".done"); err == nil {
			return
		}
		time.Sleep(time.Second)
	}
	t.Fatal("browser fixture timed out")
}
