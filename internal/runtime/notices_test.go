package runtime

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func runNoticeTurn(t *testing.T, env ...string) []string {
	t.Helper()
	c, spec := newChatTest(t, "notice_flow")
	spec.Env = append(spec.Env, env...)
	ctx := context.Background()
	h, err := c.Start(ctx, spec)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = c.Stop(ctx, h.AgentID) })
	ch, unsub, err := c.Subscribe(h.AgentID)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	defer unsub()
	if err := c.SendPrompt(ctx, h.AgentID, "go"); err != nil {
		t.Fatalf("SendPrompt: %v", err)
	}
	var rows []string
	for _, ev := range drainTurn(t, ch) {
		switch ev.Type {
		case EvAssistantText:
			var d AssistantTextData
			_ = json.Unmarshal(ev.Data, &d)
			rows = append(rows, "text|"+d.Delta)
		case EvNotice:
			rows = append(rows, "notice|"+string(ev.Data))
		}
	}
	return rows
}

// FS-03.A49, TS-04.R80: offered notices become bounded `notice` events in
// stream order; an unknown severity reads as info and an untitled notice is
// dropped. The fake gates notices on the offer exactly as Claude 0.85.1 does.
func TestNoticesMapToDurableNoticeEvents(t *testing.T) {
	got := runNoticeTurn(t)
	want := []string{
		"text|Before.",
		`notice|{"severity":"info","title":"Context compacted"}`,
		`notice|{"severity":"warning","title":"Usage limit approaching","description":"Five-hour limit at 90%."}`,
		`notice|{"severity":"info","title":"Hook failed"}`,
		"text|After.",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("rows =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// An adapter without notice support keeps its message-text fallback; Chuck
// does not reinterpret that text as a notice (FS-03.R68).
func TestNoticeFallbackTextStaysText(t *testing.T) {
	for _, row := range runNoticeTurn(t, "FAKEACP_NO_NOTICES=1") {
		if strings.HasPrefix(row, "notice|") {
			t.Fatalf("fallback produced a notice: %s", row)
		}
	}
}

func TestNoticeBounds(t *testing.T) {
	long := strings.Repeat("é", maxNoticeDescription+10)
	params, _ := json.Marshal(map[string]any{"sessionId": "s", "update": map[string]any{
		"sessionUpdate": "notice", "severity": "warning", "title": long, "description": long,
	}})
	evs := mapSessionUpdate(params)
	if len(evs) != 1 {
		t.Fatalf("events = %d", len(evs))
	}
	d := evs[0].Data.(NoticeData)
	if n := len([]rune(d.Title)); n != maxNoticeTitle {
		t.Fatalf("title runes = %d", n)
	}
	if n := len([]rune(d.Description)); n != maxNoticeDescription {
		t.Fatalf("description runes = %d", n)
	}
}

// TS-04.R108 bounds: oversized detail drops bodies from the last message
// backward, keeps every header, and says it was truncated.
func TestMailNoticeDataClipsBodiesFromTheEnd(t *testing.T) {
	big := strings.Repeat("x", 40*1024)
	messages := []MailDetail{
		{FromName: "A", Subject: "first", Body: "keep me"},
		{FromName: "B", Subject: "second", Body: big},
		{FromName: "C", Subject: "third", Body: big},
		{FromName: "D", Subject: "fourth", Body: big},
	}
	data := MailNoticeData("Mail from A", "", MailNotice{Direction: "received", Messages: messages})
	raw, err := json.Marshal(data.Mail)
	if err != nil {
		t.Fatal(err)
	}
	got := data.Mail.Messages
	if len(raw) > maxMailNoticeBytes || !data.Mail.Truncated || data.Origin != MailNoticeOrigin {
		t.Fatalf("detail bytes=%d truncated=%v origin=%q", len(raw), data.Mail.Truncated, data.Origin)
	}
	if got[0].Body != "keep me" || got[1].Body != big || got[2].Body != "" || got[3].Body != "" || got[3].Subject != "fourth" {
		t.Fatalf("clip kept bodies %q/%d/%d/%d, want first two only", got[0].Body, len(got[1].Body), len(got[2].Body), len(got[3].Body))
	}
	if messages[3].Body != big {
		t.Fatal("clipping mutated the caller's messages")
	}
	small := MailNoticeData("t", "", MailNotice{Direction: "read", Messages: messages[:1]})
	if small.Mail.Truncated || small.Mail.Messages[0].Body != "keep me" {
		t.Fatalf("small detail was clipped: %+v", small.Mail)
	}
}
