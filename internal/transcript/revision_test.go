package transcript

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/AsaphNoam/Chuck/internal/runtime"
)

func TestSteerSettlementFoldsBeforeSinceFilter(t *testing.T) {
	home := t.TempDir()
	w, err := Open(home, "a_test", &runtime.SessionMetaData{})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	for _, ev := range []runtime.Event{
		event(t, 1, runtime.EvUserPrompt, runtime.UserPromptData{Text: "refused", SteerPending: true}),
		event(t, 2, runtime.EvUserPrompt, runtime.UserPromptData{Text: "confirmed", SteerPending: true}),
		event(t, 3, runtime.EvUserPrompt, runtime.UserPromptData{Text: "interrupted", SteerPending: true}),
		event(t, 4, runtime.EvSteerResolution, runtime.SteerResolutionData{PromptSeq: 1, Outcome: "retracted"}),
		event(t, 5, runtime.EvSteerResolution, runtime.SteerResolutionData{PromptSeq: 2, Outcome: "accepted"}),
	} {
		if err := w.Append(ev); err != nil {
			t.Fatal(err)
		}
	}
	events, err := ReadFile(home, "a_test", ReadOptions{})
	if err != nil || len(events) != 5 {
		t.Fatalf("active read: %v, %v", events, err)
	}
	if events[0].Type == runtime.EvUserPrompt || len(events[0].Data) != 0 {
		t.Fatalf("refusal exposed prompt: %+v", events[0])
	}
	events, err = ReadFile(home, "a_test", ReadOptions{SinceSeq: 1})
	if err != nil || len(events) != 4 {
		t.Fatalf("incremental read: %v, %v", events, err)
	}
	for i, pending := range []bool{false, true} {
		var prompt runtime.UserPromptData
		if err := json.Unmarshal(events[i].Data, &prompt); err != nil || prompt.SteerPending != pending {
			t.Fatalf("seq %d prompt: %+v, %v", events[i].Seq, prompt, err)
		}
	}
}

// revisedJournal writes three turns, then a revision replacing the third
// turn (seq 7–10) and its replacement turn. An annotation recorded inside the
// superseded turn is unrelated audit state and stays visible (TS-02.R70).
func revisedJournal(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	w, err := Open(home, "a_test", &runtime.SessionMetaData{})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	add := func(seq int64, typ string, data any) {
		t.Helper()
		if err := w.Append(event(t, seq, typ, data)); err != nil {
			t.Fatal(err)
		}
	}
	turn := func(first int64, text string) {
		add(first, runtime.EvUserPrompt, runtime.UserPromptData{Text: text})
		add(first+1, runtime.EvAssistantText, runtime.AssistantTextData{Delta: "reply to " + text})
		add(first+2, runtime.EvTurnEnd, map[string]string{"stop_reason": "end_turn"})
	}
	turn(1, "one")
	turn(4, "two")
	turn(7, "three")
	add(10, runtime.EvAnnotation, runtime.AnnotationData{OverallInstruction: "keep me"})
	add(11, runtime.EvConversationRevision, runtime.ConversationRevisionData{RequestID: "r1", Revision: 1, ReplacesFromSeq: 7, ReplacesThroughSeq: 10})
	turn(12, "three, revised")
	return home
}

func seqs(events []runtime.Event) []int64 {
	out := make([]int64, 0, len(events))
	for _, ev := range events {
		out = append(out, ev.Seq)
	}
	return out
}

