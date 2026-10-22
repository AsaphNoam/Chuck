package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unicode/utf8"

	"github.com/AsaphNoam/Chuck/internal/attachment"
	"github.com/spf13/cobra"
)

func operatorMessage(cmd *cobra.Command, method, path string, body map[string]json.RawMessage) error {
	text, present, err := operatorText(cmd)
	if err != nil {
		return err
	}
	if present {
		if err = operatorMerge(body, "text", text); err != nil {
			return operatorInvalid(err.Error())
		}
	}
	if raw, ok := body["text"]; ok {
		if json.Unmarshal(raw, &text) != nil {
			return operatorInvalid("text must be a string")
		}
	}
	if len(text) > 64<<10 {
		return operatorInvalid("message text exceeds 64 KiB")
	}
	if !utf8.ValidString(text) {
		return operatorInvalid("message text must be valid UTF-8")
	}
	var owned []string
	if raw, ok := body["attachment_ids"]; ok {
		if err := json.Unmarshal(raw, &owned); err != nil || len(owned) > attachment.MaxFiles {
			return operatorInvalid("attachment_ids must contain at most 10 owned ids")
		}
	}
	files, _ := cmd.Flags().GetStringArray("attach")
	if len(files) > attachment.MaxFiles {
		return operatorInvalid("choose at most 10 attachments")
	}
	if _, ok := body["files"]; ok {
		return operatorInvalid("use --attach for local files")
	}
	if cmd.Name() == "send" && strings.TrimSpace(text) == "" && len(files) == 0 && len(owned) == 0 {
		return operatorInvalid("Send requires text or attachments")
	}
	if text != "" || len(owned) > 0 {
		if _, ok := body["request_id"]; !ok {
			raw, _ := json.Marshal(newPipelineRequestID())
			body["request_id"] = raw
		}
	}
	if raw, ok := body["request_id"]; ok {
		var id string
		if json.Unmarshal(raw, &id) != nil || !operatorValidReceiptID(id) {
			return operatorInvalid("request_id must be 1–128 ASCII letters, digits, - or _")
		}
	}
	if len(files) == 0 {
		return operatorRun(cmd, method, path, body, nil)
	}
	if _, ok := body["attachment_ids"]; ok {
		return operatorInvalid("--attach conflicts with attachment_ids")
	}
	if _, ok := body["request_id"]; !ok {
		raw, _ := json.Marshal(newPipelineRequestID())
		body["request_id"] = raw
	}
	var requestID string
	if json.Unmarshal(body["request_id"], &requestID) != nil || requestID == "" {
		return operatorInvalid("request_id must be a nonempty string")
	}
	opened := make([]*os.File, 0, len(files))
	defer func() {
		for _, f := range opened {
			_ = f.Close()
		}
	}()
	declarations := make([]map[string]string, 0, len(files))
	var total int64
	for i, name := range files {
		info, err := os.Stat(name)
		if err != nil || !info.Mode().IsRegular() {
			return operatorInvalid("attachment must be a readable regular file: " + safeOperatorText(name))
		}
		if info.Size() > attachment.MaxFileBytes {
			return operatorInvalid("attachment exceeds 10 MiB: " + safeOperatorText(name))
		}
		total += info.Size()
		if total > attachment.MaxTotalBytes {
			return operatorInvalid("attachments exceed 25 MiB total")
		}
		fd, err := syscall.Open(name, syscall.O_RDONLY|syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0)
		if err != nil {
			return operatorInvalid("cannot open attachment: " + safeOperatorText(name))
		}
		f := os.NewFile(uintptr(fd), name)
		opened = append(opened, f)
		actual, err := f.Stat()
		if err != nil || !actual.Mode().IsRegular() || !os.SameFile(info, actual) {
			return operatorInvalid("attachment changed while opening: " + safeOperatorText(name))
		}
		declarations = append(declarations, map[string]string{"part": fmt.Sprintf("file%d", i), "name": filepath.Base(name)})
	}
	raw, _ := json.Marshal(declarations)
	body["files"] = raw
	metadata, err := json.Marshal(body)
	if err != nil || len(metadata) > 64<<10 {
		return operatorInvalid("message metadata exceeds 64 KiB")
	}
	ctx, cancel, err := operatorContext(cmd)
	if err != nil {
		return err
	}
	defer cancel()
	client, err := operatorClientFactory()
	if err != nil {
		return err
	}
	reader, writer := io.Pipe()
	defer reader.Close()
	multi := multipart.NewWriter(writer)
	done := make(chan error, 1)
	go func() {
		writeErr := func() error {
			part, err := multi.CreateFormField("message")
			if err != nil {
				return err
			}
			if _, err = part.Write(metadata); err != nil {
				return err
			}
			var copiedTotal int64
			for i, f := range opened {
				part, err := multi.CreateFormFile(declarations[i]["part"], declarations[i]["name"])
				if err != nil {
					return err
				}
				remaining := int64(attachment.MaxTotalBytes) - copiedTotal
				limit := int64(attachment.MaxFileBytes)
				if remaining < limit {
					limit = remaining
				}
				n, err := io.Copy(part, io.LimitReader(f, limit+1))
				if err != nil {
					return err
				}
				if n > limit {
					return operatorInvalid("attachment grew beyond the file or total byte limit")
				}
				copiedTotal += n
			}
			return multi.Close()
		}()
		_ = writer.CloseWithError(writeErr)
		done <- writeErr
	}()
	data, requestErr := client.requestReader(ctx, method, path, reader, http.Header{"Content-Type": []string{multi.FormDataContentType()}})
	_ = reader.Close()
	writeErr := <-done
	if requestErr != nil {
		var op *operatorError
		if errors.As(requestErr, &op) && op.Code == "result_unknown" {
			op.Message += "; request id " + safeOperatorText(requestID)
		}
		return requestErr
	}
	if writeErr != nil {
		return &operatorError{Code: "result_unknown", Message: "attachment input failed during submission; inspect held/history before retrying; request id " + safeOperatorText(requestID), ExitCode: 1}
	}
	return operatorOutput(cmd, data)
}

