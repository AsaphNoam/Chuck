package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/spf13/cobra"

	"github.com/AsaphNoam/Chuck/internal/config"
)

const (
	operatorInputLimit    int64 = 1 << 20
	operatorResponseLimit int64 = 64 << 20
)

// operatorError is the stable, caller-inspectable error returned by operator
// commands. HTTPStatus is zero for local and transport errors.
type operatorError struct {
	Code       string
	Message    string
	HTTPStatus int
	ExitCode   int
}

func (e *operatorError) Error() string { return e.Message }

type operatorClient struct {
	HTTP            *http.Client
	BaseURL         string
	responseHeaders http.Header // current response metadata, scoped to one invocation
}

func (c *operatorClient) httpClient() *http.Client {
	client := http.Client{}
	if c.HTTP != nil {
		client = *c.HTTP
	}
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &client
}

var operatorClientFactory = newOperatorClient

var errOperatorResponseLimit = errors.New("response byte limit exceeded")

func newOperatorClient() (*operatorClient, error) {
	store, err := config.New()
	if err != nil {
		return nil, &operatorError{Code: "invalid_input", Message: err.Error(), ExitCode: 2}
	}
	return &operatorClient{
		HTTP:    &http.Client{},
		BaseURL: fmt.Sprintf("http://127.0.0.1:%d", dashboardPort(store)),
	}, nil
}

func (c *operatorClient) Request(ctx context.Context, method, path string, body any, headers http.Header) (json.RawMessage, error) {
	var payload io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, operatorInputError(err)
		}
		payload = bytes.NewReader(data)
	}
	if headers == nil {
		headers = make(http.Header)
	}
	if body != nil {
		headers = cloneOperatorHeaders(headers)
		headers.Set("Content-Type", "application/json")
	}
	result, err := c.requestReader(ctx, method, path, payload, headers)
	if err != nil {
		var op *operatorError
		if errors.As(err, &op) && op.Code == "result_unknown" {
			if id := operatorRequestID(body); id != "" {
				op.Message += "; request id " + safeOperatorText(id)
			}
		}
	}
	return result, err
}

func operatorRequestID(body any) string {
	switch value := body.(type) {
	case map[string]any:
		for _, key := range []string{"request_id", "command_id"} {
			if id, ok := value[key].(string); ok {
				return id
			}
		}
	case map[string]json.RawMessage:
		for _, key := range []string{"request_id", "command_id"} {
			var id string
			if json.Unmarshal(value[key], &id) == nil {
				return id
			}
		}
	default:
		data, err := json.Marshal(body)
		if err == nil {
			var value map[string]json.RawMessage
			if json.Unmarshal(data, &value) == nil {
				return operatorRequestID(value)
			}
		}
	}
	return ""
}

// requestReader is the wire seam for callers sending multipart or otherwise
// pre-encoded request bodies. The caller owns payload; this helper owns the
// response and error contract.
func (c *operatorClient) requestReader(ctx context.Context, method, path string, payload io.Reader, headers http.Header) (json.RawMessage, error) {
	c.responseHeaders = nil
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(c.BaseURL, "/")+path, payload)
	if err != nil {
		return nil, operatorInputError(err)
	}
	for key, values := range headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	resp, err := c.httpClient().Do(req)
	if err != nil {
		code, message, exit := "dashboard_unavailable", "could not reach dashboard; run `chuck dashboard start` first", 3
		if method != http.MethodGet && method != http.MethodHead && method != http.MethodOptions {
			code, message = "result_unknown", "dashboard connection failed; result is unknown; inspect status before retrying or run `chuck dashboard start` if unavailable"
		}
		return nil, &operatorError{Code: code, Message: message, ExitCode: exit}
	}
	defer resp.Body.Close()
	c.responseHeaders = resp.Header.Clone()
	data, err := readOperatorBounded(resp.Body, operatorResponseLimit)
	if err != nil {
		code, exit, message := "dashboard_unavailable", 3, "dashboard response was interrupted; run `chuck dashboard start` if unavailable"
		if errors.Is(err, errOperatorResponseLimit) {
			code, exit, message = "response_too_large", 1, "response exceeds 64 MiB"
		}
		if isMutationMethod(method) {
			code = "result_unknown"
			message += "; result is unknown; inspect status before retrying"
		}
		return nil, &operatorError{Code: code, Message: message, HTTPStatus: resp.StatusCode, ExitCode: exit}
	}
	trimmed := bytes.TrimSpace(data)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, operatorHTTPError(resp.StatusCode, trimmed)
	}
	if len(trimmed) == 0 {
		return json.RawMessage("{}"), nil
	}
	if !json.Valid(trimmed) || !utf8.Valid(trimmed) {
		code, exit := "invalid_response", 1
		if isMutationMethod(method) {
			code, exit = "result_unknown", 1
		}
		return nil, &operatorError{Code: code, Message: "dashboard returned invalid JSON; result is unknown", HTTPStatus: resp.StatusCode, ExitCode: exit}
	}
	return json.RawMessage(append([]byte(nil), trimmed...)), nil
}

