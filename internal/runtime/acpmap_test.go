package runtime

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
)

// Regression (review fix): a tool_call_update must only produce a tool_result on a
// TERMINAL status (completed/failed). An in-progress update — or one that omits
// status — previously defaulted to "completed", flipping the tool to done early
// and repeating tool_results in the transcript.
func TestMapToolCallUpdateOnlyTerminalStatusEmitsResult(t *testing.T) {
	countResults := func(raw string) int {
		n := 0
		for _, e := range mapSessionUpdate(json.RawMessage(raw)) {
			if e.Type == EvToolResult {
				n++
			}
		}
		return n
	}

	cases := []struct {
		name string
		raw  string
		want int
	}{
		{"in_progress", `{"update":{"sessionUpdate":"tool_call_update","toolCallId":"tc1","status":"in_progress"}}`, 0},
		{"no_status", `{"update":{"sessionUpdate":"tool_call_update","toolCallId":"tc1"}}`, 0},
		{"completed", `{"update":{"sessionUpdate":"tool_call_update","toolCallId":"tc1","status":"completed"}}`, 1},
		{"failed", `{"update":{"sessionUpdate":"tool_call_update","toolCallId":"tc1","status":"failed"}}`, 1},
	}
	for _, c := range cases {
		if got := countResults(c.raw); got != c.want {
			t.Errorf("%s: %d tool_result(s), want %d", c.name, got, c.want)
		}
	}
}

// TC-4 regression (TS-04.R99): Codex ACP reports a failed MCP call with no
// content and the reason in rawOutput.error.message. The reason must reach the
// tool result's Error, bounded; absent or unknown rawOutput keeps Error empty.
func TestMapFailedToolCallUpdateKeepsRawOutputReason(t *testing.T) {
	resultOf := func(t *testing.T, raw string) ToolResultData {
		t.Helper()
		for _, e := range mapSessionUpdate(json.RawMessage(raw)) {
			if e.Type == EvToolResult {
				return e.Data.(ToolResultData)
			}
		}
		t.Fatalf("no tool_result for %s", raw)
		return ToolResultData{}
	}
	const reason = "MCP tool call requires approval, but approval policy is never"
	denied := `{"update":{"sessionUpdate":"tool_call_update","toolCallId":"tc1","status":"failed",
	  "rawInput":{"server":"chuck","tool":"report_task_result","arguments":{}},
	  "rawOutput":{"result":null,"error":{"message":" ` + reason + ` "}}}}`
	got := resultOf(t, denied)
	if got.Error != reason || string(got.Content) != "null" || got.Status != "failed" {
		t.Fatalf("denied MCP call = %+v, want Error %q and null content", got, reason)
	}

	long := `{"update":{"sessionUpdate":"tool_call_update","toolCallId":"tc1","status":"failed",
	  "rawOutput":{"result":null,"error":{"message":"` + strings.Repeat("é", maxToolFailureReason+50) + `"}}}}`
	if n := len([]rune(resultOf(t, long).Error)); n != maxToolFailureReason {
		t.Fatalf("long reason = %d runes, want %d", n, maxToolFailureReason)
	}

	for name, raw := range map[string]string{
		"absent":          `{"update":{"sessionUpdate":"tool_call_update","toolCallId":"tc1","status":"failed"}}`,
		"string":          `{"update":{"sessionUpdate":"tool_call_update","toolCallId":"tc1","status":"failed","rawOutput":"boom"}}`,
		"error_string":    `{"update":{"sessionUpdate":"tool_call_update","toolCallId":"tc1","status":"failed","rawOutput":{"error":"boom"}}}`,
		"result_no_err":   `{"update":{"sessionUpdate":"tool_call_update","toolCallId":"tc1","status":"failed","rawOutput":{"result":{"content":[]},"error":null}}}`,
		"completed_err":   `{"update":{"sessionUpdate":"tool_call_update","toolCallId":"tc1","status":"completed","rawOutput":{"error":{"message":"x"}}}}`,
		"content_present": `{"update":{"sessionUpdate":"tool_call_update","toolCallId":"tc1","status":"failed","content":[{"type":"content","content":{"type":"text","text":"out"}}],"rawOutput":{"error":{"message":"x"}}}}`,
	} {
		if got := resultOf(t, raw); got.Error != "" {
			t.Errorf("%s: Error = %q, want empty (unchanged behavior)", name, got.Error)
		}
	}
}

