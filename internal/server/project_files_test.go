package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/config"
)

// projectFilesServer returns a server with project "p1" whose shared root holds
// files with controlled times, plus a second project "p2" and a checkout file
// that must never appear in p1's browser (FS-11.A6/A9).
func projectFilesServer(t *testing.T) (*Server, http.Handler, string) {
	t.Helper()
	srv := testServer(t, true)
	checkout := t.TempDir()
	if err := os.WriteFile(filepath.Join(checkout, "checkout.md"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"p1", "p2"} {
		if err := srv.configStore.WriteProject(id, config.Project{Title: id, Cwd: checkout}); err != nil {
			t.Fatal(err)
		}
	}
	root, err := srv.configStore.EnsureProjectResources("p1")
	if err != nil {
		t.Fatal(err)
	}
	other, err := srv.configStore.EnsureProjectResources("p2")
	if err != nil {
		t.Fatal(err)
	}
	writeAt := func(base, rel, body string, at time.Time) {
		t.Helper()
		p := filepath.Join(base, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(p, at, at); err != nil {
			t.Fatal(err)
		}
	}
	base := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	writeAt(root, "Notes.md", "# notes\n", base)
	writeAt(root, "docs/notes.md", "# nested\n", base.Add(2*time.Hour))
	writeAt(root, "docs/deep/data.CSV", "a,b\n1,2\n", base.Add(time.Hour))
	writeAt(root, ".hidden", "h", base.Add(-time.Hour))
	writeAt(root, "docs/same.ts", "1", base.Add(3*time.Hour))
	writeAt(root, "zz/same.ts", "2", base.Add(3*time.Hour))
	writeAt(other, "secret.md", "other project", base)
	if err := os.Mkdir(filepath.Join(root, "empty"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(other, filepath.Join(root, "to-other")); err != nil {
		t.Fatal(err)
	}
	return srv, srv.routes(), root
}

func getCatalog(t *testing.T, h http.Handler, target string) (resourceCatalog, *httptest.ResponseRecorder) {
	t.Helper()
	rec := doGET(t, h, target)
	var out resourceCatalog
	if rec.Code == http.StatusOK {
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
	}
	return out, rec
}

func entryPaths(c resourceCatalog) []string {
	var out []string
	for _, e := range c.Entries {
		out = append(out, e.Path+":"+e.Kind)
	}
	return out
}

func TestProjectResourceCatalogOrderAndScope(t *testing.T) {
	_, h, _ := projectFilesServer(t)
	cat, rec := getCatalog(t, h, "/api/projects/p1/resource-files")
	if rec.Code != http.StatusOK {
		t.Fatalf("catalog: %d %s", rec.Code, rec.Body)
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("cache-control = %q", rec.Header().Get("Cache-Control"))
	}
	if cat.Partial {
		t.Fatalf("unexpected partial: %+v", cat)
	}
	var files []string
	kinds := map[string]string{}
	for _, e := range cat.Entries {
		kinds[e.Path] = e.Kind
		if e.Kind == "file" {
			files = append(files, e.Path)
		}
	}
	want := []string{"docs/same.ts", "zz/same.ts", "docs/notes.md", "docs/deep/data.CSV", "Notes.md", ".hidden"}
	if strings.Join(files, ",") != strings.Join(want, ",") {
		t.Fatalf("file order = %v, want %v", files, want)
	}
	for path, kind := range map[string]string{"empty": "directory", "docs": "directory", "docs/deep": "directory", "to-other": "symlink"} {
		if kinds[path] != kind {
			t.Fatalf("%s kind = %q, want %q (all: %v)", path, kinds[path], kind, entryPaths(cat))
		}
	}
	body := rec.Body.String()
	for _, leak := range []string{"secret.md", "checkout.md"} {
		if strings.Contains(body, leak) {
			t.Fatalf("catalog leaked %s: %s", leak, body)
		}
	}

	scoped, rec := getCatalog(t, h, "/api/projects/p1/resource-files?folder=docs")
	if rec.Code != http.StatusOK {
		t.Fatalf("scoped: %d %s", rec.Code, rec.Body)
	}
	for _, e := range scoped.Entries {
		if !strings.HasPrefix(e.Path, "docs/") {
			t.Fatalf("scoped catalog escaped folder: %v", entryPaths(scoped))
		}
	}
	if len(scoped.Entries) != 4 {
		t.Fatalf("scoped entries = %v", entryPaths(scoped))
	}
}

// Times within one second order at full precision; an exact tie falls back to
// name, then path (FS-11.R12).
func TestProjectResourceCatalogSubsecondOrder(t *testing.T) {
	_, h, root := projectFilesServer(t)
	second := time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC)
	for rel, at := range map[string]time.Time{
		"sub/a-older.md":   second.Add(100 * time.Millisecond),
		"sub/b-newer.md":   second.Add(900 * time.Millisecond),
		"sub/c-tie.md":     second.Add(50 * time.Millisecond),
		"sub/x/c-tie.md":   second.Add(50 * time.Millisecond),
		"sub/d-tie-too.md": second.Add(50 * time.Millisecond),
	} {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(p, at, at); err != nil {
			t.Fatal(err)
		}
	}
	cat, rec := getCatalog(t, h, "/api/projects/p1/resource-files?folder=sub")
	if rec.Code != http.StatusOK {
		t.Fatalf("catalog: %d %s", rec.Code, rec.Body)
	}
	var files []string
	for _, e := range cat.Entries {
		if e.Kind == "file" {
			files = append(files, e.Path)
		}
	}
	want := []string{"sub/b-newer.md", "sub/a-older.md", "sub/c-tie.md", "sub/x/c-tie.md", "sub/d-tie-too.md"}
	if strings.Join(files, ",") != strings.Join(want, ",") {
		t.Fatalf("file order = %v, want %v", files, want)
	}
}

func TestProjectResourceCatalogRefusals(t *testing.T) {
	srv, h, root := projectFilesServer(t)
	for target, status := range map[string]int{
		"/api/projects/nope/resource-files":                     http.StatusNotFound,
		"/api/projects/p1/resource-files?folder=missing":        http.StatusNotFound,
		"/api/projects/p1/resource-files?folder=..":             http.StatusUnprocessableEntity,
		"/api/projects/p1/resource-files?folder=%2Ftmp":         http.StatusUnprocessableEntity,
		"/api/projects/p1/resource-files?folder=docs%2F..%2F..": http.StatusUnprocessableEntity,
		"/api/projects/p1/resource-files?folder=to-other":       http.StatusUnprocessableEntity,
		"/api/projects/p1/resource-files?folder=Notes.md":       http.StatusUnprocessableEntity,
	} {
		if _, rec := getCatalog(t, h, target); rec.Code != status {
			t.Fatalf("%s: %d %s, want %d", target, rec.Code, rec.Body, status)
		}
	}

	// A legacy project whose shared folder was never created lists an empty
	// root and does not create it (FS-11.R14).
	if err := srv.configStore.WriteProject("legacy", config.Project{Title: "legacy", Cwd: t.TempDir()}); err != nil {
		t.Fatal(err)
	}
	cat, rec := getCatalog(t, h, "/api/projects/legacy/resource-files")
	if rec.Code != http.StatusOK || cat.Partial || cat.Entries == nil || len(cat.Entries) != 0 {
		t.Fatalf("legacy: %d %s", rec.Code, rec.Body)
	}
	legacyRoot, _ := srv.configStore.ProjectResourcesPath("legacy")
	if _, err := os.Lstat(legacyRoot); !os.IsNotExist(err) {
		t.Fatalf("catalog created the legacy root: %v", err)
	}

	// An invalid id stops before the config lookup: a directory where its
	// config file would be makes that lookup fail as internal (TS-05.R30).
	home := filepath.Dir(filepath.Dir(root))
	if err := os.MkdirAll(filepath.Join(home, "projects", "BAD.json"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, rec := getCatalog(t, h, "/api/projects/BAD/resource-files"); rec.Code != http.StatusNotFound {
		t.Fatalf("invalid id reached the config lookup: %d %s", rec.Code, rec.Body)
	}

	// An unreadable nested folder makes the scan partial, not failed; scoped to
	// that folder it is unreadable, not a wrong kind (FS-11.R14).
	locked := filepath.Join(root, "locked")
	if err := os.Mkdir(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o700) })
	if os.Geteuid() != 0 {
		cat, rec = getCatalog(t, h, "/api/projects/p1/resource-files")
		if rec.Code != http.StatusOK || !cat.Partial || cat.PartialReason != "unreadable" {
			t.Fatalf("unreadable folder: %d %s", rec.Code, rec.Body)
		}
		if _, rec = getCatalog(t, h, "/api/projects/p1/resource-files?folder=locked"); !strings.Contains(rec.Body.String(), `"file_unreadable"`) {
			t.Fatalf("scoped unreadable folder: %d %s", rec.Code, rec.Body)
		}
	}
	if _, rec = getCatalog(t, h, "/api/projects/p1/resource-files?folder=Notes.md"); !strings.Contains(rec.Body.String(), `"not_a_file"`) {
		t.Fatalf("file as folder: %d %s", rec.Code, rec.Body)
	}
	_ = os.Chmod(locked, 0o700)

	// A shared root that is a symlink or group-accessible is unusable (TS-05.R30).
	if err := os.Chmod(root, 0o750); err != nil {
		t.Fatal(err)
	}
	if _, rec := getCatalog(t, h, "/api/projects/p1/resource-files"); rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("group-readable root: %d %s", rec.Code, rec.Body)
	}
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	other, _ := srv.configStore.ProjectResourcesPath("p2")
	if err := os.Rename(root, root+".real"); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(other, root); err != nil {
		t.Fatal(err)
	}
	if _, rec := getCatalog(t, h, "/api/projects/p1/resource-files"); rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("symlinked root: %d %s", rec.Code, rec.Body)
	}
	if rec := doGET(t, h, "/api/projects/p1/resource-file?path=secret.md"); rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("read through symlinked root: %d %s", rec.Code, rec.Body)
	}
}