func cloneOperatorHeaders(in http.Header) http.Header {
	out := make(http.Header, len(in))
	for k, v := range in {
		out[k] = append([]string(nil), v...)
	}
	return out
}
func isMutationMethod(method string) bool {
	return method != http.MethodGet && method != http.MethodHead && method != http.MethodOptions
}
func operatorInputError(err error) *operatorError {
	return &operatorError{Code: "invalid_input", Message: err.Error(), ExitCode: 2}
}

func readOperatorBounded(r io.Reader, limit int64) ([]byte, error) {
	b, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > limit {
		return nil, fmt.Errorf("response exceeds %d MiB: %w", limit>>20, errOperatorResponseLimit)
	}
	return b, nil
}

func operatorHTTPError(status int, data []byte) error {
	code, message := "api_error", fmt.Sprintf("dashboard request failed (%d)", status)
	var envelope map[string]json.RawMessage
	if json.Unmarshal(data, &envelope) == nil {
		code = operatorRawString(envelope["code"], code)
		message = operatorRawString(envelope["message"], message)
		if raw := envelope["error"]; len(raw) > 0 && string(raw) != "null" {
			var nested map[string]json.RawMessage
			if json.Unmarshal(raw, &nested) == nil {
				code = operatorRawString(nested["code"], code)
				message = operatorRawString(nested["message"], message)
			} else {
				message = operatorRawString(raw, message)
				if code == "api_error" {
					var flat string
					if json.Unmarshal(raw, &flat) == nil && flat != "" && !strings.ContainsAny(flat, " \t\r\n") {
						code = safeOperatorText(flat)
					}
				}
			}
		}
		if raw := envelope["errors"]; len(raw) > 0 && string(raw) != "null" {
			message += ": " + operatorValidationDetails(raw)
		}
	}
	exit := 1
	if operatorExitClass(code) == 3 {
		exit = 3
	}
	return &operatorError{Code: code, Message: message, HTTPStatus: status, ExitCode: exit}
}

func operatorRawString(raw json.RawMessage, fallback string) string {
	if len(raw) == 0 {
		return fallback
	}
	var s string
	if json.Unmarshal(raw, &s) == nil && s != "" {
		return s
	}
	return fallback
}

func operatorValidationDetails(raw json.RawMessage) string {
	var value any
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if dec.Decode(&value) != nil {
		return "validation failed"
	}
	b, _ := json.Marshal(value)
	return safeOperatorText(string(b))
}

func operatorExitClass(code string) int {
	switch code {
	case "dashboard_unavailable", "provider_unavailable", "provider_missing", "provider_not_ready", "provider_executable_missing", "provider_executable_invalid", "bundled_provider_unavailable", "provider_incompatible":
		return 3
	default:
		return 1
	}
}

