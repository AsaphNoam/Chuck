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
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func putViewerFile(t *testing.T, h http.Handler, route, path, revision, content string) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(map[string]string{"path": path, "base_revision": revision, "content": content})
	if err != nil {
		t.Fatal(err)
	}
	r := newLocalRequest(http.MethodPut, route, bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

// The registered HTTP route, exact disk bytes and independent stat metadata are
// the oracle: a helper-only result would miss missing route wiring or lossy saves.
func TestFileViewerConditionalSaveAndProbe(t *testing.T) {
	s := testServer(t, false)
	dir := seedReadableWorkspace(t, s, "viewer_save")
	h := s.routes()
	path := filepath.Join(dir, "main.go")
	if err := os.Chmod(path, 0640); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	old := readFileOK(t, h, "viewer_save", "main.go")
	if !old.Editable || len(old.Revision) != 64 {
		t.Fatalf("not editable: %+v", old)
	}
	probe := doGET(t, h, "/api/sessions/viewer_save/file?path=main.go&probe=1")
	var p map[string]any
	if err := json.Unmarshal(probe.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	if probe.Code != 200 || len(p) != 2 || p["path"] != "main.go" || p["observed_revision"] != old.ObservedRevision {
		t.Fatalf("probe: %d %s", probe.Code, probe.Body.String())
	}
	want := "\ufefffirst\r\nsecond\r\nλ"
	w := putViewerFile(t, h, "/api/sessions/viewer_save/file", "main.go", old.Revision, want)
	if w.Code != 200 {
		t.Fatalf("save: %d %s", w.Code, w.Body.String())
	}
	body, err := os.ReadFile(path)
	if err != nil || string(body) != want {
		t.Fatalf("disk bytes %q err=%v", body, err)
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if os.SameFile(before, after) || after.Mode() != before.Mode() {
		t.Fatalf("replacement/mode: before=%v after=%v", before, after)
	}
	var saved fileContent
	if err := json.Unmarshal(w.Body.Bytes(), &saved); err != nil {
		t.Fatal(err)
	}
	read := readFileOK(t, h, "viewer_save", "main.go")
	if saved.Content != want || saved.Revision != read.Revision || saved.ObservedRevision != read.ObservedRevision {
		t.Fatalf("save snapshot differs from disk: saved=%+v read=%+v", saved, read)
	}
	w = putViewerFile(t, h, "/api/sessions/viewer_save/file", "main.go", old.Revision, "stale")
	if w.Code != 409 || apiErrorCode(t, w.Body.Bytes()) != "file_conflict" {
		t.Fatalf("stale save: %d %s", w.Code, w.Body.String())
	}
	if b, _ := os.ReadFile(path); string(b) != want {
		t.Fatalf("conflict changed disk: %q", b)
	}
	entries, _ := filepath.Glob(filepath.Join(dir, ".chuck-save-*"))
	if len(entries) != 0 {
		t.Fatalf("leaked temporary files: %v", entries)
	}
}

func TestFileViewerUnsafeTargetsAndPreparedRaces(t *testing.T) {
	for _, kind := range []string{"symlink", "hardlink", "readonly", "fifo", "xattr", "target_swap", "parent_swap", "external_write"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "source.txt")
			writeFile(t, path, "original\r\n")
			original, ae := readLocalFile(path, "source.txt")
			if ae != nil {
				t.Fatal(ae)
			}
			var prepared func()
			switch kind {
			case "symlink":
				if err := os.Rename(path, path+".real"); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(path+".real", path); err != nil {
					t.Fatal(err)
				}
			case "hardlink":
				if err := os.Link(path, path+".link"); err != nil {
					t.Fatal(err)
				}
			case "readonly":
				if err := os.Chmod(path, 0400); err != nil {
					t.Fatal(err)
				}
			case "fifo":
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err := unix.Mkfifo(path, 0600); err != nil {
					t.Fatal(err)
				}
			case "xattr":
				if err := unix.Setxattr(path, "user.chuck_test", []byte("retained"), 0); err != nil {
					t.Skipf("filesystem cannot create xattr: %v", err)
				}
			case "target_swap":
				prepared = func() {
					if err := os.Rename(path, path+".real"); err != nil {
						t.Fatal(err)
					}
					if err := os.Symlink(path+".real", path); err != nil {
						t.Fatal(err)
					}
				}
			case "parent_swap":
				prepared = func() {
					if err := os.Rename(dir, dir+".moved"); err != nil {
						t.Fatal(err)
					}
					t.Cleanup(func() { os.RemoveAll(dir + ".moved") })
					if err := os.Mkdir(dir, 0700); err != nil {
						t.Fatal(err)
					}
					writeFile(t, path, "replacement parent")
				}
			case "external_write":
				prepared = func() { writeFile(t, path, "external writer") }
			}
			_, ae = saveLocalFile(path, "source.txt", original.Revision, []byte("operator"), prepared)
			if ae == nil {
				t.Fatal("unsafe save succeeded")
			}
			if kind == "target_swap" || kind == "parent_swap" || kind == "external_write" {
				if ae.Code != "file_conflict" {
					t.Fatalf("race code: %v", ae)
				}
			} else if ae.Code != "file_not_editable" {
				t.Fatalf("refusal: %v", ae)
			}
			if kind != "fifo" {
				b, err := os.ReadFile(path)
				if err != nil || string(b) == "operator" {
					t.Fatalf("unsafe mutation %q err=%v", b, err)
				}
			}
			entries, _ := filepath.Glob(filepath.Join(dir, ".chuck-save-*"))
			movedEntries, _ := filepath.Glob(filepath.Join(dir+".moved", ".chuck-save-*"))
			if len(entries)+len(movedEntries) != 0 {
				t.Fatalf("temporary file leaked: %v %v", entries, movedEntries)
			}
		})
	}
}

