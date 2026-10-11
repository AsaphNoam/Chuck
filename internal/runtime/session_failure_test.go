package runtime

import (
	"encoding/json"
	"strings"
	"testing"
)

// These envelopes are copied from the pinned Claude 0.85.1 and Codex 2.1.1
// producers' AIR sessionFailure publishers. They intentionally contain no reset.
const (
	claudeQuotaPromptResult = `{"stopReason":"end_turn","_meta":{"jetbrains":{"air":{"version":1,"sessionFailure":{"id":"turn-1:error","revision":1,"category":"limit","severity":"error","title":"The Claude account has no available quota.","actions":[]}}}}}`
	codexQuotaUpdate        = `{"sessionId":"s1","update":{"sessionUpdate":"session_info_update","_meta":{"jetbrains":{"air":{"version":1,"sessionFailure":{"id":"turn-2:error","revision":1,"category":"limit","severity":"error","title":"Codex usage limit exceeded","actions":[]}}}}}}`
)

func TestDecodeSessionFailurePinnedQuotaShapes(t *testing.T) {
	for _, raw := range []string{claudeQuotaPromptResult, codexQuotaUpdate} {
		got, ok := decodeSessionFailure(json.RawMessage(raw))
		if !ok || got.FailureID == "" || got.Revision != 1 {
			t.Fatalf("decodeSessionFailure(%s) = %+v, %v; want bounded quota evidence", raw, got, ok)
		}
	}
	td, _ := mapPromptResult(json.RawMessage(claudeQuotaPromptResult))
	if td.StopReason != "error" || td.Quota == nil || td.Quota.FailureID != "turn-1:error" {
		t.Fatalf("mapPromptResult quota = %+v, want error with Claude evidence", td)
	}
}

func TestDecodeSessionFailureRequiresCompleteTerminalQuotaPolicy(t *testing.T) {
	base := `{"_meta":{"jetbrains":{"air":{"version":1,"sessionFailure":{"id":"f1","revision":1,"category":"limit","severity":"error","title":"Quota exhausted","actions":[]}}}}}`
	for _, tc := range []struct {
		name string
		raw  string
	}{
		{"warning is not terminal", strings.Replace(base, `"severity":"error"`, `"severity":"warning"`, 1)},
		{"retry policy is rate limiting", strings.Replace(base, `"actions":[]`, `"actions":["retry"]`, 1)},
		{"new session policy is context budget", strings.Replace(base, `"actions":[]`, `"actions":["new_session"]`, 1)},
		{"wrong category", strings.Replace(base, `"category":"limit"`, `"category":"service"`, 1)},
		{"missing revision", strings.Replace(base, `,"revision":1`, ``, 1)},
		{"missing actions", strings.Replace(base, `,"actions":[]`, ``, 1)},
		{"not session info update", strings.Replace(codexQuotaUpdate, `"session_info_update"`, `"notice"`, 1)},
		{"oversized envelope", strings.Repeat(" ", maxSessionFailureBytes+1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got, ok := decodeSessionFailure(json.RawMessage(tc.raw)); ok {
				t.Fatalf("unexpected quota evidence: %+v", got)
			}
		})
	}
}

func TestDecodeSessionFailureRecordSupportsSameIDRecoveryRevision(t *testing.T) {
	recovered := strings.Replace(claudeQuotaPromptResult,
		`"revision":1,"category":"limit","severity":"error"`,
		`"revision":2,"category":"service","severity":"warning"`, 1)
	record, ok := decodeSessionFailureRecord(json.RawMessage(recovered))
	if !ok || record.ID != "turn-1:error" || record.Revision != 2 || record.Quota {
		t.Fatalf("recovery revision = %+v, %v; want same-ID non-quota revision 2", record, ok)
	}
	td, _ := mapPromptResult(json.RawMessage(claudeQuotaPromptResult))
	if td.Quota == nil || td.Quota.FailureID != "turn-1:error" {
		t.Fatalf("turn quota evidence = %+v", td.Quota)
	}
}