func TestProjectResourceCatalogAdmission(t *testing.T) {
	_, h, _ := projectFilesServer(t)
	for i := 0; i < resourceScanSlots; i++ {
		resourceScans <- struct{}{}
	}
	defer func() {
		for i := 0; i < resourceScanSlots; i++ {
			<-resourceScans
		}
	}()
	_, rec := getCatalog(t, h, "/api/projects/p1/resource-files")
	if rec.Code != http.StatusTooManyRequests || !strings.Contains(rec.Body.String(), "resource_scan_busy") {
		t.Fatalf("busy: %d %s", rec.Code, rec.Body)
	}
}

func TestProjectResourceFileRead(t *testing.T) {
	_, h, root := projectFilesServer(t)
	rec := doGET(t, h, "/api/projects/p1/resource-file?path="+url.QueryEscape("docs/notes.md"))
	if rec.Code != http.StatusOK {
		t.Fatalf("read: %d %s", rec.Code, rec.Body)
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("cache-control = %q", rec.Header().Get("Cache-Control"))
	}
	var got fileContent
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Content != "# nested\n" || got.Path != "docs/notes.md" || got.ProjectID != "p1" || got.Editable || got.Revision != "" || got.EditRefusal != resourceReadOnly {
		t.Fatalf("read payload = %+v", got)
	}

	if err := os.Symlink(filepath.Join(root, "Notes.md"), filepath.Join(root, "docs", "link.md")); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(filepath.Join(root, "fifo"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "bin.dat"), []byte{0xff, 0xfe, 0}, 0o600); err != nil {
		t.Fatal(err)
	}
	for path, code := range map[string]string{
		"to-other/secret.md": "not_a_file",
		"docs/link.md":       "not_a_file",
		"fifo":               "not_a_file",
		"docs":               "not_a_file",
		"missing.md":         "not_found",
		"bin.dat":            "not_text",
		"../p2/secret.md":    "path_refused",
		"/etc/hosts":         "path_refused",
		"docs//notes.md":     "path_refused",
		"./Notes.md":         "path_refused",
		"":                   "path_refused",
	} {
		rec := doGET(t, h, "/api/projects/p1/resource-file?path="+url.QueryEscape(path))
		if !strings.Contains(rec.Body.String(), `"`+code+`"`) {
			t.Fatalf("%q: %d %s, want %s", path, rec.Code, rec.Body, code)
		}
	}
}

// The project browser is desktop-only: the phone mux never serves it, and the
// loopback guard rejects a non-local Host (FS-11.A9, INV §14).
func TestProjectResourceRoutesDeniedRemotely(t *testing.T) {
	s, h, _ := projectFilesServer(t)
	remote := s.remoteRoutes(testDomain, testWhoIs(map[string]string{"100.64.0.2:5000": "n"}))
	token := pairTestDevice(t, s, "d1", "n")
	for _, path := range []string{"/api/projects/p1/resource-files", "/api/projects/p1/resource-file?path=Notes.md"} {
		rec := httptest.NewRecorder()
		remote.ServeHTTP(rec, phoneRequest(http.MethodGet, path, "", token))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("remote %s: %d %s", path, rec.Code, rec.Body)
		}
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Host = "attacker.example"
		rec = httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("non-local host %s: %d", path, rec.Code)
		}
	}
}
