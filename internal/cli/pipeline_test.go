package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/AsaphNoam/Chuck/internal/pipeline"
)

func TestParsePipelineCLIValues(t *testing.T) {
	inputs, err := parseKeyValues([]string{"spec=Do it", "note=a=b"})
	if err != nil || !reflect.DeepEqual(inputs, map[string]string{"spec": "Do it", "note": "a=b"}) {
		t.Fatalf("inputs = %v err=%v", inputs, err)
	}
	assignments, err := parseStageAssignments([]string{"work=codex/gpt", "review=claude/sonnet"})
	if err != nil || assignments["work"] != (pipeline.RuntimeAssignment{Backend: "codex", Model: "gpt"}) {
		t.Fatalf("assignments = %v err=%v", assignments, err)
	}
	if _, err := parseStageAssignments([]string{"work=codex"}); err == nil {
		t.Fatal("invalid assignment succeeded")
	}
	rooms, err := parseRoomAssignments([]string{"debate.pro=codex/gpt", "debate.judge=claude/opus"}, []string{"debate=claude/sonnet"})
	if err != nil || rooms["debate"].Participants["judge"].Model != "opus" || rooms["debate"].Judge.Model != "sonnet" || rooms["debate"].Participants["pro"].Backend != "codex" {
		t.Fatalf("rooms = %+v err=%v", rooms, err)
	}
	if _, err := parseRoomAssignments([]string{"debate=codex/gpt"}, nil); err == nil {
		t.Fatal("room participant without stage.participant succeeded")
	}
}

// FS-22.R11: pipeline refusals show the server reason and its field
// diagnostics, bounded, in both the operator and legacy paths, never raw JSON.
func TestPipelineErrorsShowAPIDiagnostics(t *testing.T) {
	var diags []string
	for i := 0; i < 12; i++ {
		diags = append(diags, fmt.Sprintf(`{"field":"stages[%d].id","code":"required","message":"stage id is required"}`, i))
	}
	body := `{"error":{"code":"validation","message":"pipeline validation failed","details":{"pipeline_code":"invalid_template","diagnostics":[` + strings.Join(diags, ",") + `]}}}`
	err := operatorHTTPError(http.StatusBadRequest, []byte(body))
	var op *operatorError
	if !errors.As(err, &op) || op.Code != "validation" || op.ExitCode != 1 ||
		!strings.HasPrefix(op.Message, "pipeline validation failed: stages[0].id: stage id is required; ") ||
		!strings.HasSuffix(op.Message, "; and 2 more") || strings.Contains(op.Message, "stages[10]") {
		t.Fatalf("operator error = %#v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()
	u, _ := url.Parse(server.URL)
	port, _ := strconv.Atoi(u.Port())
	home := t.TempDir()
	t.Setenv("CHUCK_HOME", home)
	if err := writePidfile(home, pidInfo{PID: os.Getpid(), Port: port}); err != nil {
		t.Fatal(err)
	}
	err = runPipelineRequest(http.MethodPost, "/api/pipelines/validate", map[string]any{"id": "q"})
	if err == nil || strings.Contains(err.Error(), `{"error"`) || !strings.Contains(err.Error(), "pipeline request failed (400): pipeline validation failed: stages[0].id: stage id is required") {
		t.Fatalf("legacy error = %v", err)
	}
}

// FS-14.A1: CLI commands remain thin clients over the same REST contract.
func TestPipelineAPIRequestContract(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"run":{"run":{"run_id":"pr_test"}}}`))
	}))
	defer server.Close()
	portPart := strings.TrimPrefix(server.URL, "http://127.0.0.1:")
	var port int
	if _, err := fmt.Sscanf(portPart, "%d", &port); err != nil {
		t.Fatal(err)
	}
	status, _, err := pipelineAPIRequest(port, http.MethodPost, "/api/pipeline-runs", map[string]any{"request_id": "r1"})
	if err != nil || status != http.StatusCreated || gotMethod != http.MethodPost || gotPath != "/api/pipeline-runs" || gotBody["request_id"] != "r1" {
		t.Fatalf("request status=%d err=%v method=%s path=%s body=%v", status, err, gotMethod, gotPath, gotBody)
	}
}
