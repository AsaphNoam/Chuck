package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

// Exercise the real route and local/remote guards. The native launch seam is
// observed directly: invalid inputs must never reach it, and success must pass
// the original normalized file path rather than a served copy or a URL.
func TestFileOpenBrowser(t *testing.T) {
	s := testServer(t, false)
	root := seedReadableWorkspace(t, s, "a_browser")
	writeFile(t, filepath.Join(root, "preview.HTML"), "<h1>Preview</h1>")
	writeFile(t, filepath.Join(root, "locked.html"), "private")
	if err := os.Chmod(filepath.Join(root, "locked.html"), 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(root, "locked.html"), 0600) })
	var opened []string
	var launchErr error
	s.openBrowserFile = func(path string) error {
		opened = append(opened, path)
		return launchErr
	}
	h := s.routes()
	request := func(path, origin string) *httptest.ResponseRecorder {
		body, _ := json.Marshal(map[string]string{"path": path})
		r := newLocalRequest(http.MethodPost, "/api/sessions/a_browser/file/open-browser", bytes.NewReader(body))
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		out := httptest.NewRecorder()
		h.ServeHTTP(out, r)
		return out
	}
	if rec := request("internal/../preview.HTML", ""); rec.Code != 200 {
		t.Fatalf("open: %d %s", rec.Code, rec.Body)
	}
	if len(opened) != 1 || opened[0] != filepath.Join(root, "preview.HTML") {
		t.Fatalf("opened = %v", opened)
	}
	for _, tc := range []struct {
		path   string
		status int
	}{
		{"missing.html", 404}, {"main.go", 422}, {"", 422},
		{"https://example.com/preview.html", 404},
	} {
		if rec := request(tc.path, ""); rec.Code != tc.status {
			t.Fatalf("%q: %d %s", tc.path, rec.Code, rec.Body)
		}
	}
	if os.Geteuid() != 0 {
		if rec := request("locked.html", ""); rec.Code != 422 {
			t.Fatalf("unreadable: %d %s", rec.Code, rec.Body)
		}
	}
	if rec := request("preview.HTML", "https://evil.example"); rec.Code != 403 {
		t.Fatalf("cross origin: %d %s", rec.Code, rec.Body)
	}
	if len(opened) != 1 {
		t.Fatalf("invalid requests launched files: %v", opened)
	}
	launchErr = errors.New("launch refused")
	if rec := request("preview.HTML", ""); rec.Code != 500 || !bytes.Contains(rec.Body.Bytes(), []byte("launch refused")) {
		t.Fatalf("launch error: %d %s", rec.Code, rec.Body)
	}
}

func TestHTMLFileURLPreservesFilename(t *testing.T) {
	path := "/tmp/preview #?%.html"
	got := htmlFileURL(path)
	if got != "file:///tmp/preview%20%23%3F%25.html" {
		t.Fatalf("file URL = %q", got)
	}
	parsed, err := url.Parse(got)
	if err != nil || parsed.Path != path || parsed.RawQuery != "" || parsed.Fragment != "" {
		t.Fatalf("parsed URL = %+v, %v", parsed, err)
	}
}

func TestFileOpenBrowserExcludedFromRemote(t *testing.T) {
	s := testServer(t, true)
	h := s.remoteRoutes(testDomain, testWhoIs(map[string]string{"100.64.0.2:5000": "n"}))
	token := pairTestDevice(t, s, "d1", "n")
	s.openBrowserFile = func(string) error { t.Fatal("remote request launched browser"); return nil }
	for _, path := range []string{
		"/api/sessions/a/file/open-browser",
		"/api/think-tanks/room/sources/source/file/open-browser",
	} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, phoneRequest(http.MethodPost, path, `{"path":"preview.html"}`, token))
		if rec.Code != 404 {
			t.Fatalf("remote %s: %d %s", path, rec.Code, rec.Body)
		}
	}
}

func TestFileOpenBrowserRetainedRoomSource(t *testing.T) {
	s, h := roomRESTServer(t)
	var room thinkTankDetailWire
	if err := json.Unmarshal(doJSON(t, h, http.MethodPost, "/api/think-tanks", roomBody("a_one", "a_two")).Body.Bytes(), &room); err != nil {
		t.Fatal(err)
	}
	cwd, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(cwd, "preview.htm"), "<p>retained</p>")
	beginCapturedTurn(t, s, room.RoomID, "a_one", "t1", cwd)
	sources, err := s.stateStore.ThinkTankSources(room.RoomID)
	if err != nil || len(sources) != 1 {
		t.Fatalf("sources: %v %v", sources, err)
	}
	if _, err := s.stateStore.DB().Exec(`DELETE FROM sessions WHERE agent_id = 'a_one'`); err != nil {
		t.Fatal(err)
	}
	var opened string
	s.openBrowserFile = func(path string) error { opened = path; return nil }
	base := "/api/think-tanks/" + room.RoomID + "/sources/"
	if rec := doJSON(t, h, http.MethodPost, base+sources[0].SourceID+"/file/open-browser", `{"path":"preview.htm"}`); rec.Code != 200 {
		t.Fatalf("retained source: %d %s", rec.Code, rec.Body)
	}
	if opened != filepath.Join(cwd, "preview.htm") {
		t.Fatalf("opened = %q", opened)
	}
	if rec := doJSON(t, h, http.MethodPost, base+"missing/file/open-browser", `{"path":"preview.htm"}`); rec.Code != 404 {
		t.Fatalf("missing source: %d %s", rec.Code, rec.Body)
	}
}