func TestRevisionFoldHidesSupersededTurnFromActiveReads(t *testing.T) {
	home := revisedJournal(t)
	active := []int64{1, 2, 3, 4, 5, 6, 10, 11, 12, 13, 14}
	got, err := ReadFile(home, "a_test", ReadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(seqs(got), active) {
		t.Fatalf("active read = %v, want %v", seqs(got), active)
	}
	// A reader resuming after the superseded message but before the revision
	// record still has the boundary applied.
	since, err := ReadFile(home, "a_test", ReadOptions{SinceSeq: 7})
	if err != nil {
		t.Fatal(err)
	}
	if want := []int64{10, 11, 12, 13, 14}; !reflect.DeepEqual(seqs(since), want) {
		t.Fatalf("since 7 = %v, want %v", seqs(since), want)
	}
	raw, err := ReadFile(home, "a_test", ReadOptions{Raw: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) != 14 {
		t.Fatalf("raw read kept %d events, want all 14 retained", len(raw))
	}
}

func TestRevisionFoldAppliesToBoundedPrefixReadsWithIdenticalTee(t *testing.T) {
	home := revisedJournal(t)
	limit, err := CompletePrefix(home, "a_test")
	if err != nil {
		t.Fatal(err)
	}
	var tee bytes.Buffer
	var got []runtime.Event
	if err := ForEachFilePrefix(home, "a_test", limit, ReadOptions{Tee: &tee}, func(ev runtime.Event) error {
		got = append(got, ev)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if want := []int64{1, 2, 3, 4, 5, 6, 10, 11, 12, 13, 14}; !reflect.DeepEqual(seqs(got), want) {
		t.Fatalf("prefix read = %v, want %v", seqs(got), want)
	}
	// The fingerprint still covers the exact retained bytes once.
	file, err := os.ReadFile(transcriptPath(home, "a_test"))
	if err != nil {
		t.Fatal(err)
	}
	if sha256.Sum256(tee.Bytes()) != sha256.Sum256(file[:limit]) {
		t.Fatal("tee did not receive the retained prefix exactly once")
	}
}

func TestRevisionFoldUsesOnePrefixForBothPasses(t *testing.T) {
	home := revisedJournal(t)
	// A revision appended after a reader captured its prefix belongs to a later
	// read, never to a mixed page.
	limit, err := CompletePrefix(home, "a_test")
	if err != nil {
		t.Fatal(err)
	}
	w, err := Open(home, "a_test", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Append(event(t, 15, runtime.EvConversationRevision, runtime.ConversationRevisionData{RequestID: "r2", Revision: 2, ReplacesFromSeq: 12, ReplacesThroughSeq: 14})); err != nil {
		t.Fatal(err)
	}
	w.Close()
	var got []int64
	if err := ForEachFilePrefix(home, "a_test", limit, ReadOptions{}, func(ev runtime.Event) error {
		got = append(got, ev.Seq)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if want := []int64{1, 2, 3, 4, 5, 6, 10, 11, 12, 13, 14}; !reflect.DeepEqual(got, want) {
		t.Fatalf("captured prefix = %v, want %v", got, want)
	}
	latest, err := ReadFile(home, "a_test", ReadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if want := []int64{1, 2, 3, 4, 5, 6, 10, 11, 15}; !reflect.DeepEqual(seqs(latest), want) {
		t.Fatalf("after second revision = %v, want %v", seqs(latest), want)
	}
}

func TestLegacyJournalWithoutRevisionsReadsUnchanged(t *testing.T) {
	home := t.TempDir()
	w, err := Open(home, "a_test", &runtime.SessionMetaData{})
	if err != nil {
		t.Fatal(err)
	}
	for seq := int64(1); seq <= 3; seq++ {
		if err := w.Append(event(t, seq, runtime.EvAssistantText, runtime.AssistantTextData{Delta: "x"})); err != nil {
			t.Fatal(err)
		}
	}
	w.Close()
	active, err := ReadFile(home, "a_test", ReadOptions{IncludeMeta: true})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := ReadFile(home, "a_test", ReadOptions{IncludeMeta: true, Raw: true})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(active, raw) || len(active) != 4 {
		t.Fatalf("legacy active %v differs from raw %v", seqs(active), seqs(raw))
	}
	if _, err := ReadFile(t.TempDir(), "missing", ReadOptions{}); err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing transcript: %v", err)
	}
}

// A clone renumbers the folded prefix, so a copied source revision record
// would hide the clone's own active turns (EDIT-REVIEW-5, TS-02.R70).
func TestClonePrefixOmitsSourceRevisionRecords(t *testing.T) {
	home := revisedJournal(t)
	prefix, boundary, err := CloneCompletedPrefix(home, "a_test", 0)
	if err != nil {
		t.Fatal(err)
	}
	if boundary.Seq != 14 {
		t.Fatalf("boundary = %d, want the revised turn's end 14", boundary.Seq)
	}
	w, err := Open(home, "a_clone", &runtime.SessionMetaData{})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	for i, ev := range prefix {
		if ev.Type == runtime.EvConversationRevision {
			t.Fatalf("prefix copied source revision record %d", ev.Seq)
		}
		ev.AgentID, ev.Seq = "a_clone", int64(i+1)
		if err := w.Append(ev); err != nil {
			t.Fatal(err)
		}
	}
	got, err := ReadFile(home, "a_clone", ReadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(prefix) {
		t.Fatalf("clone active read kept %d of %d copied events", len(got), len(prefix))
	}
}
