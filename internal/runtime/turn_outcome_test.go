package runtime

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestResolveTurnOutcome(t *testing.T) {
	quota := &QuotaEvidence{FailureID: "f1", Revision: 1}
	for _, tc := range []struct {
		name                     string
		provider                 TurnOutcome
		cancelled, ended, failed bool
		quota                    *QuotaEvidence
		want                     TurnOutcome
		reason                   string
	}{
		{"completion", TurnCompleted, false, false, false, nil, TurnCompleted, "end_turn"},
		{"late completion", TurnCompleted, true, false, false, nil, TurnCancelled, "cancelled"},
		{"terminal provider failure", TurnCompleted, false, false, true, nil, TurnFailed, "error"},
		{"quota refusal", TurnCompleted, false, false, true, quota, TurnQuotaBlocked, "error"},
		{"RPC error", TurnUnknown, false, false, true, nil, TurnFailed, "error"},
		{"process exit", TurnUnknown, false, true, false, nil, TurnProcessEnded, "error"},
		{"provider output limit", TurnProviderTruncated, false, false, false, nil, TurnProviderTruncated, "max_tokens"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			outcome := resolveTurnOutcome(tc.provider, tc.cancelled, tc.ended, tc.failed, tc.quota)
			if outcome != tc.want {
				t.Fatalf("resolved %v, want %v", outcome, tc.want)
			}
			raw, err := json.Marshal(outcome.eventData(contextReading{}, tc.quota))
			if err != nil {
				t.Fatal(err)
			}
			got, data, err := DecodeTurnEnd(Event{Type: EvTurnEnd, Data: raw})
			if err != nil || got != tc.want || data.StopReason != tc.reason {
				t.Fatalf("decoded %v, %s, %v; want %v / %s", got, raw, err, tc.want, tc.reason)
			}
		})
	}
}

func TestDecodeHistoricalTurnEnd(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		want TurnOutcome
	}{
		{`{"stop_reason":"end_turn"}`, TurnCompleted},
		{`{"stop_reason":"cancelled"}`, TurnCancelled},
		{`{"stop_reason":"max_tokens"}`, TurnProviderTruncated},
		{`{"stop_reason":"max_turn_requests"}`, TurnRequestLimit},
		{`{"stop_reason":"refusal"}`, TurnRefused},
		{`{"stop_reason":"error"}`, TurnFailed},
		{`{"stop_reason":"error","quota":{"failure_id":"old","revision":1}}`, TurnQuotaBlocked},
		{`{"stop_reason":"future_value"}`, TurnUnknown},
		{`{}`, TurnUnknown},
	} {
		got, _, err := DecodeTurnEnd(Event{Type: EvTurnEnd, Data: json.RawMessage(tc.raw)})
		if err != nil || got != tc.want {
			t.Errorf("%s decoded as %v, %v; want %v", tc.raw, got, err, tc.want)
		}
	}
}

func TestTurnOutcomeOwnsRawReasonClassification(t *testing.T) {
	// Match selectors, direct JSON map reads and alternate payload structs.
	// Projection may copy the raw field as text without classifying it.
	raw := regexp.MustCompile(`\.StopReason\b|\["stop_reason"\]|json:"stop_reason"`)
	err := filepath.WalkDir("..", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() && (entry.Name() == "testdata" || entry.Name() == "dist") {
			return filepath.SkipDir
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") ||
			path == "../runtime/turn_outcome.go" || path == "../runtime/event.go" {
			return nil
		}
		source, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, line := range strings.Split(string(source), "\n") {
			if path == "../transcript/project.go" && strings.Contains(line, "Kind: PartTurnStopReason, Text: d.StopReason") {
				continue
			}
			if raw.MatchString(line) {
				t.Errorf("%s reads the raw turn reason: %s", path, strings.TrimSpace(line))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// The browser renders the server's decoded outcome on desktop and phone,
	// live and retained; it never reads the raw reason to classify a turn.
	scanned := 0
	err = filepath.WalkDir("../../ui/src", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !(strings.HasSuffix(path, ".ts") || strings.HasSuffix(path, ".tsx")) ||
			strings.Contains(path, ".test.") || strings.Contains(path, "/test/") {
			return nil
		}
		source, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		scanned++
		for _, line := range strings.Split(string(source), "\n") {
			if strings.Contains(line, "stop_reason") || strings.Contains(line, "stopReason") {
				t.Errorf("%s reads the raw turn reason: %s", path, strings.TrimSpace(line))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if scanned == 0 {
		t.Fatal("no browser sources scanned")
	}
}
