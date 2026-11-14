package state

import (
	"testing"
	"time"
)

type touchOrderPublisher func(AgentStateUpdate)

func (p touchOrderPublisher) PublishStateUpdate(update AgentStateUpdate) { p(update) }

func TestTouchPublishesBeforeLaterHookMutation(t *testing.T) {
	st, _ := newTestStore(t)
	agent := testAgent("a_touch_order", time.Now())
	if err := st.WriteAgent(agent); err != nil {
		t.Fatal(err)
	}
	if err := st.WriteRunning(RunningEntry{AgentID: agent.AgentID, PID: 1234, HookToken: "token", Interface: "chat", StartedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	if err := st.WriteStatus(Status{AgentID: agent.AgentID, State: "idle"}); err != nil {
		t.Fatal(err)
	}
	entered := make(chan struct{})
	release := make(chan struct{})
	published := make(chan string, 2)
	mgr := NewManager(st, touchOrderPublisher(func(update AgentStateUpdate) {
		if update.State == "idle" {
			close(entered)
			<-release
		}
		published <- update.State
	}))
	touched := make(chan error, 1)
	go func() {
		_, err := mgr.Touch(agent.AgentID)
		touched <- err
	}()
	select {
	case <-entered:
	case err := <-touched:
		close(release)
		t.Fatalf("Touch returned before publishing idle: %v", err)
	case <-time.After(time.Second):
		close(release)
		t.Fatal("Touch did not reach publisher")
	}

	applied := make(chan error, 1)
	go func() {
		_, err := mgr.ApplyHook("token", HookPayload{AgentID: agent.AgentID, Event: "status", State: "busy"})
		applied <- err
	}()
	select {
	case err := <-applied:
		close(release)
		<-touched
		t.Fatalf("hook overtook blocked Touch publication: %v", err)
	case <-time.After(100 * time.Millisecond):
		close(release)
	}
	for _, result := range []<-chan error{touched, applied} {
		select {
		case err := <-result:
			if err != nil {
				t.Fatal(err)
			}
		case <-time.After(time.Second):
			t.Fatal("state publication did not complete")
		}
	}
	if len(published) != 2 {
		t.Fatalf("published %d updates, want two", len(published))
	}
	if first, second := <-published, <-published; first != "idle" || second != "busy" {
		t.Fatalf("published %q then %q, want idle then busy", first, second)
	}
}
