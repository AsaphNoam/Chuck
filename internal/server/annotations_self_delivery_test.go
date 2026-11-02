package server

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/transcript"
)

// FS-13.R37 / TS-03.R78: a self-targeted batch is delivered like an ordinary
// chat message — woken, held or steered — rather than refused.

func selfAnnotation(t *testing.T, ts string, id, command string, steer bool) (int, []byte) {
	t.Helper()
	raw, _ := json.Marshal(runtime.AnnotationData{
		CommandID:   command,
		Annotations: []runtime.Annotation{{Seq: 1, Excerpt: "target phrase", Instruction: "review " + command}},
		Target:      runtime.AnnotationTarget{Kind: "self", Steer: steer},
	})
	resp, body := post(t, ts+"/api/sessions/"+id+"/annotations", json.RawMessage(raw))
	return resp.StatusCode, body
}

func annotationEventCount(t *testing.T, srv *Server, id string) int {
	t.Helper()
	events, err := transcript.ReadFile(srv.configStore.Home(), id, transcript.ReadOptions{})
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	n := 0
	for _, ev := range events {
		if ev.Type == runtime.EvAnnotation {
			n++
		}
	}
	return n
}

func TestSelfAnnotationWakesStoppedSource(t *testing.T) {
	srv, ts := wakeTestServer(t)
	id := launchThenStop(t, srv, ts)

	code, body := selfAnnotation(t, ts.URL, id, "wake-1", false)
	if code != http.StatusAccepted {
		t.Fatalf("stopped self annotation = %d: %s", code, body)
	}
	if got := fieldOf(t, body, "delivery"); got != "sent" {
		t.Fatalf("delivery = %q, want sent", got)
	}
	waitRunning(t, srv, id, true)
	if n := annotationEventCount(t, srv, id); n != 1 {
		t.Fatalf("annotation events = %d, want 1", n)
	}
}

func TestSelfAnnotationRefusesUnwakeableSourceBeforeRecording(t *testing.T) {
	srv, ts := wakeTestServer(t)
	id := launchThenStop(t, srv, ts)
	if err := srv.stateStore.SetAgentsArchived([]string{id}, true); err != nil {
		t.Fatalf("SetAgentsArchived: %v", err)
	}
	if code, body := selfAnnotation(t, ts.URL, id, "archived-1", false); code != http.StatusConflict {
		t.Fatalf("archived self annotation = %d: %s, want 409", code, body)
	}
	if code, body := selfAnnotation(t, ts.URL, id, "archived-steer", true); code != http.StatusConflict {
		t.Fatalf("stopped steer = %d: %s, want 409", code, body)
	}
	if n := annotationEventCount(t, srv, id); n != 0 {
		t.Fatalf("annotation events = %d, want none after refusal", n)
	}
	waitRunning(t, srv, id, false)
}

func TestSelfAnnotationHoldsBehindRunningTurn(t *testing.T) {
	srv, ts, id, release := busyChatServer(t, nil)

	code, body := selfAnnotation(t, ts.URL, id, "hold-1", false)
	if code != http.StatusAccepted {
		t.Fatalf("busy self annotation = %d: %s", code, body)
	}
	if got := fieldOf(t, body, "delivery"); got != "held" {
		t.Fatalf("delivery = %q, want held", got)
	}
	getResp, err := http.Get(ts.URL + "/api/sessions/" + id + "/prompt")
	if err != nil {
		t.Fatalf("GET held prompt: %v", err)
	}
	held, _ := io.ReadAll(getResp.Body)
	getResp.Body.Close()
	if text := fieldOf(t, held, "text"); !strings.HasPrefix(text, annotationBlockSentinelFromUI(t)) {
		t.Fatalf("held text = %q, want the annotation block", text)
	}

	release()
	deadline := time.Now().Add(10 * time.Second)
	for {
		events, err := transcript.ReadFile(srv.configStore.Home(), id, transcript.ReadOptions{})
		if err != nil {
			t.Fatalf("ReadFile: %v", err)
		}
		for _, ev := range events {
			var prompt runtime.UserPromptData
			if ev.Type == runtime.EvUserPrompt && json.Unmarshal(ev.Data, &prompt) == nil && strings.Contains(prompt.Text, "review hold-1") {
				return
			}
		}
		if time.Now().After(deadline) {
			t.Fatal("held annotation block was never sent after the turn ended")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestSelfAnnotationSteersRunningTurn(t *testing.T) {
	_, ts, id, _ := busyChatServer(t, map[string]string{"FAKEACP_STEERING": "1"})

	code, body := selfAnnotation(t, ts.URL, id, "steer-1", true)
	if code != http.StatusAccepted {
		t.Fatalf("steered self annotation = %d: %s", code, body)
	}
	if got := fieldOf(t, body, "delivery"); got != "steered" {
		t.Fatalf("delivery = %q, want steered", got)
	}
}
