package cli

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/transcript"
	"github.com/spf13/cobra"
)

const (
	operatorSSEFrameLimit = 16 << 20
	operatorFollowEvents  = 256
	operatorFollowBytes   = 32 << 20
)

func operatorContext(cmd *cobra.Command) (context.Context, context.CancelFunc, error) {
	timeout, err := cmd.Flags().GetDuration("timeout")
	if err != nil {
		return cmd.Context(), func() {}, nil
	}
	if timeout <= 0 {
		return nil, nil, operatorInvalid("--timeout must be positive")
	}
	ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
	return ctx, cancel, nil
}

func newAgentTranscriptCmd() *cobra.Command {
	c := &cobra.Command{Use: "transcript <id>", Short: "Read durable history or follow until interruption", Args: cobra.ExactArgs(1)}
	addOperatorFlags(c)
	c.Flags().Int64("since-seq", 0, "durable cursor; read events after this sequence")
	c.Flags().Int64("before-seq", 0, "non-follow read ending before this sequence")
	c.Flags().Int("limit", 0, "non-follow window size, at most 500")
	c.Flags().Bool("include-meta", false, "include retained session metadata")
	c.Flags().Bool("follow", false, "observe live updates; Ctrl-C leaves work running")
	c.Long = "Follow emits ordered durable events once, plus live-only reasoning activity. Gaps are reported on stderr and recovered from retained history. --timeout bounds ordinary reads; follow has a 30-second liveness/recovery window. History is limited to 64 MiB; use --since-seq for a smaller suffix."
	c.Example = "chuck agent transcript AGENT_ID --since-seq 42 --follow --json"
	c.RunE = func(c *cobra.Command, args []string) error {
		since, _ := c.Flags().GetInt64("since-seq")
		before, _ := c.Flags().GetInt64("before-seq")
		limit, _ := c.Flags().GetInt("limit")
		follow, _ := c.Flags().GetBool("follow")
		meta, _ := c.Flags().GetBool("include-meta")
		if since < 0 || before < 0 || limit < 0 || limit > 500 {
			return operatorInvalid("sequences must be nonnegative and --limit must be 0–500")
		}
		if follow && (limit != 0 || before != 0) {
			return operatorInvalid("--follow cannot be combined with --limit or --before-seq")
		}
		client, err := operatorClientFactory()
		if err != nil {
			return err
		}
		if follow {
			timeout, _ := c.Flags().GetDuration("timeout")
			if timeout <= 0 {
				return operatorInvalid("--timeout must be positive")
			}
			ctx, cancel := signal.NotifyContext(c.Context(), os.Interrupt)
			defer cancel()
			c.SetContext(ctx)
			signal.Ignore(syscall.SIGPIPE)
			defer signal.Reset(syscall.SIGPIPE)
			return operatorFollow(c, client, args[0], since)
		}
		ctx, cancel, err := operatorContext(c)
		if err != nil {
			return err
		}
		defer cancel()
		q := url.Values{"since_seq": {fmt.Sprint(since)}, "include_meta": {fmt.Sprint(meta)}}
		if before > 0 {
			q.Set("before_seq", fmt.Sprint(before))
		}
		if limit > 0 {
			q.Set("limit", fmt.Sprint(limit))
		}
		data, err := client.Request(ctx, "GET", "/api/sessions/"+url.PathEscape(args[0])+"/transcript?"+q.Encode(), nil, nil)
		if err != nil {
			return operatorHistoryError(err, since)
		}
		events, err := operatorHistoryEvents(data, args[0])
		if err != nil {
			return err
		}
		if len(events) == 0 {
			if err := operatorVerifyHistory(ctx, client, args[0]); err != nil {
				return err
			}
		}
		if machine, _ := c.Flags().GetBool("json"); machine {
			return operatorOutput(c, data)
		}
		for _, event := range events {
			if err := operatorFollowLine(c, event); err != nil {
				return err
			}
		}
		return nil
	}
	return c
}