func TestFileViewerConcurrentSavesHaveOneWinner(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "source.txt")
	writeFile(t, path, "original")
	old, ae := readLocalFile(path, "source.txt")
	if ae != nil {
		t.Fatal(ae)
	}
	start := make(chan struct{})
	results := make(chan string, 2)
	var wg sync.WaitGroup
	for _, text := range []string{"first", "second"} {
		wg.Add(1)
		go func(text string) {
			defer wg.Done()
			<-start
			_, err := saveLocalFile(path, "source.txt", old.Revision, []byte(text), nil)
			if err == nil {
				results <- "saved"
			} else {
				results <- err.Code
			}
		}(text)
	}
	close(start)
	wg.Wait()
	close(results)
	counts := map[string]int{}
	for result := range results {
		counts[result]++
	}
	if counts["saved"] != 1 || counts["file_conflict"] != 1 {
		t.Fatalf("results: %v", counts)
	}
	fileSaveLocks.Lock()
	retained := len(fileSaveLocks.entries)
	fileSaveLocks.Unlock()
	if retained != 0 {
		t.Fatalf("retained %d save locks", retained)
	}
}

func TestFileViewerCaseAliasConcurrentSavesHaveOneWinner(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "source.txt")
	alias := filepath.Join(dir, "SOURCE.TXT")
	writeFile(t, path, "original")
	first, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	second, err := os.Stat(alias)
	if err != nil || !os.SameFile(first, second) {
		t.Skip("filesystem does not resolve these names to the same file")
	}
	old, ae := readLocalFile(path, "source.txt")
	if ae != nil {
		t.Fatal(ae)
	}
	ready := make(chan struct{}, 2)
	release := make(chan struct{})
	results := make(chan string, 2)
	for i, target := range []string{path, alias} {
		go func(target, content string) {
			_, saveErr := saveLocalFile(target, target, old.Revision, []byte(content), func() {
				ready <- struct{}{}
				<-release
			})
			if saveErr == nil {
				results <- "saved"
			} else {
				results <- saveErr.Code
			}
		}(target, []string{"first", "second"}[i])
	}
	<-ready
	// An incorrect lock permits both saves to reach the prepared boundary.
	overlapped := false
	select {
	case <-ready:
		overlapped = true
	case <-time.After(200 * time.Millisecond):
	}
	close(release)
	counts := map[string]int{<-results: 1}
	counts[<-results]++
	if overlapped {
		t.Fatal("case aliases entered save preparation concurrently")
	}
	if counts["saved"] != 1 || counts["file_conflict"] != 1 {
		t.Fatalf("results: %v", counts)
	}
	content, err := os.ReadFile(path)
	if err != nil || (string(content) != "first" && string(content) != "second") {
		t.Fatalf("saved content = %q, error = %v", content, err)
	}
}

