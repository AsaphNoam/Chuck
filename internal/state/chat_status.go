package state

type ChatProcessState uint8

const (
	ChatProcessLive ChatProcessState = iota
	ChatProcessStopped
	ChatProcessFailed
)

// ComposeChatStatus derives the visible state for the current generation.
// Callers pass a zero turn when the stored row belongs to an older generation.
func ComposeChatStatus(process ChatProcessState, turn ConversationTurn, permissionPending bool) string {
	switch process {
	case ChatProcessStopped:
		return "done"
	case ChatProcessFailed:
		return "error"
	}
	if permissionPending {
		return "waiting_input"
	}
	if turn.State == "running" {
		return "busy"
	}
	switch turn.Outcome {
	case "failed", "quota_blocked", "process_ended", "unknown", "refused":
		return "error"
	default:
		return "idle"
	}
}