// A diff block still streams on an in-progress update even though no tool_result
// is emitted for it.
func TestMapToolCallUpdateEmitsDiffOnInProgress(t *testing.T) {
	raw := `{"update":{"sessionUpdate":"tool_call_update","toolCallId":"tc1","status":"in_progress",
	  "content":[{"type":"diff","path":"a.go","oldText":"x","newText":"y"}]}}`
	var diffs, results int
	for _, e := range mapSessionUpdate(json.RawMessage(raw)) {
		switch e.Type {
		case EvDiff:
			diffs++
		case EvToolResult:
			results++
		}
	}
	if diffs != 1 || results != 0 {
		t.Fatalf("in-progress diff update: diffs=%d results=%d, want diffs=1 results=0", diffs, results)
	}
}

// TestContextUsageFromRealClaudeAdapterShapes is the reproduction for the field
// bug "the context bar is broken, just stuck at 0%" (HANDOFF review finding;
// governs FS-02.R26 and the ACP→context_pct mapping under TS-04). Committed
// SKIPPED so the tree stays green; the /fix session un-skips it as the
// regression.
//
// The pinned Claude adapter 0.59.0 reports token accounting in its prompt
// result and context usage in a separate usage_update notification:
//
//  1. The session/prompt result `usage` object is
//     {inputTokens,outputTokens,cachedReadTokens,cachedWriteTokens,totalTokens}
//     — it has no `used` and no `window`, so it cannot produce context_pct.
//  2. The adapter also emits a session/update notification
//     {sessionUpdate:"usage_update", used:<tokens>, size:<contextWindow>} — the
//     actual context channel.
//
// fakeacp now emits these true shapes too (INV §11).
func TestContextUsageFromRealClaudeAdapterShapes(t *testing.T) {
	// (1) Real prompt-result usage shape: token counts only, no used/window.
	result := `{"stopReason":"end_turn","usage":{"inputTokens":42000,"outputTokens":800,` +
		`"cachedReadTokens":1200,"cachedWriteTokens":0,"totalTokens":44000}}`
	td, hasPct := mapPromptResult(json.RawMessage(result))
	if hasPct || td.ContextPct != 0 {
		t.Fatalf("prompt-result usage produced context_pct=%v hasPct=%v, want no context fraction without a window", td.ContextPct, hasPct)
	}

	// (2) Real usage_update session notification: used + size (context window).
	// The exact pair is retained as reported, including over-window usage,
	// while only the percentage is capped (TS-04.R68).
	update := func(fields string) string {
		return `{"sessionId":"s","update":{"sessionUpdate":"usage_update",` + fields + `}}`
	}
	for _, tc := range []struct {
		name     string
		raw      string
		wantPct  float64
		wantUsed int64
		wantSize int64
		wantOK   bool
	}{
		{"normal", update(`"used":44000,"size":200000`), 0.22, 44000, 200000, true},
		{"zero", update(`"used":0,"size":200000`), 0, 0, 200000, true},
		{"capped", update(`"used":210000,"size":200000`), 1, 210000, 200000, true},
		{"negative ignored", update(`"used":-1,"size":200000`), 0, 0, 0, false},
		{"zero window ignored", update(`"used":44000,"size":0`), 0, 0, 0, false},
		{"missing used ignored", update(`"size":200000`), 0, 0, 0, false},
		{"missing size ignored", update(`"used":44000`), 0, 0, 0, false},
		{"fractional ignored", update(`"used":1.5,"size":200000`), 0, 0, 0, false},
		{"string ignored", update(`"used":"44000","size":200000`), 0, 0, 0, false},
		{"out of range ignored", update(`"used":44000,"size":99999999999999999999`), 0, 0, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := decodeContextUsage(json.RawMessage(tc.raw))
			if ok != tc.wantOK || got.pct != tc.wantPct {
				t.Fatalf("usage_update = pct=%v ok=%v, want %v/%v", got.pct, ok, tc.wantPct, tc.wantOK)
			}
			if !tc.wantOK {
				return
			}
			if got.counts == nil || got.counts.ContextUsed != tc.wantUsed || got.counts.ContextSize != tc.wantSize {
				t.Fatalf("usage_update counts = %+v, want %d/%d", got.counts, tc.wantUsed, tc.wantSize)
			}
		})
	}
}

