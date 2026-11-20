package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func runEditCmd(t *testing.T, handler http.HandlerFunc, stdin string, args ...string) (string, string, error) {
	t.Helper()
	server := httptest.NewServer(handler)
	defer server.Close()
	old := operatorClientFactory
	operatorClientFactory = func() (*operatorClient, error) {
		return &operatorClient{HTTP: server.Client(), BaseURL: server.URL}, nil
	}
	defer func() { operatorClientFactory = old }()
	cmd := newOperatorAgentCmd()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetIn(strings.NewReader(stdin))
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), errOut.String(), err
}

func TestOperatorEditLastBodyAndGeneratedRequestID(t *testing.T) {
	var got, path string
	out, errOut, err := runEditCmd(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		got, path = string(b), r.Method+" "+r.URL.Path
		w.WriteHeader(202)
		_, _ = w.Write([]byte(`{"agent_id":"a","request_id":"x","phase":"preparing"}`))
	}, "line1\nünï \n\n", "edit-last", "a", "--message-seq", "7", "--expected-revision", "0", "--text-file", "-", "--json")
	if err != nil {
		t.Fatal(err)
	}
	if path != "POST /api/sessions/a/edit-last" {
		t.Fatalf("path %s", path)
	}
	m := map[string]any{}
	if err := json.Unmarshal([]byte(got), &m); err != nil {
		t.Fatal(err)
	}
	id, _ := m["request_id"].(string)
	if m["text"] != "line1\nünï \n\n" || m["message_seq"] != float64(7) || m["expected_revision"] != float64(0) || !operatorValidReceiptID(id) || len(m) != 4 {
		t.Fatalf("body %s", got)
	}
	if !strings.Contains(errOut, id) || !strings.Contains(out, `"phase":"preparing"`) {
		t.Fatalf("out %q err %q", out, errOut)
	}
}

func TestOperatorEditLastRequestFilePassthroughAndRetryID(t *testing.T) {
	var got string
	body := `{"request_id":"r-1","message_seq":3,"expected_revision":2,"text":"a\n"}`
	_, errOut, err := runEditCmd(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		got = string(b)
		w.WriteHeader(202)
		_, _ = w.Write([]byte(`{}`))
	}, body, "edit-last", "a", "--request-file", "-", "--json")
	if err != nil {
		t.Fatal(err)
	}
	if got != `{"expected_revision":2,"message_seq":3,"request_id":"r-1","text":"a\n"}` || errOut != "" {
		t.Fatalf("body %s err %q", got, errOut)
	}
}

func TestOperatorEditLastRejectsBeforeRequest(t *testing.T) {
	for _, args := range [][]string{
		{"edit-last", "a", "--expected-revision", "0", "--text", "x"},
		{"edit-last", "a", "--message-seq", "1", "--text", "x"},
		{"edit-last", "a", "--message-seq", "1", "--expected-revision", "0"},
		{"edit-last", "a", "--message-seq", "0", "--expected-revision", "0", "--text", "x"},
		{"edit-last", "a", "--message-seq", "1", "--expected-revision", "0", "--text", "x", "--request-id", "bad id"},
		{"edit-last", "a", "--message-seq", "1", "--expected-revision", "0", "--text", "x", "--text-file", "-"},
	} {
		_, _, err := runEditCmd(t, func(w http.ResponseWriter, r *http.Request) { t.Errorf("request sent for %v", args) }, "", args...)
		var op *operatorError
		if !errors.As(err, &op) || op.ExitCode != 2 {
			t.Errorf("%v: %v", args, err)
		}
	}
}

func TestOperatorEditLastErrorMapping(t *testing.T) {
	for _, tc := range []struct {
		status int
		code   string
	}{{409, "edit_conflict"}, {422, "edit_unavailable"}, {409, "request_conflict"}} {
		_, _, err := runEditCmd(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(tc.status)
			_, _ = fmt.Fprintf(w, `{"code":%q,"message":"nope","details":{"reason":"busy"}}`, tc.code)
		}, "", "edit-last", "a", "--message-seq", "1", "--expected-revision", "0", "--text", "x", "--request-id", "r")
		var op *operatorError
		if !errors.As(err, &op) || op.Code != tc.code || op.HTTPStatus != tc.status || op.ExitCode != 1 {
			t.Errorf("%s: %#v", tc.code, err)
		}
	}
}

func TestOperatorEditLastStatusQuery(t *testing.T) {
	var path string
	out, _, err := runEditCmd(t, func(w http.ResponseWriter, r *http.Request) {
		path = r.Method + " " + r.URL.Path
		_, _ = w.Write([]byte(`{"agent_id":"a","request_id":"r-1","phase":"committed","revision":1,"delivery":"sent","active":"revised"}`))
	}, "", "edit-last-status", "a", "r-1", "--json")
	if err != nil || path != "GET /api/sessions/a/edit-last/r-1" || !strings.Contains(out, `"phase":"committed"`) {
		t.Fatalf("%v %s %s", err, path, out)
	}
}