func operatorInput(cmd *cobra.Command) (map[string]json.RawMessage, error) {
	file, err := cmd.Flags().GetString("request-file")
	if err != nil || file == "" {
		if err != nil {
			return nil, operatorInputError(err)
		}
		return map[string]json.RawMessage{}, nil
	}
	data, err := operatorReadInput(cmd, file)
	if err != nil {
		return nil, operatorInputError(err)
	}
	if !utf8.Valid(data) {
		return nil, operatorInputError(errors.New("request JSON is not valid UTF-8"))
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	first, err := dec.Token()
	if err != nil {
		return nil, operatorInputError(fmt.Errorf("invalid request JSON: %w", err))
	}
	if d, ok := first.(json.Delim); !ok || d != '{' {
		return nil, operatorInputError(errors.New("request JSON must be one object"))
	}
	out := map[string]json.RawMessage{}
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, operatorInputError(fmt.Errorf("invalid request JSON: %w", err))
		}
		key := tok.(string)
		if _, exists := out[key]; exists {
			return nil, operatorInputError(fmt.Errorf("duplicate request field %q", key))
		}
		var value json.RawMessage
		if err := dec.Decode(&value); err != nil {
			return nil, operatorInputError(fmt.Errorf("invalid request JSON: %w", err))
		}
		out[key] = append([]byte(nil), value...)
	}
	if _, err := dec.Token(); err != nil {
		return nil, operatorInputError(errors.New("invalid request JSON: expected object end"))
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return nil, operatorInputError(errors.New("request JSON must contain one value"))
	}
	return out, nil
}

func operatorReadInput(cmd *cobra.Command, name string) ([]byte, error) {
	var r io.Reader
	if name == "-" {
		r = cmd.InOrStdin()
	} else {
		f, err := os.Open(name)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		r = f
	}
	b, err := io.ReadAll(io.LimitReader(r, operatorInputLimit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > operatorInputLimit {
		return nil, fmt.Errorf("input exceeds %d MiB limit", operatorInputLimit>>20)
	}
	return b, nil
}

func operatorText(cmd *cobra.Command) (string, bool, error) {
	textChanged := cmd.Flags().Changed("text")
	fileChanged := cmd.Flags().Changed("text-file")
	if textChanged && fileChanged {
		return "", false, operatorInputError(errors.New("use exactly one of --text or --text-file"))
	}
	if !textChanged && !fileChanged {
		return "", false, nil
	}
	if textChanged {
		value, _ := cmd.Flags().GetString("text")
		if !utf8.ValidString(value) {
			return "", false, operatorInputError(errors.New("text input is not valid UTF-8"))
		}
		if len([]byte(value)) > int(operatorInputLimit) {
			return "", false, operatorInputError(errors.New("text exceeds 1 MiB limit"))
		}
		return value, true, nil
	}
	file, err := cmd.Flags().GetString("text-file")
	if err != nil {
		return "", false, err
	}
	b, err := operatorReadInput(cmd, file)
	if err != nil {
		return "", false, operatorInputError(err)
	}
	if !utf8.Valid(b) {
		return "", false, operatorInputError(errors.New("text input is not valid UTF-8"))
	}
	return string(b), true, nil
}

func operatorMerge(body map[string]json.RawMessage, key string, value any) error {
	if _, exists := body[key]; exists {
		return operatorInputError(fmt.Errorf("duplicate request field %q", key))
	}
	data, err := json.Marshal(value)
	if err != nil {
		return operatorInputError(err)
	}
	body[key] = data
	return nil
}

func addOperatorFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("json", false, "write machine-readable JSON")
	cmd.Flags().String("request-file", "", "read request JSON from a file or - for stdin")
	cmd.Flags().Duration("timeout", 30*time.Second, "request timeout")
}

func addOperatorTextFlags(cmd *cobra.Command) {
	cmd.Flags().String("text", "", "text input")
	cmd.Flags().String("text-file", "", "read text from a file or - for stdin")
}

func operatorOutput(cmd *cobra.Command, data json.RawMessage) error {
	machine, _ := cmd.Flags().GetBool("json")
	value, err := decodeOperatorJSON(data)
	if err != nil {
		return &operatorError{Code: "invalid_response", Message: "dashboard returned invalid JSON", ExitCode: 1}
	}
	if machine {
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), string(data)); err != nil {
			return &operatorError{Code: "output_failed", Message: "could not write command output", ExitCode: 1}
		}
		if operatorMutationOutput(cmd) && operatorBulkFailed(value) {
			return &operatorError{Code: "bulk_failed", Message: "one or more operation results failed", ExitCode: 1}
		}
		return nil
	}
	if human, ok := operatorHumanOutput(cmd, value); ok {
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), human); err != nil {
			return &operatorError{Code: "output_failed", Message: "could not write command output", ExitCode: 1}
		}
		if operatorMutationOutput(cmd) && operatorBulkFailed(value) {
			return &operatorError{Code: "bulk_failed", Message: "one or more operation results failed", ExitCode: 1}
		}
		return nil
	}
	var pretty bytes.Buffer
	if json.Indent(&pretty, data, "", "  ") != nil {
		pretty.Write(data)
	}
	if pretty.Len() > 0 {
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), safeOperatorJSON(pretty.String())); err != nil {
			return &operatorError{Code: "output_failed", Message: "could not write command output", ExitCode: 1}
		}
	}
	if operatorMutationOutput(cmd) && operatorBulkFailed(value) {
		return &operatorError{Code: "bulk_failed", Message: "one or more operation results failed", ExitCode: 1}
	}
	return nil
}

