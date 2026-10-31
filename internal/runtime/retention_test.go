package runtime

import (
	"reflect"
	"strings"
	"testing"
)

// MEM-4 / INV §16: committed output must still reach the journal and live
// subscribers, but agentState must not hold a second full-history event slice.
// Inspect retained event slices because delivery alone cannot expose this leak.
func TestCommittedOutputDoesNotRetainFullHistory(t *testing.T) {
	w := &recordingWriter{}
	c, as, published := newChatJournalState(t, w)
	ch, cancel := as.hub.Subscribe()
	defer cancel()

	const count = 100
	for i := 0; i < count; i++ {
		c.emit(as, EvAssistantText, AssistantTextData{Delta: strings.Repeat("x", 8192)})
		select {
		case ev := <-ch:
			if ev.Type != EvAssistantText || len(ev.Data) < 8192 {
				t.Fatalf("live output lost: type=%s bytes=%d", ev.Type, len(ev.Data))
			}
		default:
			t.Fatal("committed output was not published live")
		}
	}
	c.emit(as, EvNotice, NoticeData{Title: "keep notice deduplication"})
	<-ch
	if len(w.seqs) != count+1 || len(*published) != count+1 {
		t.Fatalf("committed output lost: durable=%d published=%d", len(w.seqs), len(*published))
	}

	as.mu.Lock()
	defer as.mu.Unlock()
	state := reflect.ValueOf(as).Elem()
	for i := 0; i < state.NumField(); i++ {
		field := state.Field(i)
		if field.Type() != reflect.TypeOf([]Event{}) {
			continue
		}
		for j := 0; j < field.Len(); j++ {
			if kind := field.Index(j).FieldByName("Type").String(); kind != string(EvNotice) {
				t.Fatalf("agentState.%s retains %s payload after commit", state.Type().Field(i).Name, kind)
			}
		}
	}
}

// Removing full history must preserve both retry identities that consume notices.
func TestNoticeRetriesReuseCommittedEvent(t *testing.T) {
	for _, kind := range []string{"quota", "peer-stop"} {
		t.Run(kind, func(t *testing.T) {
			w := &recordingWriter{}
			c, as, _ := newChatJournalState(t, w)
			c.agents[as.agentID] = as
			appendNotice := func() (Event, error) {
				if kind == "quota" {
					return c.AppendQuotaNoticeAndSync(as.agentID, NoticeData{ContinuationAttempt: "attempt-1"})
				}
				return c.AppendPeerStopNoticeAndSync(as.agentID, NoticeData{PeerStop: &PeerStopNotice{
					RequestID: "request-1", RequesterID: "requester", TargetID: "target", Phase: "approved",
				}})
			}
			first, err := appendNotice()
			if err != nil {
				t.Fatal(err)
			}
			c.emit(as, EvAssistantText, AssistantTextData{Delta: "intervening output"})
			retry, err := appendNotice()
			if err != nil {
				t.Fatal(err)
			}
			if retry.Seq != first.Seq || len(w.seqs) != 2 {
				t.Fatalf("retry duplicated notice: first=%d retry=%d appends=%v", first.Seq, retry.Seq, w.seqs)
			}
		})
	}
}