type operatorSSEFrame struct {
	event string
	data  []byte
}
type operatorObserver struct {
	frames chan operatorSSEFrame
	done   chan error
	bytes  atomic.Int64
	items  atomic.Int64
	close  func()
}

func (o *operatorObserver) release(f operatorSSEFrame) {
	o.bytes.Add(-int64(len(f.data)))
	o.items.Add(-1)
}

// One connection and one catch-up request own a bounded tail. Retained history
// is discarded after emission; global SSE ids never become transcript cursors.
func operatorFollow(cmd *cobra.Command, client *operatorClient, agentID string, since int64) error {
	ctx := cmd.Context()
	last := since
	recoveryEnd := time.Now().Add(30 * time.Second)
	for {
		if ctx.Err() != nil {
			return nil
		}
		if !time.Now().Before(recoveryEnd) {
			return operatorGap(last, "recovery window expired")
		}
		o, err := operatorSubscribe(ctx, client, agentID, recoveryEnd)
		if err != nil {
			var op *operatorError
			if errors.As(err, &op) && op.HTTPStatus != 0 {
				return operatorHistoryError(err, last)
			}
			operatorFollowProgress(cmd, fmt.Sprintf("connection unavailable; reconnecting after sequence %d", last))
			if !operatorFollowWait(ctx, recoveryEnd) {
				if ctx.Err() != nil {
					return nil
				}
				return operatorGap(last, "recovery window expired")
			}
			continue
		}
		err = func() error {
			defer o.close()
			idle := time.NewTimer(time.Until(recoveryEnd))
			defer idle.Stop()
			tail := make([]operatorSSEFrame, 0, 16)
			ready := false
			for !ready {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-idle.C:
					return operatorGap(last, "hydration timed out")
				case f, ok := <-o.frames:
					if !ok {
						return <-o.done
					}
					if f.event == "state_update" && operatorHydrated(f.data) {
						o.release(f)
						ready = true
						break
					}
					if f.event == "new_message" {
						tail = append(tail, f)
					} else {
						o.release(f)
					}
				}
			}
			catchup := func() error {
				end := time.Now().Add(30 * time.Second)
				if recoveryEnd.Before(end) {
					end = recoveryEnd
				}
				readCtx, cancel := context.WithDeadline(ctx, end)
				defer cancel()
				return operatorFollowCatchup(readCtx, client, cmd, agentID, &last)
			}
			if err := catchup(); err != nil {
				return err
			}
			consume := func(f operatorSSEFrame) error {
				defer o.release(f)
				if f.event == "ping" {
					// Recover a dropped terminal tail even when no later durable
					// event arrives to reveal its sequence gap.
					return catchup()
				}
				if f.event == "state_update" {
					return nil
				}
				err := operatorFollowFrame(cmd, f, agentID, &last)
				if operatorIsGap(err) {
					operatorFollowProgress(cmd, fmt.Sprintf("history gap after sequence %d; catching up", last))
					if err := catchup(); err != nil {
						return err
					}
					return operatorFollowFrame(cmd, f, agentID, &last)
				}
				return err
			}
			for _, f := range tail {
				if err := consume(f); err != nil {
					return err
				}
			}
			recoveryEnd = time.Now().Add(30 * time.Second)
			resetOperatorFollowTimer(idle)
			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-idle.C:
					recoveryEnd = time.Now().Add(30 * time.Second)
					return &operatorError{Code: "stream_disconnected", Message: "ping timed out", ExitCode: 1}
				case f, ok := <-o.frames:
					if !ok {
						return <-o.done
					}
					resetOperatorFollowTimer(idle)
					recoveryEnd = time.Now().Add(30 * time.Second)
					if err := consume(f); err != nil {
						return err
					}
				}
			}
		}()
		if ctx.Err() != nil || errors.Is(err, syscall.EPIPE) {
			return nil
		}
		var op *operatorError
		if errors.As(err, &op) && op.Code != "stream_disconnected" && op.Code != "stream_overflow" && op.Code != "dashboard_unavailable" {
			return operatorHistoryError(err, last)
		}
		operatorFollowProgress(cmd, fmt.Sprintf("connection gap; reconnecting after sequence %d", last))
		if !operatorFollowWait(ctx, recoveryEnd) {
			if ctx.Err() != nil {
				return nil
			}
			return operatorGap(last, "recovery window expired")
		}
	}
}

