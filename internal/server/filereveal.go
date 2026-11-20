package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os/exec"
	"path/filepath"

	"golang.org/x/sys/unix"

	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
)

// Finder reveal for chat and project files (FS-03.R119, FS-11.R16, TS-03.R94,
// TS-05.R31). Like Open in browser, these routes are absent from the phone
// allowlist: only an explicit local menu selection may act on the Mac.

var errRevealUnsupported = errors.New("finder reveal unsupported on this host")

// revealWithRunner runs the fixed reveal command with separate arguments and no
// shell. The runner seam proves the argv boundary without opening Finder.
func revealWithRunner(ctx context.Context, path string, run func(*exec.Cmd) ([]byte, error)) error {
	if _, err := run(exec.CommandContext(ctx, "/usr/bin/open", "-R", "--", path)); err != nil {
		return errors.New("finder reveal failed")
	}
	return nil
}

func decodeRevealPath(w http.ResponseWriter, r *http.Request) (string, *runtime.APIError) {
	var body struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&body); err != nil {
		return "", apiError(runtime.CodeValidation, "invalid file request")
	}
	return body.Path, nil
}

func (s *Server) reveal(path string) *runtime.APIError {
	reveal := s.revealFile
	if reveal == nil {
		reveal = revealInFinder
	}
	if err := reveal(path); err != nil {
		if errors.Is(err, errRevealUnsupported) {
			return apiError(runtime.CodeNotImplemented, "Open in Finder is not available on this computer")
		}
		return apiError(runtime.CodeInternal, "Finder could not reveal that file")
	}
	return nil
}

// regularFileIdentity verifies fd as a regular file and returns its identity.
func regularFileIdentity(fd int) (unix.Stat_t, *runtime.APIError) {
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil {
		return st, apiError(runtime.CodeFileUnreadable, "that file could not be read")
	}
	if st.Mode&unix.S_IFMT != unix.S_IFREG {
		return st, apiError(runtime.CodeNotAFile, "that path does not name a regular file")
	}
	return st, nil
}

// handleChatFileReveal serves POST /api/sessions/{id}/file/reveal-in-finder with
// the chat file reader's authority and working-directory rules (TS-05.R24/R31).
func (s *Server) handleChatFileReveal(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	refuse := func(apiErr *runtime.APIError) {
		s.log.Info("file reveal refused", "agent_id", id, "code", apiErr.Code)
		writeAPIError(w, apiErr)
	}
	cwd, apiErr := s.chatSessionCwd(id)
	if apiErr != nil {
		refuse(apiErr)
		return
	}
	raw, apiErr := decodeRevealPath(w, r)
	if apiErr != nil {
		refuse(apiErr)
		return
	}
	path, _, apiErr := resolveFileReadPath(cwd, raw)
	if apiErr != nil {
		refuse(apiErr)
		return
	}
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err != nil {
		if errors.Is(err, unix.ENOENT) {
			refuse(apiError(runtime.CodeNotFound, "that file no longer exists"))
		} else {
			refuse(apiError(runtime.CodeFileUnreadable, "that file could not be read"))
		}
		return
	}
	_, apiErr = regularFileIdentity(fd)
	unix.Close(fd)
	if apiErr != nil {
		refuse(apiErr)
		return
	}
	if apiErr := s.reveal(path); apiErr != nil {
		refuse(apiErr)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleProjectResourceReveal serves
// POST /api/projects/{project}/resource-file/reveal-in-finder. The target is
// opened beneath the shared root by descriptor, and its identity is checked
// again just before the pathname is handed to Finder (TS-05.R30/R31).
func (s *Server) handleProjectResourceReveal(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("project")
	refuse := func(apiErr *runtime.APIError) {
		s.log.Info("project files refused", "project", projectID, "op", "reveal", "code", apiErr.Code)
		writeAPIError(w, apiErr)
	}
	rel, apiErr := decodeRevealPath(w, r)
	if apiErr != nil {
		refuse(apiErr)
		return
	}
	identity := func() (unix.Stat_t, *runtime.APIError) {
		fd, apiErr := s.openProjectResource(projectID, rel)
		if apiErr != nil {
			return unix.Stat_t{}, apiErr
		}
		defer unix.Close(fd)
		return regularFileIdentity(fd)
	}
	first, apiErr := identity()
	if apiErr != nil {
		refuse(apiErr)
		return
	}
	root, err := s.configStore.ProjectResourcesPath(projectID)
	if err != nil {
		refuse(apiError(runtime.CodeNotFound, "no such project: "+projectID))
		return
	}
	again, apiErr := identity()
	if apiErr != nil {
		refuse(apiErr)
		return
	}
	if again.Dev != first.Dev || again.Ino != first.Ino {
		refuse(apiError(runtime.CodeFileChanging, "that file is changing; try again"))
		return
	}
	if apiErr := s.reveal(filepath.Join(root, filepath.FromSlash(rel))); apiErr != nil {
		refuse(apiErr)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// chatSessionCwd applies the chat file authority shared by the read, browser
// and reveal routes: a known chat agent with a recorded session.
func (s *Server) chatSessionCwd(id string) (string, *runtime.APIError) {
	agent, err := s.stateStore.ReadAgent(id)
	if err != nil {
		if errors.Is(err, state.ErrNotFound) {
			return "", apiError(runtime.CodeNotFound, "no such agent: "+id)
		}
		return "", apiError(runtime.CodeInternal, "the agent could not be read")
	}
	if agent.Interface != "chat" {
		return "", apiError(runtime.CodeConflict, "file actions are only available for chat agents")
	}
	snap, err := s.stateStore.ReadSession(id)
	if err != nil {
		if errors.Is(err, state.ErrNotFound) {
			return "", apiError(runtime.CodeNotFound, "no such agent: "+id)
		}
		return "", apiError(runtime.CodeInternal, "the agent could not be read")
	}
	return snap.Cwd, nil
}