func TestValidInlineReceiptRequiresRecognizedNonCancelledResult(t *testing.T) {
	for _, tc := range []struct {
		name string
		raw  string
		want bool
	}{
		{"completed", `{"stopReason":"end_turn"}`, true},
		{"token limit", `{"stopReason":"max_tokens"}`, true},
		{"cancelled", `{"stopReason":"cancelled"}`, false},
		{"missing reason", `{}`, false},
		{"malformed", `{`, false},
		{"unknown", `{"stopReason":"future_value"}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := validInlineReceipt(json.RawMessage(tc.raw)); got != tc.want {
				t.Fatalf("validInlineReceipt(%s) = %v, want %v", tc.raw, got, tc.want)
			}
		})
	}
}

// commandsUpdateParams builds an available_commands_update session/update params
// with a raw availableCommands array (TS-04.R24).
func commandsUpdateParams(commandsJSON string) json.RawMessage {
	return json.RawMessage(`{"sessionId":"s","update":{"sessionUpdate":"available_commands_update","availableCommands":` + commandsJSON + `}}`)
}

// TestDecodeAvailableCommands covers the ACP command decode boundary (TS-04.R24):
// valid mapping incl. the optional input hint, skipping nameless entries without
// losing valid ones (INV §7), rune-clamping (INV §8), the entry cap, empty-clears,
// and ok=false for a non-command update so mapping falls through.
func TestDecodeAvailableCommands(t *testing.T) {
	items, ok := decodeAvailableCommands(commandsUpdateParams(
		`[{"name":"review","description":"Review code","input":{"hint":"branch"}},{"description":"no name"},{"name":"$plan","description":"Plan"}]`))
	if !ok {
		t.Fatalf("ok = false, want true for available_commands_update")
	}
	if len(items) != 2 {
		t.Fatalf("items = %d, want 2 (nameless skipped); got %+v", len(items), items)
	}
	if items[0].Name != "review" || items[0].Description != "Review code" || items[0].InputHint != "branch" {
		t.Fatalf("item[0] = %+v, want review/Review code/branch", items[0])
	}
	if items[1].Name != "$plan" || items[1].InputHint != "" {
		t.Fatalf("item[1] = %+v, want $plan with no hint", items[1])
	}

	// Empty update returns an empty (non-nil) slice that clears the snapshot.
	empty, ok := decodeAvailableCommands(commandsUpdateParams(`[]`))
	if !ok || empty == nil || len(empty) != 0 {
		t.Fatalf("empty update: ok=%v items=%v, want ok=true, empty non-nil", ok, empty)
	}

	// A non-command update is not recognized so mapping proceeds normally.
	if _, ok := decodeAvailableCommands(json.RawMessage(`{"update":{"sessionUpdate":"agent_message_chunk"}}`)); ok {
		t.Fatalf("ok = true for agent_message_chunk, want false")
	}

	// Rune clamping and the entry cap.
	long := `[{"name":"` + strings.Repeat("n", 300) + `","description":"` + strings.Repeat("d", 3000) + `"}]`
	clamped, _ := decodeAvailableCommands(commandsUpdateParams(long))
	if len([]rune(clamped[0].Name)) != maxCommandName || len([]rune(clamped[0].Description)) != maxCommandText {
		t.Fatalf("clamp: name=%d desc=%d, want %d/%d", len([]rune(clamped[0].Name)), len([]rune(clamped[0].Description)), maxCommandName, maxCommandText)
	}
	var many []string
	for i := 0; i < maxCommandEntries+10; i++ {
		many = append(many, `{"name":"c`+strconv.Itoa(i)+`","description":""}`)
	}
	capped, _ := decodeAvailableCommands(commandsUpdateParams(`[` + strings.Join(many, ",") + `]`))
	if len(capped) != maxCommandEntries {
		t.Fatalf("cap: items=%d, want %d", len(capped), maxCommandEntries)
	}
}

// TestAvailableCommandsSnapshotReplaceOnly proves the runtime snapshot is
// replace-only live state (TS-04.R24): each update overwrites the prior list, an
// empty update clears it, and an unknown agent yields ErrNoHandle.
func TestAvailableCommandsSnapshotReplaceOnly(t *testing.T) {
	c := NewChatRuntime(nil)
	as := &agentState{agentID: "a1"}
	c.agents["a1"] = as

	c.onNotification(as, "session/update", commandsUpdateParams(`[{"name":"review","description":"Review"}]`))
	got, err := c.Commands("a1")
	if err != nil || len(got) != 1 || got[0].Name != "review" {
		t.Fatalf("after first update: got=%+v err=%v, want [review]", got, err)
	}

	c.onNotification(as, "session/update", commandsUpdateParams(`[{"name":"plan","description":"Plan"}]`))
	got, _ = c.Commands("a1")
	if len(got) != 1 || got[0].Name != "plan" {
		t.Fatalf("after replace: got=%+v, want [plan] only (not merged)", got)
	}

	c.onNotification(as, "session/update", commandsUpdateParams(`[]`))
	got, _ = c.Commands("a1")
	if len(got) != 0 {
		t.Fatalf("after empty update: got=%+v, want cleared", got)
	}

	if _, err := c.Commands("nope"); err == nil {
		t.Fatalf("Commands(unknown) err = nil, want ErrNoHandle")
	}
}

// TS-04.R98: Claude surfaces a streamed call with `{}` and refines it with
// status-less updates. Each changed input becomes one update carrying the
// call's known name; a repeat, a frame without input, and an update after the
// result add nothing.
func TestToolCallUpdateRecordsLaterInputOnce(t *testing.T) {
	frame := func(update string) []mappedEvent {
		return mapSessionUpdate(json.RawMessage(`{"sessionId":"s","update":` + update + `}`))
	}
	as := &agentState{toolNames: map[string]string{"t1": "edit"}}
	admitted := func(ms []mappedEvent) []mappedEvent {
		var out []mappedEvent
		for _, m := range ms {
			if m, ok := as.admitToolInput(activityScope{}, m); ok {
				out = append(out, m)
			}
		}
		return out
	}
	if got := admitted(frame(`{"sessionUpdate":"tool_call","toolCallId":"t1","kind":"edit","rawInput":{}}`)); len(got) != 1 || got[0].Type != EvToolCall {
		t.Fatalf("call = %+v", got)
	}
	if got := admitted(frame(`{"sessionUpdate":"tool_call_update","toolCallId":"t1","rawInput":{}}`)); len(got) != 0 {
		t.Fatalf("empty input frame = %+v, want nothing", got)
	}
	refined := `{"sessionUpdate":"tool_call_update","toolCallId":"t1","kind":"edit","title":"Edit a.go","rawInput":{"file_path":"a.go"}}`
	got := admitted(frame(refined))
	if len(got) != 1 || got[0].Type != EvToolCallUpdate {
		t.Fatalf("refinement = %+v", got)
	}
	if d := got[0].Data.(ToolCallUpdateData); d.Name != "edit" || d.Title != "Edit a.go" || string(d.Args) != `{"file_path":"a.go"}` {
		t.Fatalf("update data = %+v", d)
	}
	if got := admitted(frame(refined)); len(got) != 0 {
		t.Fatalf("repeat = %+v, want nothing", got)
	}
	// A title-only frame keeps the recorded input, so it is a repeat too.
	if got := admitted(frame(`{"sessionUpdate":"tool_call_update","toolCallId":"t1","title":"Edit a.go","status":"in_progress"}`)); len(got) != 0 {
		t.Fatalf("title-only repeat = %+v, want nothing", got)
	}
	if got := admitted(frame(`{"sessionUpdate":"tool_call_update","toolCallId":"t1","status":"completed"}`)); len(got) != 1 || got[0].Type != EvToolResult {
		t.Fatalf("result = %+v", got)
	}
	if got := admitted(frame(`{"sessionUpdate":"tool_call_update","toolCallId":"t1","title":"Late","rawInput":{"x":1}}`)); len(got) != 0 {
		t.Fatalf("update after result = %+v, want nothing", got)
	}
	if len(as.toolInputs) != 0 {
		t.Fatalf("call memory retained after result: %v", as.toolInputs)
	}
}

// A canonical name wins over a misleading title while an unnamed event keeps
// the kind/title fallback; the permission's display name follows the same rule
// but its auto-approve identity stays the title (TS-04.R66, FS-03.A42).
func TestCanonicalToolNameWinsOverFallback(t *testing.T) {
	named := mapSessionUpdate(json.RawMessage(`{"sessionId":"s","update":{"sessionUpdate":"tool_call","toolCallId":"t1","name":"exec_command","kind":"execute","title":"Editing files"}}`))
	unnamed := mapSessionUpdate(json.RawMessage(`{"sessionId":"s","update":{"sessionUpdate":"tool_call","toolCallId":"t2","kind":"edit","title":"Editing files"}}`))
	if got := named[0].Data.(ToolCallData); got.Name != "exec_command" || got.Title != "Editing files" {
		t.Fatalf("named = %+v", got)
	}
	if got := unnamed[0].Data.(ToolCallData); got.Name != "edit" {
		t.Fatalf("unnamed = %+v", got)
	}
	perm := json.RawMessage(`{"sessionId":"s","toolCall":{"toolCallId":"t1","name":"exec_command","title":"mcp__chuck__send"},"options":[]}`)
	if data, _ := mapPermissionRequest(perm, "", false); data.Name != "exec_command" {
		t.Fatalf("permission name = %q", data.Name)
	}
	if id := permissionToolIdentity(perm); id != "mcp__chuck__send" {
		t.Fatalf("permission identity = %q", id)
	}
}

// The read loop routes each session/update by one peek; a malformed frame has
// no kind, so no kind-specific decoder runs for it (INV §16).
func TestPeekSessionUpdate(t *testing.T) {
	for _, tc := range []struct{ raw, id, kind string }{
		{`{"sessionId":"child-1","update":{"sessionUpdate":"usage_update","used":1,"size":2}}`, "child-1", "usage_update"},
		{`{"update":{"sessionUpdate":"tool_call_update","content":[{"type":"content"}]}}`, "", "tool_call_update"},
		{`{"sessionId":"s","update":{"sessionUpdate":"agent_thought_chunk"`, "", ""},
	} {
		if id, kind := peekSessionUpdate(json.RawMessage(tc.raw)); id != tc.id || kind != tc.kind {
			t.Fatalf("peek %s = %q/%q, want %q/%q", tc.raw, id, kind, tc.id, tc.kind)
		}
	}
}
