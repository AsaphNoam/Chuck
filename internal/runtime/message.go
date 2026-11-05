package runtime

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"github.com/AsaphNoam/Chuck/internal/attachment"
	"github.com/AsaphNoam/Chuck/internal/state"
	"io"
	"log/slog"
	"net/url"
)

// Message carries the exact user input; host instructions are assembled separately.
type Message struct {
	recorded    bool
	Text        string                  `json:"text"`
	Attachments []attachment.Descriptor `json:"attachments,omitempty"`
	RequestID   string                  `json:"request_id,omitempty"`
	HeldID      string                  `json:"held_id,omitempty"`
}

func (m Message) empty() bool { return m.Text == "" && len(m.Attachments) == 0 }
func (m Message) copy() Message {
	m.Attachments = append([]attachment.Descriptor{}, m.Attachments...)
	return m
}
func (m Message) promptData() UserPromptData {
	return UserPromptData{Text: m.Text, Attachments: m.Attachments, RequestID: m.RequestID}
}

func hold(m Message, seq int64) heldMessage {
	id := "hold_" + rand.Text()
	return heldMessage{HeldID: id, Text: m.Text, Attachments: m.Attachments, RequestID: m.RequestID, AfterSeq: seq, recorded: m.recorded}
}
func (h heldMessage) message() Message {
	return Message{Text: h.Text, Attachments: h.Attachments, RequestID: h.RequestID, HeldID: h.HeldID, recorded: h.recorded}
}
func (c *ChatRuntime) messageTransition(agentID, requestID, outcome string) error {
	return c.messageTransitionReason(agentID, requestID, outcome, "delivery", "runtime_transition")
}

// Keep diagnostics to receipt identity and fixed vocabulary. Provider errors,
// message bodies and attachment descriptors can contain private input.
func (c *ChatRuntime) messageTransitionReason(agentID, requestID, outcome, operation, reason string) error {
	if requestID == "" {
		return nil
	}
	c.messageMu.RLock()
	fn := c.messageOutcome
	c.messageMu.RUnlock()
	if fn != nil {
		if err := fn(agentID, requestID, outcome); err != nil {
			return err
		}
	}
	slog.Info("runtime: message delivery transition", "agent", boundedMessageIdentity(agentID),
		"request_id", boundedMessageIdentity(requestID), "operation", operation, "delivery", outcome, "reason", reason)
	return nil
}

func boundedMessageIdentity(id string) string {
	const limit = 128
	if len(id) > limit {
		return id[:limit]
	}
	return id
}
func (c *ChatRuntime) SetMessageOutcomeHandler(fn func(string, string, string) error) {
	c.messageMu.Lock()
	c.messageOutcome = fn
	c.messageMu.Unlock()
}

// queuedInputPreamble introduces a person's queued input inside the
// assignment continuation that carries it (TS-09.R69, FS-14.R100).
const queuedInputPreamble = "The person sent this message for your assignment while it was waiting. Treat it as an instruction or correction; it does not approve a permission request or a pipeline checkpoint:"

// takeQueuedAssignmentInput claims the oldest input a person queued for the
// assignment this activation continues. Input that cannot be composed (its
// bytes vanished or the runtime cannot read images) is refused for explicit
// recovery rather than silently dropped or retried forever.
func (c *ChatRuntime) takeQueuedAssignmentInput(as *agentState) (Message, []map[string]any, bool) {
	if c.store == nil {
		return Message{}, nil, false
	}
	rec, ok, err := c.store.TakeQueuedAssignmentInput(as.agentID)
	if err != nil || !ok {
		return Message{}, nil, false
	}
	var m Message
	if err := json.Unmarshal([]byte(rec.MessageJSON), &m); err != nil {
		_ = c.messageTransition(as.agentID, rec.RequestID, "failed")
		return Message{}, nil, false
	}
	m.RequestID = rec.RequestID
	if err := c.cloneQueuedAttachments(rec, m); err != nil {
		_ = c.messageTransition(as.agentID, rec.RequestID, "failed")
		return Message{}, nil, false
	}
	content, err := c.messageContent(as, m)
	if err != nil || len(content) == 0 {
		_ = c.messageTransition(as.agentID, rec.RequestID, "failed")
		return Message{}, nil, false
	}
	return m, content, true
}

// cloneQueuedAttachments gives an assignment's new owner its own copy of the
// attachments a person sent the previous owner (TS-09.R69).
func (c *ChatRuntime) cloneQueuedAttachments(rec state.AttachmentReceipt, m Message) error {
	if rec.SourceAgentID == "" || len(m.Attachments) == 0 {
		return nil
	}
	c.mu.Lock()
	home := c.transcriptHome
	c.mu.Unlock()
	return CloneQueuedAttachments(home, rec, m)
}

// CloneQueuedAttachments copies a moved receipt's attachments from the
// conversation that accepted them into the one that will carry them.
func CloneQueuedAttachments(home string, rec state.AttachmentReceipt, m Message) error {
	if rec.SourceAgentID == "" || len(m.Attachments) == 0 {
		return nil
	}
	ids := make([]string, 0, len(m.Attachments))
	for _, d := range m.Attachments {
		ids = append(ids, d.ID)
	}
	return attachment.New(home).Clone(rec.SourceAgentID, rec.AgentID, ids)
}

func (c *ChatRuntime) messageContent(as *agentState, m Message) ([]map[string]any, error) {
	blocks := make([]map[string]any, 0, 1+len(m.Attachments)*2)
	if m.Text != "" {
		blocks = append(blocks, map[string]any{"type": "text", "text": m.Text})
	}
	if len(m.Attachments) > 10 {
		return nil, NewAPIError(CodeAttachmentLimit, "Too many attachments")
	}
	c.mu.Lock()
	home := c.transcriptHome
	c.mu.Unlock()
	var total int64
	for _, d := range m.Attachments {
		f, actual, err := attachment.New(home).Open(as.agentID, d.ID)
		if err != nil {
			return nil, NewAPIError(CodeAttachmentStorage, "Uploaded attachment is unavailable")
		}
		if actual != d {
			f.Close()
			return nil, NewAPIError(CodeAttachmentStorage, "Uploaded attachment identity changed")
		}
		total += actual.Size
		if actual.Size > 10*1024*1024 || total > 25*1024*1024 {
			f.Close()
			return nil, NewAPIError(CodeAttachmentLimit, "Attachment bytes exceed limits")
		}
		if actual.Kind == "image" {
			if imageInput := as.capabilities().ImageInput; imageInput == nil || !*imageInput {
				f.Close()
				return nil, ErrAttachmentCapability
			}
			b, err := io.ReadAll(io.LimitReader(f, 10*1024*1024+1))
			f.Close()
			if err != nil {
				return nil, NewAPIError(CodeAttachmentStorage, "Uploaded attachment could not be read")
			}
			blocks = append(blocks, map[string]any{"type": "image", "data": base64.StdEncoding.EncodeToString(b), "mimeType": actual.MIMEType})
		} else {
			f.Close()
			path, err := attachment.New(home).Path(as.agentID, d.ID)
			if err != nil {
				return nil, NewAPIError(CodeAttachmentStorage, "Uploaded attachment is unavailable")
			}
			uri := (&url.URL{Scheme: "file", Path: path}).String()
			blocks = append(blocks, map[string]any{"type": "resource_link", "uri": uri, "name": actual.Name, "mimeType": actual.MIMEType, "size": actual.Size}, map[string]any{"type": "text", "text": "Read the uploaded copy at " + uri + " through your ordinary file tools. File permissions still apply."})
		}
	}
	return blocks, nil
}
