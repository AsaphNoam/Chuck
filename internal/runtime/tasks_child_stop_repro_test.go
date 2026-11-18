package runtime

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"testing"
)

// TASK-STOP-CHILD reproduction. The pinned claude-agent-acp 0.85.1 keeps one
// async-task registry per root session: stopAsyncTask resolves
// this.sessions[params.sessionId], and only root sessions are registered there.
// A subagent's task is announced routed to the child session id, which
// onAsyncTaskUpdate stores as the task's sessionID (tasks.go), so Stop on a
// running child task reaches the adapter with the child id and is answered
// {stopped:false} -> 409 -> "Could not stop this task. Try again."
func TestStopRunningChildBackgroundTaskAddressesTheRootSession(t *testing.T) {
	t.Skip("TASK-STOP-CHILD: reproduction; remove the skip with the fix")
	const root, child = "sess_root", "sess_child"
	c := NewChatRuntime(nil)
	toPeer, fromChuck := io.Pipe()
	fromPeer, toChuck := io.Pipe()
	defer fromChuck.Close()
	defer toChuck.Close()
	as := &agentState{agentID: "a", sessionID: root}
	as.caps.BackgroundTaskStop = true
	as.transport = NewTransport(fromChuck, nil, nil)
	go func() { _ = as.transport.Run(fromPeer) }()
	scope := activityScope{ActivityID: activityIDFor(child)}
	taskID := scope.toolCallID("task_1")
	// What onAsyncTaskUpdate records for a child-routed async_task_spawned.
	as.tasks = map[string]*taskRef{taskID: {sessionID: child, rawID: "task_1", scope: scope, running: true}}
	as.openTasks = 1
	c.mu.Lock()
	c.agents = map[string]*agentState{"a": as}
	c.mu.Unlock()

	// Peer modelled on claude-agent-acp 0.85.1 stopAsyncTask.
	go func() {
		sc := bufio.NewScanner(toPeer)
		for sc.Scan() {
			var req struct {
				ID     int64 `json:"id"`
				Params struct {
					SessionID string `json:"sessionId"`
				} `json:"params"`
			}
			if json.Unmarshal(sc.Bytes(), &req) != nil {
				continue
			}
			fmt.Fprintf(toChuck, `{"jsonrpc":"2.0","id":%d,"result":{"stopped":%t}}`+"\n", req.ID, req.Params.SessionID == root)
		}
	}()

	if err := c.StopBackgroundTask(context.Background(), "a", taskID); err != nil {
		t.Fatalf("stop running child task = %v; want accepted (adapter only knows root session %q)", err, root)
	}
}