func operatorValidReceiptID(id string) bool {
	if len(id) < 1 || len(id) > 128 {
		return false
	}
	for _, r := range id {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_') {
			return false
		}
	}
	return true
}

func newAgentDownloadCmd(image bool) *cobra.Command {
	use := "attachment <id> <attachment_id>"
	nargs := 2
	if image {
		use = "image <id>"
		nargs = 1
	}
	c := &cobra.Command{Use: use, Short: "Save explicitly selected image or attachment bytes", Args: cobra.ExactArgs(nargs)}
	addOperatorFlags(c)
	c.Flags().String("output", "", "destination file; never writes bytes to the terminal")
	c.Flags().Bool("overwrite", false, "replace an existing destination")
	if image {
		c.Flags().String("path", "", "explicit local image path")
	}
	c.Example = "chuck agent attachment AGENT_ID ATTACHMENT_ID --output copy.png --json"
	if image {
		c.Example = "chuck agent image AGENT_ID --path image.png --output copy.png"
	}
	c.Long = "Download at most 10 MiB to an owner-only temporary file, then publish the complete file. Existing destinations require --overwrite. Metadata reads do not fetch bytes."
	c.RunE = func(c *cobra.Command, args []string) error {
		if request, _ := c.Flags().GetString("request-file"); request != "" {
			return operatorInvalid("binary reads accept only explicit target and output flags")
		}
		output, _ := c.Flags().GetString("output")
		if output == "" {
			return operatorInvalid("--output is required")
		}
		overwrite, _ := c.Flags().GetBool("overwrite")
		if !overwrite {
			if _, err := os.Lstat(output); err == nil {
				return operatorInvalid("destination exists; use --overwrite")
			} else if !os.IsNotExist(err) {
				return operatorInvalid("cannot inspect destination")
			}
		}
		path := "/api/sessions/" + url.PathEscape(args[0]) + "/attachments/"
		if image {
			source, _ := c.Flags().GetString("path")
			if source == "" {
				return operatorInvalid("--path is required")
			}
			path = "/api/sessions/" + url.PathEscape(args[0]) + "/image?" + url.Values{"path": []string{source}}.Encode()
		} else {
			path += url.PathEscape(args[1])
		}
		ctx, cancel, err := operatorContext(c)
		if err != nil {
			return err
		}
		defer cancel()
		client, err := operatorClientFactory()
		if err != nil {
			return err
		}
		req, err := http.NewRequestWithContext(ctx, "GET", client.BaseURL+path, nil)
		if err != nil {
			return operatorInvalid("invalid download target")
		}
		resp, err := client.httpClient().Do(req)
		if err != nil {
			return &operatorError{Code: "dashboard_unavailable", Message: "cannot reach dashboard; run `chuck dashboard start`", ExitCode: 3}
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			data, _ := readOperatorBounded(resp.Body, 1<<20)
			return operatorHTTPError(resp.StatusCode, data)
		}
		temp, err := os.CreateTemp(filepath.Dir(output), ".chuck-download-*")
		if err != nil {
			return operatorInvalid("cannot create destination file")
		}
		defer os.Remove(temp.Name())
		defer temp.Close()
		n, err := io.Copy(temp, io.LimitReader(resp.Body, attachment.MaxFileBytes+1))
		if err != nil {
			return &operatorError{Code: "download_failed", Message: "download interrupted; destination was not published", ExitCode: 1}
		}
		if n > attachment.MaxFileBytes {
			return &operatorError{Code: "response_too_large", Message: "download exceeds 10 MiB", ExitCode: 1}
		}
		if resp.ContentLength >= 0 && n != resp.ContentLength {
			return &operatorError{Code: "download_failed", Message: "incomplete download", ExitCode: 1}
		}
		if err = temp.Sync(); err != nil {
			return err
		}
		if err = temp.Close(); err != nil {
			return err
		}
		if overwrite {
			err = os.Rename(temp.Name(), output)
		} else {
			err = os.Link(temp.Name(), output)
		}
		if err != nil {
			return operatorInvalid("cannot publish destination; it may already exist")
		}
		parent, err := os.Open(filepath.Dir(output))
		if err != nil {
			return err
		}
		syncErr := parent.Sync()
		_ = parent.Close()
		if syncErr != nil {
			return syncErr
		}
		if machine, _ := c.Flags().GetBool("json"); machine {
			return operatorOutput(c, json.RawMessage(`{}`))
		}
		_, err = fmt.Fprintln(c.OutOrStdout(), "Saved "+safeOperatorText(output))
		return err
	}
	return c
}
