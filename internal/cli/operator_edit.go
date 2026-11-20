package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/spf13/cobra"
)

// errOperatorRevised marks a committed conversation revision: events already
// emitted may be superseded, so follow must re-read the active transcript.
var errOperatorRevised = &operatorError{Code: "history_revised", Message: "conversation revised", ExitCode: 1}

func newAgentEditLastCmd() *cobra.Command {
	c := &cobra.Command{Use: "edit-last <id>", Short: "Edit the latest user message and rerun from it", Args: cobra.ExactArgs(1)}
	addOperatorFlags(c)
	addOperatorTextFlags(c)
	c.Flags().String("request-id", "", "retry id; generated and printed on stderr if omitted. Reuse it after an uncertain result")
	c.Flags().Int64("message-seq", 0, "exact sequence of the message to replace")
	c.Flags().Int64("expected-revision", 0, "conversation revision the edit was based on")
	c.Long = "Replace the latest message. Request JSON: {request_id,message_seq,expected_revision,text}. Nothing reselects a newer message; stale input fails with edit_conflict. Acceptance is not completion: query `agent edit-last-status`. After an unknown result, retry with the same request id."
	c.Example = "chuck agent edit-last AGENT_ID --message-seq 12 --expected-revision 0 --text-file new.txt --json"
	c.RunE = func(c *cobra.Command, args []string) error {
		body, err := operatorInput(c)
		if err != nil {
			return err
		}
		text, present, err := operatorText(c)
		if err != nil {
			return err
		}
		if present {
			if err := operatorMerge(body, "text", text); err != nil {
				return operatorInvalid(err.Error())
			}
		}
		for _, f := range []struct{ flag, key string }{{"message-seq", "message_seq"}, {"expected-revision", "expected_revision"}} {
			if c.Flags().Changed(f.flag) {
				v, _ := c.Flags().GetInt64(f.flag)
				if err := operatorMerge(body, f.key, v); err != nil {
					return operatorInvalid(err.Error())
				}
			}
		}
		generated := false
		if c.Flags().Changed("request-id") {
			v, _ := c.Flags().GetString("request-id")
			if err := operatorMerge(body, "request_id", v); err != nil {
				return operatorInvalid(err.Error())
			}
		} else if _, ok := body["request_id"]; !ok {
			body["request_id"], _ = json.Marshal(newPipelineRequestID())
			generated = true
		}
		var id string
		if json.Unmarshal(body["request_id"], &id) != nil || !operatorValidReceiptID(id) {
			return operatorInvalid("request_id must be 1–128 ASCII letters, digits, - or _")
		}
		var seq, rev int64
		if json.Unmarshal(body["message_seq"], &seq) != nil || seq < 1 {
			return operatorInvalid("message_seq must be a positive integer via --message-seq or --request-file")
		}
		if json.Unmarshal(body["expected_revision"], &rev) != nil || rev < 0 {
			return operatorInvalid("expected_revision must be a nonnegative integer via --expected-revision or --request-file")
		}
		var msg string
		if json.Unmarshal(body["text"], &msg) != nil {
			return operatorInvalid("text is required via --text, --text-file or --request-file")
		}
		if len(msg) > 64<<10 {
			return operatorInvalid("message text exceeds 64 KiB")
		}
		if generated {
			_, _ = fmt.Fprintln(c.ErrOrStderr(), "request id "+id+"; reuse it to retry after an uncertain result")
		}
		return operatorRun(c, http.MethodPost, "/api/sessions/"+url.PathEscape(args[0])+"/edit-last", body, nil)
	}
	return c
}

func newAgentEditLastStatusCmd() *cobra.Command {
	c := &cobra.Command{Use: "edit-last-status <id> <request-id>", Short: "Show a last-message Edit's phase and delivery", Args: cobra.ExactArgs(2)}
	c.Flags().Bool("json", false, "write machine-readable JSON")
	c.Flags().Duration("timeout", 30*time.Second, "request timeout")
	c.Example = "chuck agent edit-last-status AGENT_ID REQUEST_ID --json"
	c.RunE = func(c *cobra.Command, args []string) error {
		if !operatorValidReceiptID(args[1]) {
			return operatorInvalid("request id must be 1–128 ASCII letters, digits, - or _")
		}
		return operatorRun(c, http.MethodGet, "/api/sessions/"+url.PathEscape(args[0])+"/edit-last/"+url.PathEscape(args[1]), nil, nil)
	}
	return c
}

// operatorFollowRefetch re-reads the active transcript from the start after a
// committed revision and emits it; original event ids and seqs are unchanged.
func operatorFollowRefetch(ctx context.Context, client *operatorClient, cmd *cobra.Command, id string, last *int64) error {
	operatorFollowProgress(cmd, "conversation revised; re-reading active transcript")
	data, err := client.Request(ctx, "GET", "/api/sessions/"+url.PathEscape(id)+"/transcript?include_meta=true", nil, nil)
	if err != nil {
		return operatorHistoryError(err, *last)
	}
	events, err := operatorHistoryEvents(data, id)
	if err != nil {
		return err
	}
	var high int64
	for _, raw := range events {
		var e runtime.Event
		if json.Unmarshal(raw, &e) != nil || e.Type == "" {
			return &operatorError{Code: "invalid_response", Message: "durable event lacks sequence or type", ExitCode: 1}
		}
		if e.Seq == 0 && e.Type == runtime.EvSessionMeta {
			continue
		}
		if err := operatorFollowLine(cmd, raw); err != nil {
			return err
		}
		if e.Seq > high {
			high = e.Seq
		}
	}
	if high > *last {
		*last = high
	}
	return nil
}

func operatorIsRevised(err error) bool {
	return errors.Is(err, errOperatorRevised)
}