func operatorRun(cmd *cobra.Command, method, path string, body any, headers http.Header) error {
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	cmd.Annotations["operator-method"] = method
	timeout, flagErr := cmd.Flags().GetDuration("timeout")
	if flagErr == nil && timeout <= 0 {
		op := &operatorError{Code: "invalid_input", Message: "--timeout must be positive", ExitCode: 2}
		return op
	}
	ctx := cmd.Context()
	if flagErr == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	c, err := operatorClientFactory()
	if err != nil {
		return err
	}
	data, err := c.Request(ctx, method, path, body, headers)
	if err != nil {
		return err
	}
	if err := operatorOutput(cmd, data); err != nil {
		return err
	}
	return nil
}

func decodeOperatorJSON(data []byte) (any, error) {
	var value any
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	if err := dec.Decode(&value); err != nil {
		return nil, err
	}
	return value, nil
}

func operatorDiagnostic(cmd *cobra.Command, err error) {
	var op *operatorError
	if !errors.As(err, &op) {
		op = &operatorError{Code: "api_error", Message: err.Error(), ExitCode: 1}
	}
	if machine, _ := cmd.Flags().GetBool("json"); machine {
		v := map[string]any{"error": map[string]any{"code": op.Code, "message": op.Message}}
		if op.HTTPStatus != 0 {
			v["http_status"] = op.HTTPStatus
		}
		data, _ := json.Marshal(v)
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), string(data))
		return
	}
	_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "error: "+safeOperatorText(op.Message))
}

func operatorExitCode(err error) int {
	var op *operatorError
	if errors.As(err, &op) && op.ExitCode != 0 {
		return op.ExitCode
	}
	return 1
}

func safeOperatorText(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r < 0x20 || (r >= 0x7f && r <= 0x9f) {
			b.WriteString(fmt.Sprintf("\\x%02x", r))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func safeOperatorJSON(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r < 0x20 && r != '\n' && r != '\t') || (r >= 0x7f && r <= 0x9f) {
			fmt.Fprintf(&b, "\\u%04x", r)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func operatorBulkFailed(value any) bool {
	if m, ok := value.(map[string]any); ok {
		for _, key := range []string{"results", "outcomes", "members"} {
			if list, ok := m[key].([]any); ok {
				for _, item := range list {
					if operatorMemberFailed(item) {
						return true
					}
				}
			}
		}
	}
	if list, ok := value.([]any); ok {
		for _, item := range list {
			if operatorMemberFailed(item) {
				return true
			}
		}
	}
	return false
}

func operatorMemberFailed(value any) bool {
	m, ok := value.(map[string]any)
	if !ok {
		return false
	}
	if failed, ok := m["ok"].(bool); ok && !failed {
		return true
	}
	if _, ok := m["error"]; ok {
		return true
	}
	if status, ok := m["status"].(string); ok && (status == "error" || status == "failed") {
		return true
	}
	return false
}
