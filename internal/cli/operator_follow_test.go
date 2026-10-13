package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

type cancelOutput struct {
	buf    bytes.Buffer
	cancel context.CancelFunc
}

type countOutput struct {
	buf    bytes.Buffer
	cancel context.CancelFunc
	writes int
	target int
}

func (w *countOutput) Write(p []byte) (int, error) {
	w.writes++
	n, err := w.buf.Write(p)
	target := w.target
	if target == 0 {
		target = 3
	}
	if w.writes >= target {
		w.cancel()
	}
	return n, err
}

type epipeOutput struct{}

func (epipeOutput) Write([]byte) (int, error) { return 0, syscall.EPIPE }

func (w *cancelOutput) Write(p []byte) (int, error) {
	n, err := w.buf.Write(p)
	w.cancel()
	return n, err
}

type operatorTestServer struct {
	URL    string
	server *http.Server
	client *http.Client
}

func (s *operatorTestServer) Client() *http.Client { return s.client }
func (s *operatorTestServer) Close()               { _ = s.server.Close() }

func operatorFollowServer(t *testing.T, transcriptBody string, stream func(http.ResponseWriter)) (*operatorTestServer, *atomic.Int32) {
	t.Helper()
	var streams atomic.Int32
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/events":
			streams.Add(1)
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			stream(w)
		case "/api/sessions/a/transcript":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, transcriptBody)
		default:
			http.NotFound(w, r)
		}
	})
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("loopback unavailable: %v", err)
	}
	s := &http.Server{Handler: h}
	go func() { _ = s.Serve(ln) }()
	wrapped := &operatorTestServer{URL: "http://" + ln.Addr().String(), server: s, client: &http.Client{}}
	t.Cleanup(wrapped.Close)
	return wrapped, &streams
}

func operatorSSEEvent(event string, data any) string {
	b, _ := json.Marshal(data)
	return "event: " + event + "\ndata: " + string(b) + "\n\n"
}

func operatorFollowCmd(t *testing.T, ctx context.Context, out io.Writer) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{}
	cmd.SetContext(ctx)
	cmd.SetOut(out)
	cmd.Flags().Bool("json", true, "")
	return cmd
}

func TestOperatorReadSSEParsesMultipleDataLines(t *testing.T) {
	out := make(chan operatorSSEFrame, 2)
	operatorReadSSE(context.Background(), strings.NewReader("event: new_message\ndata: {\"a\":\ndata: 1}\n\n"), out)
	frame := <-out
	if frame.event != "new_message" || string(frame.data) != "{\"a\":\n1}" {
		t.Fatalf("frame = %#v", frame)
	}
}

func TestOperatorFollowEventRejectsGapAndSkipsDuplicate(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("json", true, "")
	cmd.SetOut(io.Discard)
	last := int64(2)
	if err := operatorFollowEvent(cmd, json.RawMessage(`{"seq":2,"type":"unknown","data":{}}`), &last); err != nil {
		t.Fatal(err)
	}
	if last != 2 {
		t.Fatalf("duplicate advanced sequence: %d", last)
	}
	if err := operatorFollowEvent(cmd, json.RawMessage(`{"seq":4,"type":"unknown","data":{}}`), &last); err == nil {
		t.Fatal("gap accepted")
	}
}

func TestOperatorHydratedMarkerRequiresExactAgent(t *testing.T) {
	if operatorHydrated([]byte(`{"agent_id":"other","data":{"hydrated":true}}`)) {
		t.Fatal("wrong marker accepted")
	}
	if !operatorHydrated([]byte(`{"agent_id":"__hydrated__","data":{"hydrated":true}}`)) {
		t.Fatal("marker rejected")
	}
}

func TestOperatorReadSSEBoundsQueuedTail(t *testing.T) {
	out := make(chan operatorSSEFrame, 2)
	input := "event: ping\ndata: {}\n\n" + "event: ping\ndata: {}\n\n" + "event: ping\ndata: {}\n\n"
	operatorReadSSE(context.Background(), strings.NewReader(input), out)
	if len(out) != cap(out) {
		t.Fatalf("queued frames = %d, want bounded %d", len(out), cap(out))
	}
}

func TestOperatorReadSSECancellationStopsBeforePublishing(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	out := make(chan operatorSSEFrame, 2)
	operatorReadSSE(ctx, strings.NewReader("event: ping\ndata: {}\n\n"), out)
	if len(out) != 0 {
		t.Fatalf("canceled stream published %d frames", len(out))
	}
}