// revisionFollowSeqs follows agent a and returns the emitted sequences in
// order. pages[i] answers stream i's since_seq read, live[i] is stream i's tail
// after hydration and full is the unbounded active view a reset re-reads.
func revisionFollowSeqs(t *testing.T, pages, live []string, full string, target int) []int64 {
	t.Helper()
	marker := map[string]any{"agent_id": "__hydrated__", "data": map[string]bool{"hydrated": true}}
	var streams atomic.Int32
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/events":
			n := int(streams.Add(1)) - 1
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = io.WriteString(w, operatorSSEEvent("state_update", marker))
			if n < len(live) && live[n] != "" {
				outer := map[string]any{"type": "new_message", "agent_id": "a", "data": json.RawMessage(live[n])}
				_, _ = io.WriteString(w, operatorSSEEvent("new_message", outer))
			}
		case "/api/sessions/a/transcript":
			if !r.URL.Query().Has("since_seq") {
				_, _ = io.WriteString(w, `{"agent_id":"a","events":[`+full+`]}`)
				return
			}
			n := int(streams.Load()) - 1
			page := ""
			if n < len(pages) {
				page = pages[n]
			}
			_, _ = io.WriteString(w, `{"agent_id":"a","events":[`+page+`]}`)
		default:
			http.NotFound(w, r)
		}
	})
	srv := httptest.NewServer(h)
	defer srv.Close()
	client := &operatorClient{HTTP: srv.Client(), BaseURL: srv.URL}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	out := &countOutput{cancel: cancel, target: target}
	cmd := operatorFollowCmd(t, ctx, out)
	cmd.SetErr(io.Discard)
	if err := operatorFollow(cmd, client, "a", 0); err != nil {
		t.Fatalf("follow: %v output=%q", err, out.buf.String())
	}
	var seqs []int64
	for _, line := range strings.Split(strings.TrimSpace(out.buf.String()), "\n") {
		var e struct {
			Seq int64 `json:"seq"`
		}
		if json.Unmarshal([]byte(line), &e) != nil {
			t.Fatalf("line %q", line)
		}
		seqs = append(seqs, e.Seq)
	}
	return seqs
}

const (
	revUser1 = `{"agent_id":"a","seq":1,"type":"user_message","data":{"text":"old"}}`
	revText2 = `{"agent_id":"a","seq":2,"type":"assistant_text","data":{"delta":"reply"}}`
	revAnn3  = `{"agent_id":"a","seq":3,"type":"annotation","data":{"text":"kept"}}`
	revText4 = `{"agent_id":"a","seq":4,"type":"assistant_text","data":{"delta":"more"}}`
	revRev5  = `{"agent_id":"a","seq":5,"type":"conversation_revision","data":{"request_id":"r","revision":1,"replaces_from_seq":1,"replaces_through_seq":4}}`
	revUser6 = `{"agent_id":"a","seq":6,"type":"user_message","data":{"text":"new"}}`
)

// The active view after the edit: the annotation stays inside the superseded
// range and the revision follows a gap.
const revActive = revAnn3 + "," + revRev5 + "," + revUser6

func TestOperatorFollowInitialFoldedReadCrossesRetainedRecord(t *testing.T) {
	got := revisionFollowSeqs(t, []string{revActive}, nil, revActive, 3)
	if fmt.Sprint(got) != "[3 5 6]" {
		t.Fatalf("seqs = %v", got)
	}
}

func TestOperatorFollowReconnectResetsBeforeRefreshedView(t *testing.T) {
	// Stream 1 emits 1–2 then ends; the reconnect page is folded past the
	// cursor, so the retained annotation sits before the revision.
	got := revisionFollowSeqs(t, []string{revUser1 + "," + revText2, revActive}, nil, revActive, 5)
	if fmt.Sprint(got) != "[1 2 5 3 6]" {
		t.Fatalf("seqs = %v; want reset boundary 5 before the refreshed view", got)
	}
}

func TestOperatorFollowLiveEditResetsBeforeRefreshedView(t *testing.T) {
	initial := revUser1 + "," + revText2 + "," + revAnn3 + "," + revText4
	got := revisionFollowSeqs(t, []string{initial}, []string{revRev5}, revActive, 7)
	if fmt.Sprint(got) != "[1 2 3 4 5 3 6]" {
		t.Fatalf("seqs = %v; want reset boundary 5 before the refreshed view", got)
	}
}

func TestOperatorContinuousSkipsOnlySupersededSpans(t *testing.T) {
	spans := [][2]int64{{3, 5}, {6, 8}}
	for _, c := range []struct {
		last, seq int64
		want      bool
	}{{2, 3, true}, {2, 9, true}, {2, 10, false}, {1, 4, false}} {
		if got := operatorContinuous(c.last, c.seq, spans); got != c.want {
			t.Errorf("continuous(%d,%d)=%v", c.last, c.seq, got)
		}
	}
}
