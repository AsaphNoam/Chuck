package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/AsaphNoam/Chuck/internal/attachment"
	"github.com/AsaphNoam/Chuck/internal/state"
)

type messageDiagnosticBuffer struct {
	sync.Mutex
	bytes.Buffer
}

func (b *messageDiagnosticBuffer) Write(p []byte) (int, error) {
	b.Lock()
	defer b.Unlock()
	return b.Buffer.Write(p)
}

func (b *messageDiagnosticBuffer) snapshot() string {
	b.Lock()
	defer b.Unlock()
	return b.Buffer.String()
}

// The real queue/Steer boundary must identify each durable receipt without
// copying its private text or uploaded file metadata into diagnostics.
func TestMessageDeliveryDiagnostics(t *testing.T) {
	for _, outcome := range []string{"injected", "reject", "lost_ack"} {
		t.Run(outcome, func(t *testing.T) {
			var logs messageDiagnosticBuffer
			previous := slog.Default()
			slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
			t.Cleanup(func() { slog.SetDefault(previous) })
			env := []string{"FAKEACP_STEERING=1", "FAKEACP_STEER_OUTCOME=" + outcome}
			ack := filepath.Join(t.TempDir(), "ack")
			steerLog := filepath.Join(t.TempDir(), "steer.log")
			if outcome == "lost_ack" {
				env = append(env, "FAKEACP_STEER_ACK_FILE="+ack, "FAKEACP_STEER_LOG="+steerLog)
			}
			c, h, _, _, _ := busyAgent(t, env...)
			c.transcriptHome = t.TempDir()
			c.SetMessageOutcomeHandler(c.store.SetAttachmentDelivery)
			file, err := attachment.New(c.transcriptHome).Put(h.AgentID, "private-filename.txt", strings.NewReader("private-file-contents"))
			if err != nil {
				t.Fatal(err)
			}
			for _, id := range []string{"old", "withdrawn", "steered"} {
				if err := c.store.CreateAttachmentReceipt(state.AttachmentReceipt{AgentID: h.AgentID, RequestID: id, PayloadHash: "hash", MessageJSON: `{}`, Action: "prompt"}); err != nil {
					t.Fatal(err)
				}
				if held, err := c.SendMessageOrHold(context.Background(), h.AgentID, Message{RequestID: id, Text: "private-message-body", Attachments: []attachment.Descriptor{file}}); err != nil || !held {
					t.Fatalf("queue %s: %v %v", id, held, err)
				}
				if id == "withdrawn" {
					if err := c.WithdrawHeld(h.AgentID); err != nil {
						t.Fatal(err)
					}
				}
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan error, 1)
			go func() { _, err := c.SteerMessage(ctx, h.AgentID, Message{}); done <- err }()
			if outcome == "lost_ack" {
				waitForPrompts(t, steerLog, 1)
				cancel()
			}
			steerErr := <-done
			if outcome == "lost_ack" {
				if err := os.WriteFile(ack, []byte("release provider"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if (steerErr != nil) != (outcome != "injected") {
				t.Fatalf("Steer %s: %v", outcome, steerErr)
			}
			raw := logs.snapshot()
			for _, secret := range []string{"private-message-body", "private-filename.txt", "private-file-contents", file.ID} {
				if strings.Contains(raw, secret) {
					t.Fatalf("diagnostics leaked payload %q", secret)
				}
			}
			want := map[string]string{
				"old/queue/accepted": "held", "old/queue/replaced": "released",
				"withdrawn/withdraw/withdrawn":      "released",
				"steered/steer/native_call_pending": "unknown",
			}
			final := "submitted"
			if outcome == "reject" {
				want["steered/steer/provider_refused"] = "held"
				final = "held"
			} else if outcome == "lost_ack" {
				want["steered/steer/provider_error"] = "unknown"
				final = "unknown"
			} else {
				want["steered/steer/provider_acknowledged"] = "submitted"
			}
			for _, line := range strings.Split(strings.TrimSpace(raw), "\n") {
				var entry map[string]any
				if err := json.Unmarshal([]byte(line), &entry); err != nil {
					t.Fatal(err)
				}
				if entry["msg"] != "runtime: message delivery transition" {
					continue
				}
				key := entry["request_id"].(string) + "/" + entry["operation"].(string) + "/" + entry["reason"].(string)
				if delivery, ok := want[key]; ok {
					if entry["agent"] != h.AgentID || entry["delivery"] != delivery {
						t.Fatalf("wrong receipt diagnostic: %v", entry)
					}
					delete(want, key)
				}
			}
			if len(want) != 0 {
				t.Fatalf("missing diagnostics: %v", want)
			}
			receipt, err := c.store.AttachmentReceipt(h.AgentID, "steered")
			if err != nil || receipt.Delivery != final {
				t.Fatalf("diagnostic differs from durable delivery: %+v %v", receipt, err)
			}
		})
	}
}