func TestOperatorFollowE2ESubscribeBeforeReadDeduplicatesCatchup(t *testing.T) {
	event := map[string]any{"agent_id": "a", "seq": 1, "type": "assistant_text", "data": map[string]string{"delta": "hello"}}
	outer := map[string]any{"type": "new_message", "seq": 2, "agent_id": "a", "data": event}
	marker := map[string]any{"agent_id": "__hydrated__", "data": map[string]bool{"hydrated": true}}
	server, _ := operatorFollowServer(t, `{"agent_id":"a","events":[{"agent_id":"a","seq":1,"type":"assistant_text","data":{"delta":"hello"}}]}`, func(w http.ResponseWriter) {
		_, _ = io.WriteString(w, operatorSSEEvent("state_update", marker)+operatorSSEEvent("new_message", outer))
	})
	client := &operatorClient{HTTP: server.Client(), BaseURL: server.URL}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	out := &cancelOutput{cancel: cancel}
	cmd := operatorFollowCmd(t, ctx, out)
	if err := operatorFollow(cmd, client, "a", 0); err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(out.buf.String(), "hello"); got != 1 {
		t.Fatalf("follow output count = %d, output=%q", got, out.buf.String())
	}
}

func TestOperatorFollowE2EOpensSubscriptionBeforeTranscriptRead(t *testing.T) {
	marker := map[string]any{"agent_id": "__hydrated__", "data": map[string]bool{"hydrated": true}}
	started := make(chan struct{})
	var readBefore atomic.Bool
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/events" {
			close(started)
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = io.WriteString(w, operatorSSEEvent("state_update", marker))
			return
		}
		if r.URL.Path == "/api/sessions/a/transcript" {
			select {
			case <-started:
			default:
				readBefore.Store(true)
			}
			_, _ = io.WriteString(w, `{"agent_id":"a","events":[{"seq":1,"type":"unknown_future","data":{}}]}`)
			return
		}
		http.NotFound(w, r)
	})
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: h}
	go srv.Serve(ln)
	defer srv.Close()
	client := &operatorClient{HTTP: &http.Client{}, BaseURL: "http://" + ln.Addr().String()}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	out := &cancelOutput{cancel: cancel}
	cmd := operatorFollowCmd(t, ctx, out)
	if err := operatorFollow(cmd, client, "a", 0); err != nil {
		t.Fatal(err)
	}
	if readBefore.Load() {
		t.Fatal("transcript read preceded event subscription")
	}
}

func TestOperatorFollowE2EGapRefillsFromAuthoritativeTranscript(t *testing.T) {
	marker := map[string]any{"agent_id": "__hydrated__", "data": map[string]bool{"hydrated": true}}
	events := []string{`{"agent_id":"a","seq":1,"type":"unknown_future","data":{"value":"one"}}`, `{"agent_id":"a","seq":2,"type":"unknown_future","data":{"value":"two"}}`, `{"agent_id":"a","seq":3,"type":"unknown_future","data":{"value":"three"}}`}
	var streams atomic.Int32
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/events" {
			n := streams.Add(1)
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(200)
			_, _ = io.WriteString(w, operatorSSEEvent("state_update", marker))
			if n == 1 {
				outer := map[string]any{"type": "new_message", "seq": 9, "agent_id": "a", "data": json.RawMessage(events[2])}
				_, _ = io.WriteString(w, operatorSSEEvent("new_message", outer))
			}
			return
		}
		if r.URL.Path == "/api/sessions/a/transcript" {
			since, _ := strconv.Atoi(r.URL.Query().Get("since_seq"))
			var chosen []string
			end := len(events)
			if since == 0 {
				end = 1
			}
			for i := since; i < end; i++ {
				chosen = append(chosen, events[i])
			}
			_, _ = fmt.Fprintf(w, `{"agent_id":"a","events":[%s]}`, strings.Join(chosen, ","))
			return
		}
		http.NotFound(w, r)
	})
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: h}
	go srv.Serve(ln)
	defer srv.Close()
	server := &operatorTestServer{URL: "http://" + ln.Addr().String(), server: srv, client: &http.Client{}}
	client := &operatorClient{HTTP: server.Client(), BaseURL: server.URL}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	out := &countOutput{cancel: cancel}
	cmd := operatorFollowCmd(t, ctx, out)
	if err := operatorFollow(cmd, client, "a", 0); err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(out.buf.String(), "seq"); got != 3 {
		t.Fatalf("gap refill lines = %d, output=%q", got, out.buf.String())
	}
}

