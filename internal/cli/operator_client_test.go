package cli

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestOperatorClientRequestPreservesRawJSONAndHeaders(t *testing.T) {
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("X-Test") != "yes" || r.Method != http.MethodPost {
			t.Errorf("request = %s, header %q", r.Method, r.Header.Get("X-Test"))
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"unknown":9007199254740993,"nested":{"x":true}}`)), Header: make(http.Header)}, nil
	})
	c := &operatorClient{HTTP: &http.Client{Transport: transport}, BaseURL: "http://operator.test"}
	got, err := c.Request(context.Background(), http.MethodPost, "/api/test", map[string]any{"x": 1}, http.Header{"X-Test": {"yes"}})
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `{"unknown":9007199254740993,"nested":{"x":true}}` {
		t.Fatalf("raw JSON changed: %s", got)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestOperatorInputRejectsDuplicateFields(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("request-file", "-", "")
	cmd.SetIn(strings.NewReader(`{"a":1,"a":2}`))
	if _, err := operatorInput(cmd); err == nil {
		t.Fatal("duplicate field accepted")
	}
}

func TestOperatorTextPreservesBytesAndRejectsConflictingSources(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("text", "", "")
	cmd.Flags().String("text-file", "", "")
	cmd.Flags().Set("text", "line 1\nline 2\n")
	value, ok, err := operatorText(cmd)
	if err != nil || !ok || value != "line 1\nline 2\n" {
		t.Fatalf("text = %q, %v, %v", value, ok, err)
	}
	cmd.Flags().Set("text-file", "-")
	if _, _, err := operatorText(cmd); err == nil {
		t.Fatal("conflicting sources accepted")
	}
}

func TestOperatorDiagnosticEscapesHumanControlData(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.Flags().Bool("json", false, "")
	var out strings.Builder
	cmd.SetErr(&out)
	operatorDiagnostic(cmd, &operatorError{Code: "bad", Message: "bad\x1b[31m", ExitCode: 1})
	if strings.Contains(out.String(), "\x1b") {
		t.Fatal("raw escape emitted")
	}
	if !strings.Contains(out.String(), `bad\x1b`) {
		t.Fatalf("diagnostic = %q", out.String())
	}
}