func operatorFollowWait(ctx context.Context, end time.Time) bool {
	d := 2 * time.Second
	if left := time.Until(end); left < d {
		d = left
	}
	if d <= 0 {
		return false
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return time.Now().Before(end)
	}
}

func operatorSubscribe(ctx context.Context, client *operatorClient, agentID string, end time.Time) (*operatorObserver, error) {
	streamCtx, cancel := context.WithCancel(ctx)
	// Bound opening headers, then stop the timer so healthy observers can live.
	timer := time.AfterFunc(time.Until(end), cancel)
	req, err := http.NewRequestWithContext(streamCtx, "GET", strings.TrimRight(client.BaseURL, "/")+"/api/events", nil)
	if err != nil {
		timer.Stop()
		cancel()
		return nil, operatorInvalid("invalid stream target")
	}
	resp, err := client.httpClient().Do(req)
	timer.Stop()
	if err != nil {
		cancel()
		return nil, &operatorError{Code: "dashboard_unavailable", Message: "cannot reach dashboard; run `chuck dashboard start`", ExitCode: 3}
	}
	if resp.StatusCode != 200 {
		defer resp.Body.Close()
		defer cancel()
		data, _ := readOperatorBounded(resp.Body, 1<<20)
		return nil, operatorHTTPError(resp.StatusCode, data)
	}
	if !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
		resp.Body.Close()
		cancel()
		return nil, &operatorError{Code: "invalid_response", Message: "dashboard did not return an event stream", ExitCode: 1}
	}
	o := &operatorObserver{frames: make(chan operatorSSEFrame, operatorFollowEvents), done: make(chan error, 1)}
	finished := make(chan struct{})
	o.close = func() { cancel(); _ = resp.Body.Close(); <-finished }
	go func() {
		defer close(finished)
		defer close(o.frames)
		defer resp.Body.Close()
		err := operatorParseSSE(streamCtx, resp.Body, func(f operatorSSEFrame) error {
			if f.event != "ping" && !operatorHydrated(f.data) && operatorFrameAgent(f.data) != agentID {
				return nil
			}
			if f.event != "ping" && f.event != "state_update" && f.event != "new_message" && f.event != "runtime_activity" {
				return nil
			}
			if o.items.Add(1) > operatorFollowEvents || o.bytes.Add(int64(len(f.data))) > operatorFollowBytes {
				return &operatorError{Code: "stream_overflow", Message: "observer tail overflowed", ExitCode: 1}
			}
			select {
			case o.frames <- f:
				return nil
			case <-streamCtx.Done():
				return streamCtx.Err()
			default:
				return &operatorError{Code: "stream_overflow", Message: "observer tail overflowed", ExitCode: 1}
			}
		})
		if err == nil || errors.Is(err, io.EOF) {
			err = &operatorError{Code: "stream_disconnected", Message: "event stream ended", ExitCode: 1}
		}
		o.done <- err
	}()
	return o, nil
}

