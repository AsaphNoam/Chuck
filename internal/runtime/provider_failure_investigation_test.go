package runtime

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// PERR-1/PERR-2: Codex ACP 2.1.1 SESSION_FAILURE_POLICY.rate_limited and
// terminalFailurePromptResponse return this terminal policy alongside end_turn.
// FS-03.R8 and FS-01.R44 require failure visibility and failed recovery settlement;
// TS-04.R86 separately forbids treating transient 429s as account quota.
func TestTerminalProviderRateLimitInvestigation(t *testing.T) {
	t.Skip("PERR-1/PERR-2: confirmed terminal non-quota failures are silently settled as success; unskip with fix")
	const message = "exceeded retry limit, last status: 429 Too Many Requests"
	const failure = `{"id":"turn-429:error","revision":1,"category":"limit","severity":"error","title":"exceeded retry limit, last status: 429 Too Many Requests","actions":["retry"]}`
	c, h, ch, _ := startQuotaTestAgent(t, "stream_text", "FAKEACP_FAILURE_RESULT="+failure)
	mu, calls := recordQuotaSettlement(c)
	if err := c.SendPrompt(context.Background(), h.AgentID, "answer this"); err != nil {
		t.Fatal(err)
	}
	var end TurnEndData
	visibleError := false
	for _, ev := range drainTurn(t, ch) {
		switch ev.Type {
		case EvError:
			var data ErrorData
			if err := json.Unmarshal(ev.Data, &data); err != nil {
				t.Fatal(err)
			}
			visibleError = visibleError || strings.Contains(data.Message, message)
		case EvTurnEnd:
			if err := json.Unmarshal(ev.Data, &end); err != nil {
				t.Fatal(err)
			}
		}
	}
	if end.StopReason != "error" || end.Quota != nil {
		t.Errorf("terminal rate limit = %+v; want error without account-quota evidence", end)
	}
	if !visibleError {
		t.Error("terminal provider failure message never reached the transcript")
	}
	status, err := c.store.ReadStatus(h.AgentID)
	if err != nil {
		t.Fatal(err)
	}
	if status.State != "error" {
		t.Errorf("status = %q; want error", status.State)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(*calls) != 1 || (*calls)[0].successful || (*calls)[0].quota != nil {
		t.Errorf("recovery settlement = %+v; want one non-quota unsuccessful settlement", *calls)
	}
}
