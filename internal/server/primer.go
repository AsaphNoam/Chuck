package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/transcript"
)

const (
	defaultPrimerTokenBudget = 8000
	primerTailTurns          = 6
)

type primerSummarizer func(context.Context, primerSummaryRequest) (string, error)

type primerSummaryRequest struct {
	AgentID    string
	Target     string
	Backend    string
	Model      string
	Transcript string
}

func defaultPrimerSummarizer(context.Context, primerSummaryRequest) (string, error) {
	return "", fmt.Errorf("target-backend one-shot summary is not configured")
}

func (s *Server) buildHistoryPrimer(ctx context.Context, target runtime.LaunchSpec, budget int) (string, error) {
	if budget <= 0 {
		budget = defaultPrimerTokenBudget
	}
	b := newPrimerBuilder(budget)
	err := transcript.ForEachFile(s.configStore.Home(), target.Agent.AgentID, transcript.ReadOptions{
		OnOversized: func(int64) error {
			b.append("\n[oversized transcript record omitted]\n")
			b.delta = false
			return nil
		},
	}, b.visit)
	if err != nil {
		return "", err
	}
	return b.render(ctx, target, s.primerSummarizer), nil
}

func synthesizeHistoryPrimer(ctx context.Context, events []runtime.Event, target runtime.LaunchSpec, budget int, summarize primerSummarizer) (string, error) {
	if budget <= 0 {
		budget = defaultPrimerTokenBudget
	}
	b := newPrimerBuilder(budget)
	for _, ev := range events {
		if err := b.visit(ev); err != nil {
			return "", err
		}
	}
	return b.render(ctx, target, summarize), nil
}

func appendBackendSwitchMarker(home string, agentID, from, to string, at time.Time) error {
	w, err := transcript.Open(home, agentID, nil)
	if err != nil {
		return err
	}
	defer w.Close()
	when := at.UTC().Format(time.RFC3339)
	data, err := json.Marshal(runtime.BackendSwitchData{From: from, To: to, At: when})
	if err != nil {
		return err
	}
	return w.Append(runtime.Event{
		AgentID: agentID,
		Type:    runtime.EvBackendSwitch,
		Data:    data,
		Ts:      when,
	})
}

// Each retained turn and older excerpt is capped independently while ingesting.
// The reader itself caps a physical record at 8 MiB.
type primerBuilder struct {
	budget         int
	current, older string
	tail           []string
	delta          bool
}

func newPrimerBuilder(budget int) *primerBuilder {
	if budget <= 0 {
		budget = defaultPrimerTokenBudget
	}
	return &primerBuilder{budget: budget}
}

func (b *primerBuilder) append(text string) {
	b.current = clipPrimerTail(b.current+text, b.budget*4)
}

func (b *primerBuilder) flush() {
	if strings.TrimSpace(b.current) != "" {
		b.tail = append(b.tail, b.current)
		if len(b.tail) > primerTailTurns {
			b.older = clipPrimerTail(b.older+"\n\n"+b.tail[0], b.budget*4)
			b.tail = b.tail[1:]
		}
	}
	b.current, b.delta = "", false
}

func (b *primerBuilder) visit(ev runtime.Event) error {
	if ev.ActivityID != "" {
		return nil
	}
	p, err := transcript.ProjectEvent(ev)
	if err != nil {
		b.append("\n[malformed transcript event omitted]\n")
		b.delta = false
		return nil
	}
	if ev.Type == runtime.EvTurnEnd {
		b.flush()
		return nil
	}
	if p.Disposition == transcript.DispositionMetadata {
		return nil
	}
	if ev.Type == runtime.EvUserPrompt {
		b.flush()
	}
	if p.Disposition == transcript.DispositionUnknown {
		b.append("\n[unknown transcript event omitted]\n")
		b.delta = false
		return nil
	}
	for _, part := range p.Parts {
		if part.Text == "" {
			continue
		}
		if part.Kind == transcript.PartAssistantDelta {
			if !b.delta {
				b.append("\nassistant: ")
			}
			b.append(part.Text)
			b.delta = true
		} else {
			b.append("\n" + p.Role + " " + string(part.Kind) + ": " + part.Text)
			b.delta = false
		}
	}
	return nil
}

func (b *primerBuilder) render(ctx context.Context, target runtime.LaunchSpec, summarize primerSummarizer) string {
	b.flush()
	header := "Chuck backend-switch history primer. Bounded transcript excerpts; omitted context may be unavailable.\n\nRecent turns (newest text retained; older text may be truncated):\n"
	maxBytes := b.budget * 4
	if len(header) >= maxBytes {
		return truncateToBudget(header, b.budget)
	}
	tail := clipPrimerTail(strings.Join(b.tail, "\n\n"), maxBytes-len(header))
	result := header + tail
	label, older := "\n\nOlder context excerpts (most recent retained; may be truncated):\n", b.older
	if older != "" && summarize != nil {
		got, err := summarize(ctx, primerSummaryRequest{AgentID: target.Agent.AgentID, Target: target.Agent.Backend + "/" + target.Agent.Model, Backend: target.BackendType, Model: target.ModelID, Transcript: older})
		if err == nil && strings.TrimSpace(got) != "" {
			older, label = got, "\n\nSummary of bounded older excerpts:\n"
		}
	}
	if room := maxBytes - len(result) - len(label); older != "" && room > 0 {
		result += label + clipPrimerTail(older, room)
	}
	return result
}

func clipPrimerTail(s string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	if len(s) <= maxBytes {
		return s
	}
	marker := "[truncated]\n"
	if len(marker) >= maxBytes {
		return marker[:maxBytes]
	}
	start := len(s) - (maxBytes - len(marker))
	for start < len(s) && !utf8.RuneStart(s[start]) {
		start++
	}
	return marker + s[start:]
}

func truncateToBudget(s string, budget int) string {
	if budget <= 0 {
		return ""
	}
	maxChars := budget*4 - 3
	if len(s) <= maxChars {
		return s
	}
	if maxChars <= len("...[truncated]") {
		return clipPrimerTail(s, maxChars)
	}
	end := maxChars - len("\n...[truncated]")
	for end > 0 && !utf8.RuneStart(s[end]) {
		end--
	}
	return s[:end] + "\n...[truncated]"
}
