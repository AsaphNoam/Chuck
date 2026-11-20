package server

import (
	"encoding/json"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/AsaphNoam/Chuck/internal/runtime"
)

// These handlers are deliberately absent from the remote route allowlist: only
// an explicit action in the local UI may launch an application on the Mac.
func (s *Server) handleFileOpenBrowser(w http.ResponseWriter, r *http.Request) {
	cwd, apiErr := s.chatSessionCwd(r.PathValue("id"))
	if apiErr != nil {
		writeAPIError(w, apiErr)
		return
	}
	s.openHTMLFile(w, r, cwd)
}

func (s *Server) handleThinkTankSourceOpenBrowser(w http.ResponseWriter, r *http.Request) {
	sources, err := s.stateStore.ThinkTankSources(r.PathValue("id"))
	if err != nil {
		s.writeThinkTankError(w, err)
		return
	}
	for _, src := range sources {
		if src.SourceID == r.PathValue("source_id") {
			s.openHTMLFile(w, r, src.Cwd)
			return
		}
	}
	writeAPIError(w, apiError(runtime.CodeNotFound, "no such room source"))
}

func (s *Server) openHTMLFile(w http.ResponseWriter, r *http.Request, cwd string) {
	var body struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&body); err != nil {
		writeAPIError(w, apiError(runtime.CodeValidation, "invalid file request"))
		return
	}
	path, display, ae := resolveFileReadPath(cwd, body.Path)
	if ae != nil {
		writeAPIError(w, ae)
		return
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".html" && ext != ".htm" {
		writeAPIError(w, apiError(runtime.CodeValidation, "only HTML files can be opened in the browser"))
		return
	}
	if _, ae := readLocalFile(path, display); ae != nil {
		writeAPIError(w, ae)
		return
	}
	open := s.openBrowserFile
	if open == nil {
		open = openHTMLInBrowser
	}
	if err := open(path); err != nil {
		writeAPIError(w, apiError(runtime.CodeInternal, "could not open file in browser: "+err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"path": display})
}

// Escape URL-significant filename characters before handing the file URL to
// the operating system. pkg/browser.OpenFile concatenates an unescaped path.
func htmlFileURL(path string) string {
	return (&url.URL{Scheme: "file", Path: path}).String()
}
