package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestOperatorErrorAndOutputContracts(t *testing.T) {
	for _, tc := range []struct {
		body, code   string
		status, exit int
	}{
		{`{"error":{"code":"provider_executable_missing","message":"Install the provider"}}`, "provider_executable_missing", 422, 3},
		{`{"error":{"code":"conflict","message":"Stale revision"}}`, "conflict", 409, 1},
		{`{"error":"validation_failed","errors":[{"field":"cwd","message":"required"}]}`, "validation_failed", 400, 1},
		{`<html>private provider dump</html>`, "api_error", 500, 1},
	} {
		t.Run(tc.code, func(t *testing.T) {
			err := operatorHTTPError(tc.status, []byte(tc.body))
			var op *operatorError
			if !errors.As(err, &op) || op.Code != tc.code || op.ExitCode != tc.exit {
				t.Fatalf("error %+v", err)
			}
			cmd := &cobra.Command{}
			cmd.Flags().Bool("json", true, "")
			var out, diag bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&diag)
			operatorDiagnostic(cmd, err)
			if out.Len() != 0 || !json.Valid(bytes.TrimSpace(diag.Bytes())) || bytes.Contains(diag.Bytes(), []byte("private provider dump")) {
				t.Fatalf("streams stdout=%q stderr=%q", out.String(), diag.String())
			}
		})
	}
	cmd := &cobra.Command{}
	cmd.Flags().Bool("json", true, "")
	var out bytes.Buffer
	cmd.SetOut(&out)
	result := json.RawMessage(`{"results":[{"agent_id":"a","ok":true},{"agent_id":"b","ok":false,"error":{"code":"stop_failed","message":"Inspect b"}}],"future":9007199254740993}`)
	err := operatorOutput(cmd, result)
	if operatorExitCode(err) != 1 || out.String() != string(result)+"\n" {
		t.Fatalf("bulk err=%v output=%q", err, out.String())
	}
}

func TestOperatorInputLimitsAndObjectShape(t *testing.T) {
	for _, input := range []string{`[]`, `null`, `{} {}`, `{"a":1,"a":2}`, "{\"text\":\"\xff\"}", strings.Repeat(" ", 1048577)} {
		cmd := &cobra.Command{}
		addOperatorFlags(cmd)
		_ = cmd.Flags().Set("request-file", "-")
		cmd.SetIn(strings.NewReader(input))
		_, err := operatorInput(cmd)
		if err == nil || operatorExitCode(err) != 2 {
			t.Fatalf("input accepted, bytes=%d err=%v", len(input), err)
		}
	}
	cmd := &cobra.Command{}
	addOperatorTextFlags(cmd)
	_ = cmd.Flags().Set("text-file", "-")
	cmd.SetIn(strings.NewReader(strings.Repeat("x", 1048577)))
	if _, _, err := operatorText(cmd); err == nil || operatorExitCode(err) != 2 {
		t.Fatalf("oversized text %v", err)
	}
}

func TestOperatorHumanReceiptsAndStoredFailures(t *testing.T) {
	for _, tc := range []struct {
		name, body, want string
		exit             int
	}{
		{"send", `{"agent_id":"a","delivery":"held"}`, "delivery=held", 0},
		{"steer", `{"agent_id":"a","outcome":"steered"}`, "outcome=steered", 0},
		{"stop", `{"results":[{"agent_id":"a","ok":true,"stopped":true},{"agent_id":"b","ok":false,"error":{"code":"stop_failed","message":"Inspect b"},"warning":"Still active"}]}`, "Inspect b", 1},
		{"list", `[{"id":"failed-run","state":"failed","ok":false}]`, "state=failed", 0},
		{"archive", `{"project":{"id":"p"},"stopped_agent_ids":[],"archived_agent_ids":[],"checkout_warning":"the checkout could not be deleted: dirty"}`, "checkout not deleted: the checkout could not be deleted: dirty", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := &cobra.Command{Use: tc.name}
			cmd.Flags().Bool("json", false, "")
			var out bytes.Buffer
			cmd.SetOut(&out)
			err := operatorOutput(cmd, json.RawMessage(tc.body))
			exit := 0
			if err != nil {
				exit = operatorExitCode(err)
			}
			if exit != tc.exit || !strings.Contains(out.String(), tc.want) || strings.Contains(out.String(), "completed") {
				t.Fatalf("output=%q error=%v", out.String(), err)
			}
		})
	}
	cmd := &cobra.Command{}
	cmd.Flags().Bool("json", true, "")
	var diag bytes.Buffer
	cmd.SetErr(&diag)
	message := "provider says \x1b[31m\u009b2J"
	operatorDiagnostic(cmd, &operatorError{Code: "conflict", Message: message, ExitCode: 1})
	var decoded struct{ Error struct{ Message string } }
	if err := json.Unmarshal(diag.Bytes(), &decoded); err != nil || decoded.Error.Message != message {
		t.Fatalf("machine diagnostic changed data: %q %v", diag.String(), err)
	}
}

func TestOperatorMutationUncertaintyDoesNotReplay(t *testing.T) {
	for _, mode := range []string{"transport", "invalid-response"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			c := &operatorClient{BaseURL: "http://127.0.0.1", HTTP: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				if mode == "transport" {
					return nil, errors.New("private provider path secret")
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("not JSON"))}, nil
			})}}
			_, err := c.Request(context.Background(), "POST", "/api/pipeline-runs", map[string]any{"request_id": "retry-1"}, nil)
			var op *operatorError
			if !errors.As(err, &op) || op.Code != "result_unknown" || calls != 1 || !strings.Contains(op.Message, "retry-1") || strings.Contains(op.Message, "private provider path secret") {
				t.Fatalf("calls %d error %#v", calls, err)
			}
		})
	}
}

func TestOperatorHumanOutputEscapesControls(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("json", false, "")
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := operatorOutput(cmd, json.RawMessage("{\"name\":\"\\u001b[31m\u009b2J\"}")); err != nil {
		t.Fatal(err)
	}
	if strings.ContainsAny(out.String(), "\x1b\u009b") {
		t.Fatalf("terminal control leaked %q", out.String())
	}
}
