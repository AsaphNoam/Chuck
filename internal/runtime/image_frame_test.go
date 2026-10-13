// FS-03.A65: bounded image output across the actual ACP wire.
package runtime

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/AsaphNoam/Chuck/internal/attachment"
)

func TestToolImageFrameBoundsAcrossActualWire(t *testing.T) {
	for _, test := range []struct {
		name                          string
		sizes                         []int
		retained, unavailable, events int
	}{
		{"eleven small images", make([]int, 11), 10, 1, 10},
		{"exact aggregate boundary", []int{10 << 20, 10 << 20, 5 << 20}, 3, 0, 3},
		{"one byte over aggregate", []int{10 << 20, 10 << 20, (5 << 20) + 1}, 2, 1, 3},
	} {
		t.Run(test.name, func(t *testing.T) {
			c, spec := newChatTest(t, "stream_text")
			home := t.TempDir()
			c.transcriptHome = home
			blocks := []any{map[string]any{"type": "content", "content": map[string]any{"type": "text", "text": "before images"}}}
			for i, size := range test.sizes {
				data := imageBytes(t)
				if size > len(data) {
					data = append(data, make([]byte, size-len(data))...)
				}
				blocks = append(blocks, map[string]any{"type": "content", "content": map[string]any{"type": "image", "mimeType": "image/png", "data": base64.StdEncoding.EncodeToString(data)}})
				if i == 0 {
					blocks = append(blocks, map[string]any{"type": "content", "content": map[string]any{"type": "text", "text": "between images"}})
				}
			}
			blocks = append(blocks, map[string]any{"type": "content", "content": map[string]any{"type": "text", "text": "after images"}})
			raw, err := json.Marshal(map[string]any{"sessionUpdate": "tool_call_update", "toolCallId": "image-batch", "status": "completed", "content": blocks})
			if err != nil {
				t.Fatal(err)
			}
			if len(raw) >= maxFrameBytes {
				t.Fatal("test frame exceeds negotiated transport bound")
			}
			fixture := filepath.Join(t.TempDir(), "update.json")
			if err := os.WriteFile(fixture, raw, 0600); err != nil {
				t.Fatal(err)
			}
			spec.Env = append(spec.Env, "FAKEACP_UPDATE_OUTPUT_FILE="+fixture)
			h, err := c.Start(context.Background(), spec)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { c.Stop(context.Background(), h.AgentID) })
			ch, unsubscribe, _ := c.Subscribe(h.AgentID)
			defer unsubscribe()
			if err := c.SendPrompt(context.Background(), h.AgentID, "produce image batch"); err != nil {
				t.Fatal(err)
			}
			events := drainTurn(t, ch)
			imageEvents := 0
			var result *ToolResultData
			for _, ev := range events {
				if ev.Type == EvAssistantImage {
					imageEvents++
					var d AssistantImageData
					if json.Unmarshal(ev.Data, &d) != nil || d.ToolCallID != "image-batch" {
						t.Fatal("image event lost tool identity")
					}
				}
				if ev.Type == EvToolResult {
					var d ToolResultData
					if err := json.Unmarshal(ev.Data, &d); err != nil {
						t.Fatal(err)
					}
					if d.ToolCallID == "image-batch" {
						result = &d
					}
				}
			}
			if imageEvents != test.events || imageEvents > attachment.MaxFiles || result == nil {
				t.Fatalf("normalized image events=%d, expected %d and one tool result", imageEvents, test.events)
			}
			var contents []json.RawMessage
			if err := json.Unmarshal(result.Content, &contents); err != nil {
				t.Fatal(err)
			}
			retained, unavailable := 0, 0
			texts := []string{}
			for _, raw := range contents {
				var block struct {
					Type    string                      `json:"type"`
					Image   AssistantImageData          `json:"image"`
					Content struct{ Type, Text string } `json:"content"`
				}
				if err := json.Unmarshal(raw, &block); err != nil {
					t.Fatal(err)
				}
				if block.Type == "image" {
					if block.Image.ToolCallID != "image-batch" {
						t.Fatal("tool image descriptor lost identity")
					}
					if block.Image.Attachment != nil {
						retained++
					} else {
						unavailable++
						if block.Image.ErrorCode != CodeAttachmentLimit {
							t.Fatalf("overflow lacks typed limit: %+v", block.Image)
						}
					}
				} else if block.Content.Type == "text" {
					texts = append(texts, block.Content.Text)
				}
			}
			if retained != test.retained || unavailable != test.unavailable {
				t.Fatalf("retained/unavailable=%d/%d, want %d/%d", retained, unavailable, test.retained, test.unavailable)
			}
			if len(texts) != 3 || texts[0] != "before images" || texts[1] != "between images" || texts[2] != "after images" {
				t.Fatalf("surrounding tool text changed: %v", texts)
			}
			entries, err := os.ReadDir(filepath.Join(home, "sessions", h.AgentID, "attachments"))
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != test.retained {
				t.Fatalf("persisted %d blobs, expected %d", len(entries), test.retained)
			}
			if bytes.Contains(result.Content, []byte(`"data":`)) {
				t.Fatal("inline image data survived normalization")
			}
		})
	}
}
