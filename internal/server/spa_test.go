package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

var twoEntryDist = fstest.MapFS{
	"index.html":         {Data: []byte("desktop")},
	"remote.html":        {Data: []byte("phone")},
	"remote-sw.js":       {Data: []byte("sw")},
	"remote.webmanifest": {Data: []byte("manifest")},
	"assets/phone.js":    {Data: []byte("phone js")},
	"assets/shared.js":   {Data: []byte("shared js")},
	"assets/main.js":     {Data: []byte("desktop js")},
	".vite/manifest.json": {Data: []byte(`{
		"remote.html":{"file":"assets/phone.js","imports":["_shared.js"]},
		"_shared.js":{"file":"assets/shared.js"},
		"index.html":{"file":"assets/main.js","imports":["_shared.js"]}
	}`)},
}

func serveBody(t *testing.T, h http.Handler, path string) (int, string) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	body, _ := io.ReadAll(rec.Body)
	return rec.Code, string(body)
}

// The tailnet serves only the phone entry and the loopback never serves it
// (TS-13.R14, TS-08.R73).
func TestStaticEntriesStayOnTheirListener(t *testing.T) {
	phone := phoneSPAHandler(twoEntryDist)
	for path, want := range map[string]string{
		"/": "phone", "/pair": "phone", "/agent/a1": "phone", "/index.html": "phone",
		"/sw.js": "sw", "/remote.webmanifest": "manifest", "/assets/phone.js": "phone js", "/assets/shared.js": "shared js", "/assets": "phone",
	} {
		if code, body := serveBody(t, phone, path); code != 200 || body != want {
			t.Errorf("phone %s = %d %q, want %q", path, code, body, want)
		}
	}

	desktop := spaHandler(twoEntryDist)
	for path, want := range map[string]string{
		"/": "desktop", "/remote.html": "desktop", "/remote-sw.js": "desktop",
		"/remote.webmanifest": "desktop", "/assets/main.js": "desktop js", "/settings": "desktop",
	} {
		if code, body := serveBody(t, desktop, path); code != 200 || body != want {
			t.Errorf("desktop %s = %d %q, want %q", path, code, body, want)
		}
	}

	if code, _ := serveBody(t, phoneSPAHandler(fstest.MapFS{"index.html": {Data: []byte("desktop")}}), "/"); code != 404 {
		t.Fatalf("unbuilt phone app = %d, want 404 rather than the desktop bundle", code)
	}
	if code, _ := serveBody(t, phone, "/assets/main.js"); code != 404 {
		t.Fatalf("desktop entry asset = %d, want 404", code)
	}
}

// Investigation: mobile stylesheet failure. Retired asset URLs must not be
// answered with app HTML, which the phone worker can cache as the stylesheet.
func TestPhoneMissingStylesheetDoesNotServeAppHTML(t *testing.T) {
	rec := httptest.NewRecorder()
	phoneSPAHandler(twoEntryDist).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/assets/remote-retired.css", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing stylesheet = %d (%s), want 404", rec.Code, rec.Header().Get("Content-Type"))
	}
}

// Investigation: the dev FS changes after a UI rebuild while the handler lives.
func TestPhoneStylesheetAfterDevRebuild(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".vite"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "assets"), 0o700); err != nil {
		t.Fatal(err)
	}
	write := func(name, contents string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, name), []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("remote.html", `<link rel="stylesheet" href="/assets/old.css">`)
	write("assets/old.css", "body { color: red; }")
	write(".vite/manifest.json", `{"remote.html":{"css":["assets/old.css"]}}`)
	handler := phoneSPAHandlerWithGraph(os.DirFS(root), nil, true)
	write("remote.html", `<link rel="stylesheet" href="/assets/new.css">`)
	write("assets/new.css", "body { color: blue; }")
	write(".vite/manifest.json", `{"remote.html":{"css":["assets/new.css"]}}`)
	if code, body := serveBody(t, handler, "/"); code != 200 || body != `<link rel="stylesheet" href="/assets/new.css">` {
		t.Fatalf("rebuilt entry = %d %q", code, body)
	}
	if code, body := serveBody(t, handler, "/assets/new.css"); code != 200 || body != "body { color: blue; }" {
		t.Fatalf("rebuilt stylesheet = %d %q, want current CSS", code, body)
	}
}