func TestOperatorFollowPingRecoversDroppedFinalTail(t *testing.T) {
	read := make(chan struct{})
	var reads atomic.Int32
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/events":
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = io.WriteString(w, operatorSSEEvent("state_update", map[string]any{"agent_id": "__hydrated__", "data": map[string]bool{"hydrated": true}}))
			w.(http.Flusher).Flush()
			select {
			case <-read:
			case <-r.Context().Done():
				return
			}
			_, _ = io.WriteString(w, "event: ping\ndata: {}\n\n")
			w.(http.Flusher).Flush()
			<-r.Context().Done()
		case "/api/sessions/a/transcript":
			if reads.Add(1) == 1 {
				_, _ = io.WriteString(w, `{"agent_id":"a","events":[{"seq":1,"type":"unknown_future","data":{"value":"first"}}]}`)
				close(read)
			} else {
				if r.URL.Query().Get("since_seq") != "1" {
					t.Errorf("catchup cursor: %s", r.URL.RawQuery)
				}
				_, _ = io.WriteString(w, `{"agent_id":"a","events":[{"seq":2,"type":"unknown_future","data":{"value":"dropped-final"}}]}`)
			}
		default:
			http.NotFound(w, r)
		}
	})
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: h}
	go srv.Serve(ln)
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out := &countOutput{cancel: cancel, target: 2}
	cmd := operatorFollowCmd(t, ctx, out)
	client := &operatorClient{HTTP: &http.Client{}, BaseURL: "http://" + ln.Addr().String()}
	if err := operatorFollow(cmd, client, "a", 0); err != nil {
		t.Fatal(err)
	}
	if reads.Load() != 2 || strings.Count(out.buf.String(), "dropped-final") != 1 || out.writes != 2 {
		t.Fatalf("reads=%d output=%q", reads.Load(), out.buf.String())
	}
}

func TestOperatorFollowE2ERejectsMissingEvents(t *testing.T) {
	marker := map[string]any{"agent_id": "__hydrated__", "data": map[string]bool{"hydrated": true}}
	server, _ := operatorFollowServer(t, `{"agent_id":"a"}`, func(w http.ResponseWriter) { _, _ = io.WriteString(w, operatorSSEEvent("state_update", marker)) })
	client := &operatorClient{HTTP: server.Client(), BaseURL: server.URL}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd := operatorFollowCmd(t, ctx, io.Discard)
	err := operatorFollow(cmd, client, "a", 0)
	if err == nil || !strings.Contains(err.Error(), "missing events") {
		t.Fatalf("error = %v", err)
	}
}

func TestOperatorFollowE2EReconnectsAndResumesCursor(t *testing.T) {
	marker := map[string]any{"agent_id": "__hydrated__", "data": map[string]bool{"hydrated": true}}
	var streams atomic.Int32
	errOut := new(bytes.Buffer)
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/events" {
			n := streams.Add(1)
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = io.WriteString(w, operatorSSEEvent("state_update", marker))
			if n == 1 {
				_, _ = io.WriteString(w, operatorSSEEvent("new_message", map[string]any{"type": "new_message", "agent_id": "a", "data": map[string]any{"agent_id": "a", "seq": 1, "type": "unknown_future", "data": map[string]string{"value": "one"}}}))
			}
			return
		}
		if r.URL.Path == "/api/sessions/a/transcript" {
			if streams.Load() == 1 {
				_, _ = io.WriteString(w, `{"agent_id":"a","events":[{"agent_id":"a","seq":1,"type":"unknown_future","data":{"value":"one"}}]}`)
			} else {
				if r.URL.Query().Get("since_seq") != "1" {
					t.Errorf("reconnect cursor %s", r.URL)
				}
				_, _ = io.WriteString(w, `{"agent_id":"a","events":[{"agent_id":"a","seq":2,"type":"unknown_future","data":{"value":"two"}}]}`)
			}
			return
		}
		http.NotFound(w, r)
	})
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: h}
	go srv.Serve(ln)
	defer srv.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	out := &countOutput{cancel: cancel, target: 2}
	cmd := operatorFollowCmd(t, ctx, out)
	cmd.SetErr(errOut)
	client := &operatorClient{HTTP: &http.Client{}, BaseURL: "http://" + ln.Addr().String()}
	if err := operatorFollow(cmd, client, "a", 0); err != nil {
		t.Fatal(err)
	}
	if streams.Load() < 2 {
		t.Fatalf("streams=%d", streams.Load())
	}
	if !strings.Contains(out.buf.String(), "two") {
		t.Fatalf("output=%q", out.buf.String())
	}
	if errOut.Len() == 0 {
		t.Fatal("missing reconnect progress")
	}
}