// Parse incrementally; a newline-free producer cannot allocate beyond the bound.
func operatorParseSSE(ctx context.Context, r io.Reader, emit func(operatorSSEFrame) error) error {
	reader := bufio.NewReaderSize(r, 64<<10)
	var line, data bytes.Buffer
	event := ""
	frameBytes := 0
	for {
		fragment, err := reader.ReadSlice('\n')
		frameBytes += len(fragment)
		if frameBytes > operatorSSEFrameLimit {
			return &operatorError{Code: "invalid_response", Message: "SSE frame exceeds 16 MiB", ExitCode: 1}
		}
		line.Write(fragment)
		if errors.Is(err, bufio.ErrBufferFull) {
			continue
		}
		if err != nil {
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		text := strings.TrimSuffix(strings.TrimSuffix(line.String(), "\n"), "\r")
		line.Reset()
		if text == "" {
			if data.Len() > 0 {
				payload := append([]byte(nil), bytes.TrimSuffix(data.Bytes(), []byte("\n"))...)
				if !json.Valid(payload) {
					return &operatorError{Code: "invalid_response", Message: "invalid SSE JSON", ExitCode: 1}
				}
				if err := emit(operatorSSEFrame{event: event, data: payload}); err != nil {
					return err
				}
			}
			data.Reset()
			event = ""
			frameBytes = 0
			continue
		}
		if strings.HasPrefix(text, "event:") {
			event = strings.TrimSpace(strings.TrimPrefix(text, "event:"))
		}
		if strings.HasPrefix(text, "data:") {
			value := strings.TrimPrefix(text, "data:")
			value = strings.TrimPrefix(value, " ")
			data.WriteString(value)
			data.WriteByte('\n')
		}
	}
}

func operatorReadSSE(ctx context.Context, r io.Reader, out chan<- operatorSSEFrame) {
	_ = operatorParseSSE(ctx, r, func(f operatorSSEFrame) error {
		select {
		case out <- f:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		default:
			return errors.New("tail full")
		}
	})
}
func operatorHydrated(data []byte) bool {
	var e struct {
		AgentID string `json:"agent_id"`
		Data    struct {
			Hydrated bool `json:"hydrated"`
		} `json:"data"`
	}
	return json.Unmarshal(data, &e) == nil && e.AgentID == "__hydrated__" && e.Data.Hydrated
}
func operatorFrameAgent(data []byte) string {
	var e struct {
		AgentID string `json:"agent_id"`
	}
	_ = json.Unmarshal(data, &e)
	return e.AgentID
}

func operatorHistoryEvents(data json.RawMessage, id string) ([]json.RawMessage, error) {
	var e struct {
		AgentID string          `json:"agent_id"`
		Events  json.RawMessage `json:"events"`
	}
	if json.Unmarshal(data, &e) != nil || len(e.Events) == 0 || bytes.Equal(e.Events, []byte("null")) || (e.AgentID != "" && e.AgentID != id) {
		return nil, &operatorError{Code: "invalid_response", Message: "transcript response is missing events or names another agent", ExitCode: 1}
	}
	var events []json.RawMessage
	if json.Unmarshal(e.Events, &events) != nil {
		return nil, &operatorError{Code: "invalid_response", Message: "invalid transcript event list", ExitCode: 1}
	}
	return events, nil
}
func operatorFollowCatchup(ctx context.Context, client *operatorClient, cmd *cobra.Command, id string, last *int64) error {
	data, err := client.Request(ctx, "GET", "/api/sessions/"+url.PathEscape(id)+"/transcript?include_meta=true&since_seq="+strconv.FormatInt(*last, 10), nil, nil)
	if err != nil {
		return operatorHistoryError(err, *last)
	}
	events, err := operatorHistoryEvents(data, id)
	if err != nil {
		return err
	}
	if len(events) == 0 {
		if err := operatorVerifyHistory(ctx, client, id); err != nil {
			return err
		}
	}
	for _, raw := range events {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := operatorFollowEvent(cmd, raw, last); err != nil {
			return err
		}
	}
	return nil
}
func operatorFollowFrame(cmd *cobra.Command, f operatorSSEFrame, id string, last *int64) error {
	if operatorFrameAgent(f.data) != id {
		return nil
	}
	if f.event == "runtime_activity" {
		if machine, _ := cmd.Flags().GetBool("json"); machine {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), string(f.data))
			return err
		}
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "live-only reasoning: "+safeOperatorText(string(f.data)))
		return err
	}
	if f.event != "new_message" {
		return nil
	}
	var e struct {
		Data json.RawMessage `json:"data"`
	}
	if json.Unmarshal(f.data, &e) != nil {
		return &operatorError{Code: "invalid_response", Message: "invalid durable event envelope", ExitCode: 1}
	}
	return operatorFollowEvent(cmd, e.Data, last)
}
func operatorFollowEvent(cmd *cobra.Command, raw json.RawMessage, last *int64) error {
	var e runtime.Event
	if json.Unmarshal(raw, &e) != nil || e.Type == "" {
		return &operatorError{Code: "invalid_response", Message: "durable event lacks sequence or type", ExitCode: 1}
	}
	if e.Seq == 0 && e.Type == runtime.EvSessionMeta {
		return nil
	}
	if e.Seq < 1 {
		return &operatorError{Code: "invalid_response", Message: "durable event has an invalid sequence", ExitCode: 1}
	}
	if _, err := transcript.ProjectEvent(e); err != nil {
		return &operatorError{Code: "invalid_response", Message: "corrupt durable event payload", ExitCode: 1}
	}
	if e.Seq <= *last {
		return nil
	}
	if e.Seq != *last+1 {
		return operatorGap(*last, "missing durable sequence")
	}
	if err := operatorFollowLine(cmd, raw); err != nil {
		return err
	}
	*last = e.Seq
	return nil
}

