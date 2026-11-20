package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func postReveal(t *testing.T, h http.Handler, target, path string) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"path": path})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, newLocalRequest(http.MethodPost, target, bytes.NewReader(body)))
	return rec
}

// The reveal command is a fixed executable with separate arguments, so a
// hostile filename is never parsed as an option or by a shell (TS-05.R31).
func TestRevealCommandArguments(t *testing.T) {
	var argv []string
	path := "/tmp/-n ; rm -rf x.txt"
	err := revealWithRunner(context.Background(), path, func(cmd *exec.Cmd) ([]byte, error) {
		argv = cmd.Args
		return nil, nil
	})
	if err != nil || strings.Join(argv, "|") != "/usr/bin/open|-R|--|"+path {
		t.Fatalf("argv = %q, err = %v", argv, err)
	}
	if err := revealWithRunner(context.Background(), path, func(*exec.Cmd) ([]byte, error) { return nil, errors.New("boom") }); err == nil {
		t.Fatal("runner failure was not reported")
	}
}

func TestProjectResourceReveal(t *testing.T) {
	s, h, root := projectFilesServer(t)
	var revealed []string
	var revealErr error
	s.revealFile = func(path string) error {
		revealed = append(revealed, path)
		return revealErr
	}
	target := "/api/projects/p1/resource-file/reveal-in-finder"
	if rec := postReveal(t, h, target, "docs/deep/data.CSV"); rec.Code != http.StatusNoContent {
		t.Fatalf("reveal: %d %s", rec.Code, rec.Body)
	}
	if len(revealed) != 1 || revealed[0] != filepath.Join(root, "docs", "deep", "data.CSV") {
		t.Fatalf("revealed = %v", revealed)
	}
	if err := os.Remove(filepath.Join(root, "Notes.md")); err != nil {
		t.Fatal(err)
	}
	for path, code := range map[string]string{
		"Notes.md":           "not_found",
		"docs":               "not_a_file",
		"to-other/secret.md": "not_a_file",
		"../p2/secret.md":    "path_refused",
	} {
		if rec := postReveal(t, h, target, path); !strings.Contains(rec.Body.String(), `"`+code+`"`) {
			t.Fatalf("%q: %d %s, want %s", path, rec.Code, rec.Body, code)
		}
	}
	revealErr = errRevealUnsupported
	if rec := postReveal(t, h, target, "docs/notes.md"); rec.Code != http.StatusNotImplemented {
		t.Fatalf("unsupported host: %d %s", rec.Code, rec.Body)
	}
	revealErr = errors.New("finder failed")
	if rec := postReveal(t, h, target, "docs/notes.md"); rec.Code != http.StatusInternalServerError {
		t.Fatalf("failed reveal: %d %s", rec.Code, rec.Body)
	}
	if len(revealed) != 3 {
		t.Fatalf("refused paths reached Finder: %v", revealed)
	}
}

func TestChatFileReveal(t *testing.T) {
	s := testServer(t, false)
	root := seedReadableWorkspace(t, s, "a_reveal")
	writeFile(t, filepath.Join(root, "report.csv"), "a,b\n")
	var revealed []string
	s.revealFile = func(path string) error {
		revealed = append(revealed, path)
		return nil
	}
	h := s.routes()
	target := "/api/sessions/a_reveal/file/reveal-in-finder"
	if rec := postReveal(t, h, target, "report.csv"); rec.Code != http.StatusNoContent {
		t.Fatalf("reveal: %d %s", rec.Code, rec.Body)
	}
	if len(revealed) != 1 || revealed[0] != filepath.Join(root, "report.csv") {
		t.Fatalf("revealed = %v", revealed)
	}
	for path, status := range map[string]int{"missing.csv": 404, ".": 422, "": 422} {
		if rec := postReveal(t, h, target, path); rec.Code != status {
			t.Fatalf("%q: %d %s", path, rec.Code, rec.Body)
		}
	}
	if rec := postReveal(t, h, "/api/sessions/nobody/file/reveal-in-finder", "report.csv"); rec.Code != 404 {
		t.Fatalf("unknown agent: %d", rec.Code)
	}
	if len(revealed) != 1 {
		t.Fatalf("refused paths reached Finder: %v", revealed)
	}
}

func TestRevealRoutesDeniedRemotely(t *testing.T) {
	s, _, _ := projectFilesServer(t)
	s.revealFile = func(string) error { t.Fatal("remote request revealed a file"); return nil }
	remote := s.remoteRoutes(testDomain, testWhoIs(map[string]string{"100.64.0.2:5000": "n"}))
	token := pairTestDevice(t, s, "d1", "n")
	for _, path := range []string{"/api/projects/p1/resource-file/reveal-in-finder", "/api/sessions/a/file/reveal-in-finder"} {
		rec := httptest.NewRecorder()
		remote.ServeHTTP(rec, phoneRequest(http.MethodPost, path, `{"path":"Notes.md"}`, token))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("remote %s: %d %s", path, rec.Code, rec.Body)
		}
	}
}