func TestOperatorFollowE2EEpipeStopsCleanly(t *testing.T) {
	marker := map[string]any{"agent_id": "__hydrated__", "data": map[string]bool{"hydrated": true}}
	srv, _ := operatorFollowServer(t, `{"agent_id":"a","events":[{"seq":1,"type":"unknown_future","data":{}}]}`, func(w http.ResponseWriter) { _, _ = io.WriteString(w, operatorSSEEvent("state_update", marker)) })
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := operatorFollowCmd(t, ctx, epipeOutput{})
	if err := operatorFollow(cmd, &operatorClient{HTTP: srv.Client(), BaseURL: srv.URL}, "a", 0); err != nil {
		t.Fatal(err)
	}
}

func TestOperatorSubscribeOverflowClosesBoundedObserver(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		for i := 0; i < 300; i++ {
			_, _ = io.WriteString(w, "event: ping\ndata: {}\n\n")
		}
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	})
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: h}
	go srv.Serve(ln)
	defer srv.Close()
	client := &operatorClient{HTTP: &http.Client{}, BaseURL: "http://" + ln.Addr().String()}
	o, err := operatorSubscribe(context.Background(), client, "a", time.Now().Add(2*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-o.done:
		var op *operatorError
		if !errors.As(got, &op) || op.Code != "stream_overflow" {
			t.Fatalf("err=%v", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("observer did not overflow")
	}
	o.close()
}

func TestOperatorParseSSERejectsOversizedFrame(t *testing.T) {
	data := strings.Repeat("x", (16<<20)+1)
	err := operatorParseSSE(context.Background(), strings.NewReader("event: ping\ndata: \""+data+"\"\n\n"), func(operatorSSEFrame) error { return nil })
	if err == nil {
		t.Fatal("oversized frame accepted")
	}
}

func TestOperatorFollowE2EPreservesLargeSSEPayload(t *testing.T) {
	marker := map[string]any{"agent_id": "__hydrated__", "data": map[string]bool{"hydrated": true}}
	large := strings.Repeat("x", 100<<10)
	outer := map[string]any{"type": "new_message", "seq": 2, "agent_id": "a", "data": map[string]any{"agent_id": "a", "seq": 1, "type": "unknown_future", "data": map[string]string{"value": large}}}
	server, _ := operatorFollowServer(t, `{"agent_id":"a","events":[{"seq":0,"type":"session_meta","data":{}}]}`, func(w http.ResponseWriter) {
		_, _ = io.WriteString(w, operatorSSEEvent("state_update", marker)+operatorSSEEvent("new_message", outer))
	})
	client := &operatorClient{HTTP: server.Client(), BaseURL: server.URL}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	out := &cancelOutput{cancel: cancel}
	cmd := operatorFollowCmd(t, ctx, out)
	if err := operatorFollow(cmd, client, "a", 0); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.buf.String(), large) {
		t.Fatalf("large SSE payload was truncated: %d bytes", out.buf.Len())
	}
}

func TestOperatorFollowHealthyPingsOutliveOrdinaryDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 36*time.Second)
	defer cancel()
	marker := map[string]any{"agent_id": "__hydrated__", "data": map[string]bool{"hydrated": true}}
	srv, _ := operatorFollowServer(t, `{"agent_id":"a","events":[{"seq":0,"type":"session_meta","data":{}}]}`, func(w http.ResponseWriter) {
		_, _ = io.WriteString(w, operatorSSEEvent("state_update", marker))
		w.(http.Flusher).Flush()
		ping := time.NewTicker(5 * time.Second)
		defer ping.Stop()
		late := time.NewTimer(31 * time.Second)
		defer late.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ping.C:
				_, _ = io.WriteString(w, "event: ping\ndata: {\"type\":\"ping\"}\n\n")
				w.(http.Flusher).Flush()
			case <-late.C:
				_, _ = io.WriteString(w, operatorSSEEvent("new_message", map[string]any{"type": "new_message", "agent_id": "a", "data": map[string]any{"seq": 1, "type": "unknown_future", "data": map[string]string{"value": "after ordinary deadline"}}}))
				w.(http.Flusher).Flush()
				<-ctx.Done()
				return
			}
		}
	})
	out := &cancelOutput{cancel: cancel}
	cmd := operatorFollowCmd(t, ctx, out)
	cmd.SetErr(io.Discard)
	if err := operatorFollow(cmd, &operatorClient{HTTP: srv.Client(), BaseURL: srv.URL}, "a", 0); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.buf.String(), "after ordinary deadline") {
		t.Fatalf("healthy observation ended early: %s", out.buf.String())
	}
}