func TestFileViewerUnstableReadAndUncertainSave(t *testing.T) {
	path := filepath.Join(t.TempDir(), "source.txt")
	writeFile(t, path, "original")
	_, ae := readLocalFileSnapshot(path, "source.txt", nil, false, false, func() { writeFile(t, path, "external change") })
	if ae == nil || ae.Code != "file_changing" || ae.HTTPStatus() != 409 {
		t.Fatalf("unstable read: %v", ae)
	}
	old, ae := readLocalFile(path, "source.txt")
	if ae != nil {
		t.Fatal(ae)
	}
	_, ae = saveLocalFileWithHooks(path, "source.txt", old.Revision, []byte("submitted"), nil, func(int) error { return errors.New("injected directory sync failure") })
	if ae == nil || ae.Code != "file_save_uncertain" || ae.HTTPStatus() != 409 {
		t.Fatalf("uncertain save: %v", ae)
	}
	if b, _ := os.ReadFile(path); string(b) != "submitted" {
		t.Fatalf("uncertain response did not preserve actual replacement: %q", b)
	}
}

func TestFileViewerSaveValidationAndLocalGuard(t *testing.T) {
	s := testServer(t, false)
	dir := seedReadableWorkspace(t, s, "viewer_validate")
	h := s.routes()
	old := readFileOK(t, h, "viewer_validate", "main.go")
	valid, _ := json.Marshal(map[string]string{"path": "main.go", "base_revision": old.Revision, "content": "updated"})
	for _, body := range []string{`{}`, `{"path":"main.go","base_revision":"bad","content":"oops"}`, string(valid[:len(valid)-1]) + `,"force":true}`, string(valid) + `{}`, `{"path":"main.go","base_revision":"` + old.Revision + `","content":"` + strings.Repeat("a", fileReadLimit+1) + `"}`} {
		r := newLocalRequest(http.MethodPut, "/api/sessions/viewer_validate/file", strings.NewReader(body))
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 422 {
			t.Fatalf("invalid body: %d %s", w.Code, w.Body.String())
		}
	}
	for _, guard := range []string{"host", "origin"} {
		r := newLocalRequest(http.MethodPut, "/api/sessions/viewer_validate/file", bytes.NewReader(valid))
		if guard == "host" {
			r.Host = "attacker.example"
		} else {
			r.Header.Set("Origin", "https://attacker.example")
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 403 {
			t.Fatalf("%s guard status %d", guard, w.Code)
		}
	}
	b, _ := os.ReadFile(filepath.Join(dir, "main.go"))
	if string(b) != old.Content {
		t.Fatalf("rejected request mutated disk: %q", b)
	}
	// Probe errors must still use the ordinary typed read envelope.
	w := doGET(t, h, "/api/sessions/viewer_validate/file?probe=1&path="+url.QueryEscape("missing.txt"))
	if w.Code != 404 {
		t.Fatalf("missing probe: %d %s", w.Code, w.Body.String())
	}
}

func TestFileViewerRetainedRoomSaveWithoutSession(t *testing.T) {
	s, h := roomRESTServer(t)
	var room thinkTankDetailWire
	if err := json.Unmarshal(doJSON(t, h, http.MethodPost, "/api/think-tanks", roomBody("a_one", "a_two")).Body.Bytes(), &room); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "notes.md"), "retained workspace")
	beginCapturedTurn(t, s, room.RoomID, "a_one", "file_edit", dir)
	base := "/api/think-tanks/" + room.RoomID
	var sources struct {
		Sources []thinkTankSourceWire `json:"sources"`
	}
	if err := json.Unmarshal(doJSON(t, h, http.MethodGet, base+"/files", "").Body.Bytes(), &sources); err != nil || len(sources.Sources) != 1 {
		t.Fatalf("sources: %+v err=%v", sources, err)
	}
	if _, err := s.stateStore.DB().Exec(`DELETE FROM sessions WHERE agent_id = 'a_one'`); err != nil {
		t.Fatal(err)
	}
	if err := s.stateStore.DeleteAgent("a_one"); err != nil {
		t.Fatal(err)
	}
	route := base + "/sources/" + sources.Sources[0].SourceID + "/file"
	get := doJSON(t, h, http.MethodGet, route+"?path=notes.md", "")
	var old fileContent
	if err := json.Unmarshal(get.Body.Bytes(), &old); err != nil || get.Code != 200 || !old.Editable {
		t.Fatalf("retained read: %d %s err=%v", get.Code, get.Body.String(), err)
	}
	probe := doJSON(t, h, http.MethodGet, route+"?path=notes.md&probe=1", "")
	if probe.Code != 200 || strings.Contains(probe.Body.String(), "content") {
		t.Fatalf("retained probe: %d %s", probe.Code, probe.Body.String())
	}
	w := putViewerFile(t, h, route, "notes.md", old.Revision, "room edit\r\n")
	if w.Code != 200 {
		t.Fatalf("retained save: %d %s", w.Code, w.Body.String())
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "notes.md")); string(b) != "room edit\r\n" {
		t.Fatalf("room disk bytes: %q", b)
	}
	w = putViewerFile(t, h, base+"/sources/not-a-member/file", "notes.md", old.Revision, "invalid member")
	if w.Code != 404 {
		t.Fatalf("unknown room source: %d %s", w.Code, w.Body.String())
	}
}

