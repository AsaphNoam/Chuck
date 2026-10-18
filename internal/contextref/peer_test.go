package contextref

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AsaphNoam/Chuck/internal/runtime"
)

// FS-06.R39/TS-04.R93: a record too large to decode after the last readable
// event still follows since_seq, so an incremental read reports its omission
// instead of "nothing new".
func TestPeerSinceSeqReportsTrailingOversized(t *testing.T) {
	f := newFixture(t)
	f.agent(t, "peer", "peer")
	f.appendEvents(t, "peer",
		ev(t, runtime.EvUserPrompt, runtime.UserPromptData{Text: "done"}),
		ev(t, runtime.EvTurnEnd, runtime.TurnEndData{StopReason: "end_turn"}),
		ev(t, runtime.EvAssistantText, runtime.AssistantTextData{Delta: strings.Repeat("x", 9*1024*1024)}),
	)
	got, err := f.svc.ReadPeerTranscript(t.Context(), "peer", 2, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got.Text, OversizedRecordMarker) {
		t.Fatalf("trailing oversized record omitted: %+v", got)
	}
	// A read starting past it does not mark a record before its range.
	got, err = f.svc.ReadPeerTranscript(t.Context(), "peer", 3, "")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got.Text, OversizedRecordMarker) {
		t.Fatalf("record before since_seq marked: %+v", got)
	}
}

// TS-02.R49: a cursor rejects a source whose prefix was replaced even when its
// length and last sequence are unchanged.
func TestPeerCursorRejectsSameBoundaryReplacement(t *testing.T) {
	f := newFixture(t)
	f.agent(t, "peer", "peer")
	for i := 0; i < 40; i++ {
		f.appendEvents(t, "peer", ev(t, runtime.EvUserPrompt, runtime.UserPromptData{Text: strings.Repeat("a", 1200)}))
	}
	first, err := f.svc.ReadPeerTranscript(t.Context(), "peer", 0, "")
	if err != nil || first.NextCursor == "" {
		t.Fatalf("first page: %+v %v", first, err)
	}
	if _, err := f.svc.ReadPeerTranscript(t.Context(), "peer", 0, first.NextCursor); err != nil {
		t.Fatalf("unchanged source continuation: %v", err)
	}
	path := filepath.Join(f.home, "sessions", "peer", "transcript.ndjson")
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	replaced := strings.Replace(string(original), "aaa", "bbb", 1)
	if err := os.WriteFile(path, []byte(replaced), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = f.svc.ReadPeerTranscript(t.Context(), "peer", 0, first.NextCursor)
	if code(err) != CodeTranscriptUnavailable {
		t.Fatalf("same-boundary replacement error = %v, want %s", err, CodeTranscriptUnavailable)
	}
}