func operatorVerifyHistory(ctx context.Context, client *operatorClient, id string) error {
	data, err := client.Request(ctx, "GET", "/api/sessions/"+url.PathEscape(id)+"/transcript?include_meta=true&limit=1", nil, nil)
	if err != nil {
		return err
	}
	events, err := operatorHistoryEvents(data, id)
	if err != nil {
		return err
	}
	if len(events) == 0 {
		return &operatorError{Code: "history_unavailable", Message: "retained history is missing or unreadable for " + safeOperatorText(id), ExitCode: 1}
	}
	return nil
}
func operatorFollowLine(cmd *cobra.Command, raw []byte) error {
	if machine, _ := cmd.Flags().GetBool("json"); machine {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), string(raw))
		return err
	}
	var e runtime.Event
	if json.Unmarshal(raw, &e) != nil {
		return &operatorError{Code: "invalid_response", Message: "invalid transcript event", ExitCode: 1}
	}
	p, err := transcript.ProjectEvent(e)
	if err != nil {
		return &operatorError{Code: "invalid_response", Message: "corrupt transcript event payload", ExitCode: 1}
	}
	parts := []string{}
	for _, part := range p.Parts {
		if part.Text != "" {
			parts = append(parts, part.Text)
		}
	}
	text := fmt.Sprintf("%d %s: %s", e.Seq, e.Type, strings.Join(parts, " "))
	if len(parts) == 0 {
		text = string(raw)
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), safeOperatorText(text))
	return err
}
func operatorGap(last int64, reason string) error {
	return &operatorError{Code: "stream_gap", Message: fmt.Sprintf("%s after durable sequence %d", reason, last), ExitCode: 1}
}
func operatorIsGap(err error) bool {
	var op *operatorError
	return errors.As(err, &op) && op.Code == "stream_gap"
}
func operatorHistoryError(err error, last int64) error {
	var op *operatorError
	if errors.As(err, &op) {
		copy := *op
		if op.Code == "response_too_large" {
			copy.Message = "history exceeds 64 MiB; use --since-seq for a smaller suffix"
		}
		copy.Message += fmt.Sprintf("; last durable sequence %d", last)
		return &copy
	}
	return err
}
func operatorFollowProgress(cmd *cobra.Command, message string) {
	operatorDiagnostic(cmd, &operatorError{Code: "stream_reconnect", Message: message, ExitCode: 1})
}
func resetOperatorFollowTimer(timer *time.Timer) {
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
	timer.Reset(30 * time.Second)
}
