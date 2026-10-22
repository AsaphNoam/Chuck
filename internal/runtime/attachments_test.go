// FS-03.A62, FS-03.A64, FS-03.A65: ordered input, held ownership and capability limits.
package runtime

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/AsaphNoam/Chuck/internal/attachment"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func imageBytes(t *testing.T) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestAttachmentPromptMaximumNativeWireAndFileOrder(t *testing.T) {
	c, spec := newChatTest(t, "stream_text")
	home := t.TempDir()
	c.transcriptHome = home
	dump := filepath.Join(t.TempDir(), "prompt.json")
	spec.Env = append(spec.Env, "FAKEACP_IMAGE_INPUT=1", "FAKEACP_PROMPT_DUMP="+dump)
	h, err := c.Start(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Stop(context.Background(), h.AgentID) })
	ch, unsub, _ := c.Subscribe(h.AgentID)
	defer unsub()
	data := imageBytes(t)
	data = append(data, make([]byte, 10*1024*1024-len(data))...)
	img, err := attachment.New(home).Put(h.AgentID, "picture.png", bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	file, err := attachment.New(home).Put(h.AgentID, "notes.txt", strings.NewReader("uploaded notes"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := attachment.New(home).Put(h.AgentID, "second.png", bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	thirdData := imageBytes(t)
	thirdData = append(thirdData, make([]byte, 5*1024*1024-int(file.Size)-len(thirdData))...)
	third, err := attachment.New(home).Put(h.AgentID, "third.png", bytes.NewReader(thirdData))
	if err != nil {
		t.Fatal(err)
	}
	m := Message{Text: "exact text", Attachments: []attachment.Descriptor{img, file, second, third}, RequestID: "req-native"}
	if held, err := c.SendMessageOrHold(context.Background(), h.AgentID, m); err != nil || held {
		t.Fatalf("send=%v %v", held, err)
	}
	events := drainTurnWithin(t, ch, 20*time.Second) // 15 MiB of attachments
	raw, err := os.ReadFile(dump)
	if err != nil {
		t.Fatal(err)
	}
	var wire struct {
		Prompt []map[string]any `json:"prompt"`
	}
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatal(err)
	}
	if len(wire.Prompt) != 6 || wire.Prompt[0]["text"] != "exact text" || wire.Prompt[1]["type"] != "image" || wire.Prompt[2]["type"] != "resource_link" || wire.Prompt[4]["type"] != "image" || wire.Prompt[5]["type"] != "image" {
		t.Fatalf("wire order or exact text changed")
	}
	decoded, err := base64.StdEncoding.DecodeString(wire.Prompt[1]["data"].(string))
	if err != nil || !bytes.Equal(decoded, data) {
		t.Fatal("native image bytes changed")
	}
	if !strings.HasPrefix(wire.Prompt[2]["uri"].(string), "file://") {
		t.Fatal("file reference is not a local URI")
	}
	for _, ev := range events {
		if bytes.Contains(ev.Data, []byte(wire.Prompt[1]["data"].(string))) {
			t.Fatal("inline image entered transcript")
		}
		if ev.Type == EvUserPrompt {
			var d UserPromptData
			json.Unmarshal(ev.Data, &d)
			if d.RequestID != m.RequestID || len(d.Attachments) != 4 || d.Attachments[0] != img || d.Attachments[1] != file || d.Attachments[2] != second || d.Attachments[3] != third {
				t.Fatal("accepted message identity changed")
			}
		}
	}
}

func TestUnsupportedNativeImageRefusesBeforeHold(t *testing.T) {
	c, h, _, _, _ := busyAgent(t)
	home := t.TempDir()
	c.transcriptHome = home
	d, err := attachment.New(home).Put(h.AgentID, "picture.png", bytes.NewReader(imageBytes(t)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.SendMessageOrHold(context.Background(), h.AgentID, Message{Attachments: []attachment.Descriptor{d}}); !errors.Is(err, ErrAttachmentCapability) {
		t.Fatalf("unsupported=%v", err)
	}
	m, _, _ := c.HeldMessage(h.AgentID)
	if !m.empty() {
		t.Fatal("refused image was held")
	}
}

func TestAttachmentOnlyHeldPromotionPreservesIdentityAndFallbackOnce(t *testing.T) {
	for _, outcome := range []string{"injected", "promptRequired"} {
		t.Run(outcome, func(t *testing.T) {
			steerLog := filepath.Join(t.TempDir(), "steer.log")
			c, h, _, release, promptLog := busyAgent(t, "FAKEACP_STEERING=1", "FAKEACP_IMAGE_INPUT=1", "FAKEACP_STEER_LOG="+steerLog, "FAKEACP_STEER_OUTCOME="+outcome)
			home := t.TempDir()
			c.transcriptHome = home
			d, err := attachment.New(home).Put(h.AgentID, "image.png", bytes.NewReader(imageBytes(t)))
			if err != nil {
				t.Fatal(err)
			}
			m := Message{Attachments: []attachment.Descriptor{d}, RequestID: "held-image"}
			if held, err := c.SendMessageOrHold(context.Background(), h.AgentID, m); err != nil || !held {
				t.Fatalf("hold=%v %v", held, err)
			}
			got, _, _ := c.HeldMessage(h.AgentID)
			if got.RequestID != m.RequestID || len(got.Attachments) != 1 || got.Attachments[0] != d {
				t.Fatal("held identity changed")
			}
			got.Attachments[0].Name = "client-mutated"
			again, _, _ := c.HeldMessage(h.AgentID)
			if again.Attachments[0].Name != d.Name {
				t.Fatal("held slice aliases caller")
			}
			if _, err := c.SteerMessage(context.Background(), h.AgentID, Message{}); err != nil {
				t.Fatal(err)
			}
			raw, err := os.ReadFile(steerLog)
			if err != nil {
				t.Fatal(err)
			}
			var wire struct {
				Prompt []map[string]any `json:"prompt"`
			}
			json.Unmarshal(bytes.TrimSpace(raw), &wire)
			if len(wire.Prompt) != 1 || wire.Prompt[0]["type"] != "image" {
				t.Fatal("attachment-only steer changed")
			}
			if outcome == "promptRequired" {
				waitForPrompts(t, promptLog, 2)
			} else {
				waitForPrompts(t, promptLog, 1)
			}
			release()
			waitForRuntimeStatus(t, c, h.AgentID, "idle")
			events, _ := c.Transcript(h.AgentID)
			count := 0
			for _, ev := range events {
				if ev.Type == EvUserPrompt {
					var d UserPromptData
					json.Unmarshal(ev.Data, &d)
					if d.RequestID == m.RequestID {
						count++
					}
				}
			}
			if count != 1 {
				t.Fatalf("accepted prompt persisted %d times", count)
			}
		})
	}
}

func TestHeldReceiptFailureKeepsPreviousOwner(t *testing.T) {
	c, h, _, _, _ := busyAgent(t)
	var mu sync.Mutex
	receipts := map[string]string{}
	c.SetMessageOutcomeHandler(func(_ string, id, outcome string) error {
		mu.Lock()
		defer mu.Unlock()
		if id == "new" && outcome == "held" {
			return errors.New("durability failure")
		}
		receipts[id] = outcome
		return nil
	})
	if held, err := c.SendMessageOrHold(context.Background(), h.AgentID, Message{Text: "original", RequestID: "old"}); err != nil || !held {
		t.Fatal(err)
	}
	if _, err := c.SendMessageOrHold(context.Background(), h.AgentID, Message{Text: "replacement", RequestID: "new"}); err == nil {
		t.Fatal("failed receipt accepted")
	}
	m, _, _ := c.HeldMessage(h.AgentID)
	if m.RequestID != "old" || m.Text != "original" {
		t.Fatal("receipt failure displaced original")
	}
	mu.Lock()
	old := receipts["old"]
	mu.Unlock()
	if old != "held" {
		t.Fatalf("old receipt=%s", old)
	}
}

func TestStoppedOrCrashedInflightSteerIsNeverReleasedForRecovery(t *testing.T) {
	for _, exit := range []string{"stop", "crash"} {
		t.Run(exit, func(t *testing.T) {
			dir := t.TempDir()
			log := filepath.Join(dir, "steer.log")
			c, h, _, _, _ := busyAgent(t, "FAKEACP_STEERING=1", "FAKEACP_STEER_OUTCOME=promptRequired", "FAKEACP_STEER_LOG="+log, "FAKEACP_STEER_WAIT_FILE="+filepath.Join(dir, "never-release"))
			home := t.TempDir()
			c.transcriptHome = home
			d, err := attachment.New(home).Put(h.AgentID, "notes.txt", strings.NewReader("selected contents"))
			if err != nil {
				t.Fatal(err)
			}
			var mu sync.Mutex
			states := []string{}
			c.SetMessageOutcomeHandler(func(_ string, id, state string) error {
				if id == "in-flight" {
					mu.Lock()
					states = append(states, state)
					mu.Unlock()
				}
				return nil
			})
			result := make(chan error, 1)
			go func() {
				_, err := c.SteerMessage(context.Background(), h.AgentID, Message{Text: "steer", Attachments: []attachment.Descriptor{d}, RequestID: "in-flight"})
				result <- err
			}()
			waitForPrompts(t, log, 1)
			if exit == "stop" {
				if err := c.Stop(context.Background(), h.AgentID); err != nil {
					t.Fatal(err)
				}
			} else {
				as, err := c.lookup(h.AgentID)
				if err != nil {
					t.Fatal(err)
				}
				if err := as.cmd.Process.Kill(); err != nil {
					t.Fatal(err)
				}
			}
			select {
			case err := <-result:
				if !errors.Is(err, ErrMessageAccepted) {
					t.Fatalf("ambiguous steer=%v", err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("steer did not settle after exit")
			}
			mu.Lock()
			defer mu.Unlock()
			if len(states) < 2 || states[0] != "unknown" || states[len(states)-1] != "unknown" {
				t.Fatalf("in-flight outcomes=%v", states)
			}
			for _, state := range states {
				if state == "released" {
					t.Fatal("in-flight steer became a duplicate-prone recovery")
				}
			}
		})
	}
}

type messageTestWriter struct {
	mu     sync.Mutex
	file   *os.File
	synced bool
}

func (w *messageTestWriter) Append(ev Event) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	b, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	_, err = w.file.Write(append(b, '\n'))
	return err
}
func (w *messageTestWriter) Sync() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	err := w.file.Sync()
	w.synced = err == nil
	return err
}
func (w *messageTestWriter) Close() error   { return w.file.Close() }
func (w *messageTestWriter) Discard() error { w.file.Close(); return os.Remove(w.file.Name()) }
func (w *messageTestWriter) NextSeq() int64 { return 1 }

func TestReceiptFailureAfterDurableMessageNeverRequestsByteRollback(t *testing.T) {
	c, spec := newChatTest(t, "stream_text")
	home := t.TempDir()
	path := filepath.Join(home, "durable.ndjson")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	w := &messageTestWriter{file: f}
	c.SetPersistence(home, func(string, string, *SessionMetaData) (TranscriptWriter, error) { return w, nil }, &removalIndexer{})
	dump := filepath.Join(home, "provider-prompt.json")
	spec.Env = append(spec.Env, "FAKEACP_PROMPT_DUMP="+dump)
	h, err := c.Start(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Stop(context.Background(), h.AgentID) })
	d, err := attachment.New(home).Put(h.AgentID, "notes.txt", strings.NewReader("retained copy"))
	if err != nil {
		t.Fatal(err)
	}
	c.SetMessageOutcomeHandler(func(_ string, id, outcome string) error {
		w.mu.Lock()
		synced := w.synced
		w.mu.Unlock()
		if id != "accepted" || outcome != "submitted" || !synced {
			t.Error("receipt preceded synced accepted event")
		}
		return errors.New("receipt unavailable")
	})
	_, err = c.SendMessageOrHold(context.Background(), h.AgentID, Message{Attachments: []attachment.Descriptor{d}, RequestID: "accepted"})
	if !errors.Is(err, ErrMessageAccepted) {
		t.Fatalf("post-event failure=%v, missing accepted sentinel", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil || !bytes.Contains(raw, []byte(`"request_id":"accepted"`)) {
		t.Fatal("accepted message not durable")
	}
	if _, err := os.Stat(dump); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("provider consumed a message after receipt failure")
	}
	blob, _, err := attachment.New(home).Open(h.AgentID, d.ID)
	if err != nil {
		t.Fatal("accepted message lost owned bytes")
	}
	blob.Close()
}

func TestAssistantImagePersistsBeforeEventAndStripsToolBlob(t *testing.T) {
	c, spec := newChatTest(t, "stream_text")
	home := t.TempDir()
	c.transcriptHome = home
	data := imageBytes(t)
	data = append(data, make([]byte, 10*1024*1024-len(data))...)
	b := acpContentBlock{Type: "image", Data: base64.StdEncoding.EncodeToString(data), MIMEType: "image/png"}
	raw, _ := json.Marshal(b)
	outputFile := filepath.Join(t.TempDir(), "image.json")
	if err := os.WriteFile(outputFile, raw, 0600); err != nil {
		t.Fatal(err)
	}
	spec.Env = append(spec.Env, "FAKEACP_IMAGE_OUTPUT_FILE="+outputFile)
	h, err := c.Start(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Stop(context.Background(), h.AgentID) })
	ch, unsub, _ := c.Subscribe(h.AgentID)
	defer unsub()
	if err := c.SendPrompt(context.Background(), h.AgentID, "generate"); err != nil {
		t.Fatal(err)
	}
	events := drainTurnWithin(t, ch, 20*time.Second) // 10 MiB image frame
	found := false
	for _, ev := range events {
		if ev.Type != EvAssistantImage {
			continue
		}
		found = true
		var d AssistantImageData
		json.Unmarshal(ev.Data, &d)
		if d.Attachment == nil {
			t.Fatalf("image unavailable: %s", ev.Data)
		}
		f, _, err := attachment.New(home).Open(h.AgentID, d.Attachment.ID)
		if err != nil {
			t.Fatal("event precedes owned bytes")
		}
		f.Close()
	}
	if !found {
		t.Fatal("assistant image missing")
	}
	as, _ := c.lookup(h.AgentID)
	b.Data = base64.StdEncoding.EncodeToString(imageBytes(t))
	params, _ := json.Marshal(map[string]any{"update": map[string]any{"sessionUpdate": "tool_call_update", "toolCallId": "tool-1", "status": "completed", "content": []any{map[string]any{"type": "content", "content": b}}}})
	mapped := mapSessionUpdateImages(params, func(b acpContentBlock, id string) AssistantImageData { return c.persistImage(as, b, id) })
	if len(mapped) != 2 || mapped[0].Type != EvToolResult || mapped[1].Type != EvAssistantImage {
		t.Fatal("tool image ordering changed")
	}
	for _, m := range mapped {
		encoded, _ := json.Marshal(m.Data)
		if bytes.Contains(encoded, []byte(b.Data)) {
			t.Fatal("tool image blob entered transcript")
		}
	}
	bad, _ := json.Marshal(map[string]any{"update": map[string]any{"sessionUpdate": "agent_message_chunk", "content": acpContentBlock{Type: "image", Data: "not base64", MIMEType: "image/png"}}})
	scope := activityScope{ActivityID: "child", ParentActivityID: "parent"}
	c.mapScoped(as, scope, "agent_message_chunk", bad)
	text, _ := json.Marshal(map[string]any{"update": map[string]any{"sessionUpdate": "agent_message_chunk", "content": acpContentBlock{Type: "text", Text: "still readable"}}})
	c.mapScoped(as, scope, "agent_message_chunk", text)
	all, _ := c.Transcript(h.AgentID)
	last := all[len(all)-2:]
	if last[0].Type != EvAssistantImage || last[1].Type != EvAssistantText || last[0].ActivityID != "child" || last[0].ParentActivityID != "parent" {
		t.Fatal("child image failure corrupted surrounding activity or ordering")
	}
	var unavailable AssistantImageData
	json.Unmarshal(last[0].Data, &unavailable)
	if unavailable.Error == "" || unavailable.Attachment != nil {
		t.Fatal("invalid base64 was retained")
	}
	c.mapScoped(as, scope, "tool_call_update", params)
	all, _ = c.Transcript(h.AgentID)
	var childImage AssistantImageData
	json.Unmarshal(all[len(all)-1].Data, &childImage)
	if all[len(all)-1].Type != EvAssistantImage || childImage.ToolCallID != scope.toolCallID("tool-1") {
		t.Fatal("child tool image identity is not scoped")
	}
	malformed := json.RawMessage(`{"update":{"sessionUpdate":"tool_call_update","toolCallId":"tool-bad","status":"completed","content":[{"type":"content","content":{"type":"image","data":"SECRET_BASE64","mimeType":[]}}]}}`)
	for _, m := range mapSessionUpdateImages(malformed, func(b acpContentBlock, id string) AssistantImageData { return c.persistImage(as, b, id) }) {
		raw, _ := json.Marshal(m.Data)
		if bytes.Contains(raw, []byte("SECRET_BASE64")) {
			t.Fatal("malformed image metadata bypassed blob stripping")
		}
	}
}

func TestAttachmentCapabilityAdvertisementIsIndependentOfSteer(t *testing.T) {
	for _, test := range []struct {
		wire         string
		image, steer bool
	}{
		{`{"agentCapabilities":{"promptCapabilities":{"image":true}}}`, true, false},
		{`{"_meta":{"steering":{"supported":true}}}`, false, true},
		{`{"agentCapabilities":{"promptCapabilities":{"image":"yes"}}}`, false, false},
	} {
		caps := negotiateCapabilities(json.RawMessage(test.wire), clientOffer{})
		if caps.ImageInput == nil || *caps.ImageInput != test.image || decodeSteeringSupport(json.RawMessage(test.wire)) != test.steer {
			t.Fatalf("capabilities inferred outside advertisement: %s", test.wire)
		}
	}
}
