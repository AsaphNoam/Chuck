package runtime

import (
	"io"
	"sync"
	"testing"
	"time"
)

func TestCancelReleasesTurnLockBeforeWritingToPeer(t *testing.T) {
	c := NewChatRuntime(nil)
	c.SetCancelGrace(0)
	pr, pw := io.Pipe() // a child that is not reading stdin
	defer pr.Close()
	as := &agentState{agentID: "a", turn: currentTurn{turnActive: true, turnSeq: 1, pending: map[string]*pendingPerm{}}}
	writeEntered := make(chan struct{})
	as.transport = NewTransport(&signalWriter{Writer: pw, entered: writeEntered}, nil, nil)
	cancelDone := make(chan struct{})
	go func() {
		defer close(cancelDone)
		_, _ = c.cancel(as, "", "")
	}()
	select {
	case <-writeEntered:
	case <-time.After(time.Second):
		t.Fatal("session/cancel never reached the peer")
	}
	got := make(chan struct{})
	go func() { as.mu.Lock(); as.mu.Unlock(); close(got) }()
	select {
	case <-got:
	case <-time.After(time.Second):
		t.Fatal("blocked peer held the turn lock against Stop")
	}
	_ = pr.Close()
	select {
	case <-cancelDone:
	case <-time.After(time.Second):
		t.Fatal("Cancel did not return after closing the peer")
	}
}

type signalWriter struct {
	io.Writer
	entered chan struct{}
	once    sync.Once
}

func (w *signalWriter) Write(p []byte) (int, error) {
	w.once.Do(func() { close(w.entered) })
	return w.Writer.Write(p)
}
