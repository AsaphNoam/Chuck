package state

import "testing"

func TestComposeChatStatusWriterInventory(t *testing.T) {
	running := ConversationTurn{State: "running"}
	failed := ConversationTurn{State: "ended", Outcome: "failed"}
	for _, tt := range []struct {
		site       string
		process    ChatProcessState
		turn       ConversationTurn
		permission bool
		want       string
	}{
		{"launch", ChatProcessLive, ConversationTurn{}, false, "idle"},
		{"resume ignores old generation", ChatProcessLive, ConversationTurn{}, false, "idle"},
		{"person prompt", ChatProcessLive, running, false, "busy"},
		{"activation", ChatProcessLive, running, false, "busy"},
		{"stop", ChatProcessStopped, running, true, "done"},
		{"owned tool activity", ChatProcessLive, running, false, "busy"},
		{"completion", ChatProcessLive, ConversationTurn{State: "ended", Outcome: "completed"}, false, "idle"},
		{"cancel", ChatProcessLive, ConversationTurn{State: "ended", Outcome: "cancelled"}, false, "idle"},
		{"failure", ChatProcessLive, failed, false, "error"},
		{"provider permission", ChatProcessLive, running, true, "waiting_input"},
		{"host permission", ChatProcessLive, running, true, "waiting_input"},
		{"transport crash", ChatProcessFailed, running, true, "error"},
		{"reconciled exit", ChatProcessStopped, running, false, "done"},
		{"failed switch", ChatProcessFailed, ConversationTurn{}, false, "error"},
	} {
		t.Run(tt.site, func(t *testing.T) {
			if got := ComposeChatStatus(tt.process, tt.turn, tt.permission); got != tt.want {
				t.Errorf("state = %q, want %q", got, tt.want)
			}
		})
	}
}