func TestFileViewerPhoneProbesFilteredAndWritesDenied(t *testing.T) {
	s := testServer(t, true)
	dir := seedReadableWorkspace(t, s, "viewer_phone")
	if _, err := s.stateStore.DB().Exec(`INSERT INTO tracked_files(agent_id, path, abs_path, first_seq, last_seq, first_ts, last_ts) VALUES (?, ?, ?, 1, 1, ?, ?)`, "viewer_phone", "main.go", filepath.Join(dir, "main.go"), time.Now().UTC().Format(time.RFC3339), time.Now().UTC().Format(time.RFC3339)); err != nil {
		t.Fatal(err)
	}
	h := s.remoteRoutes(testDomain, testWhoIs(map[string]string{"100.64.0.2:5000": "node"}))
	token := pairTestDevice(t, s, "viewer-phone", "node")
	request := func(method, route, body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, phoneRequest(method, route, body, token))
		return w
	}
	for _, path := range []string{"main.go", "notes.md"} {
		w := request(http.MethodGet, "/api/sessions/viewer_phone/file?probe=1&path="+path, "")
		want := 404
		if path == "main.go" {
			want = 200
		}
		if w.Code != want || strings.Contains(w.Body.String(), "content") || strings.Contains(w.Body.String(), dir) {
			t.Fatalf("phone probe: %d %s", w.Code, w.Body.String())
		}
	}
	read := request(http.MethodGet, "/api/sessions/viewer_phone/file?path=main.go", "")
	var old fileContent
	if err := json.Unmarshal(read.Body.Bytes(), &old); err != nil || old.Editable || old.EditRefusal != "remote_read_only" {
		t.Fatalf("phone read: %s err=%v", read.Body.String(), err)
	}
	for _, route := range []string{"/api/sessions/viewer_phone/file", "/api/think-tanks/room/sources/source/file"} {
		body, _ := json.Marshal(map[string]string{"path": "main.go", "base_revision": old.Revision, "content": "phone write"})
		w := request(http.MethodPut, route, string(body))
		if w.Code != 404 || errorCode(t, w) != codeRemoteRouteNotAvailable {
			t.Fatalf("phone PUT: %d %s", w.Code, w.Body.String())
		}
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "main.go")); string(b) != old.Content {
		t.Fatalf("phone wrote disk: %q", b)
	}
	if err := os.Remove(filepath.Join(dir, "main.go")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(dir, "notes.md"), filepath.Join(dir, "main.go")); err != nil {
		t.Fatal(err)
	}
	w := request(http.MethodGet, "/api/sessions/viewer_phone/file?probe=1&path=main.go", "")
	if w.Code != 404 || errorCode(t, w) != codeRemoteFileNotTracked {
		t.Fatalf("phone symlink probe: %d %s", w.Code, w.Body.String())
	}
}
