package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"golang.org/x/sys/unix"

	"github.com/AsaphNoam/Chuck/internal/config"
	"github.com/AsaphNoam/Chuck/internal/runtime"
)

// Project file browser routes (FS-11.R11–R16, TS-03.R93, TS-05.R30). Every
// read opens the project's shared-resources root by descriptor and walks beneath
// it without following a symlink. The chat file loaders have a broader authority
// and are never a fallback here. Resource names and paths stay out of logs.

const (
	resourceScanMaxEntries = 20_000
	resourceScanMaxBytes   = 8 << 20
	resourceScanMaxDepth   = 64
	resourceScanTimeout    = 5 * time.Second
	resourceScanSlots      = 4
	// resourceReadOnly is the edit refusal of every project preview: the browser
	// has no editable revision (TS-03.R93).
	resourceReadOnly = "project_read_only"
)

// resourceScans admits at most four concurrent catalog scans per process, with
// no queue: a fifth request is refused as retryable (TS-03.R93, INV §16).
var resourceScans = make(chan struct{}, resourceScanSlots)

type resourceEntry struct {
	Path    string  `json:"path"`
	Name    string  `json:"name"`
	Kind    string  `json:"kind"`
	ModTime *string `json:"mod_time"`
}

type resourceCatalog struct {
	ProjectID     string          `json:"project_id"`
	Folder        string          `json:"folder"`
	Entries       []resourceEntry `json:"entries"`
	Partial       bool            `json:"partial"`
	PartialReason string          `json:"partial_reason,omitempty"`
}

// errResourceRootMissing marks a legacy project whose shared folder was never
// created; the browser shows an empty root and never creates it (FS-11.R14).
var errResourceRootMissing = errors.New("project resources missing")

// openProjectResourceRoot validates the project, then opens the resource parent
// and the project leaf as real, owner-only directories without following a
// symlink at either level (TS-05.R30). The caller owns the returned descriptor.
func (s *Server) openProjectResourceRoot(projectID string) (int, *runtime.APIError, error) {
	if _, err := s.configStore.ReadProject(projectID); err != nil {
		if errors.Is(err, config.ErrNotFound) {
			return -1, apiError(runtime.CodeNotFound, "no such project: "+projectID), nil
		}
		return -1, apiError(runtime.CodeInternal, "the project could not be read"), nil
	}
	leaf, err := s.configStore.ProjectResourcesPath(projectID)
	if err != nil {
		return -1, apiError(runtime.CodeNotFound, "no such project: "+projectID), nil
	}
	unusable := apiError(runtime.CodeWorkspaceUnavailable, "the project's shared folder is unavailable")
	parent, err := unix.Open(path.Dir(leaf), unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		if errors.Is(err, unix.ENOENT) {
			return -1, nil, errResourceRootMissing
		}
		return -1, unusable, nil
	}
	if !ownerOnlyDir(parent) {
		unix.Close(parent)
		return -1, unusable, nil
	}
	root, err := openatNoFollow(parent, []string{path.Base(leaf)}, true)
	if err != nil {
		if errors.Is(err, unix.ENOENT) {
			return -1, nil, errResourceRootMissing
		}
		return -1, unusable, nil
	}
	if !ownerOnlyDir(root) {
		unix.Close(root)
		return -1, unusable, nil
	}
	return root, nil, nil
}

// ownerOnlyDir reports whether fd is a directory owned by this process's user
// with no group or world access (TS-02.R13, INV §14).
func ownerOnlyDir(fd int) bool {
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil {
		return false
	}
	return st.Mode&unix.S_IFMT == unix.S_IFDIR && int(st.Uid) == os.Geteuid() && st.Mode&0o077 == 0
}

// splitResourcePath accepts only a clean root-relative path: no absolute
// spelling, empty, "." or ".." segment, or NUL. The original spelling is kept.
func splitResourcePath(raw string) ([]string, bool) {
	if raw == "" || strings.ContainsRune(raw, 0) || strings.HasPrefix(raw, "/") {
		return nil, false
	}
	parts := strings.Split(raw, "/")
	for _, p := range parts {
		if p == "" || p == "." || p == ".." {
			return nil, false
		}
	}
	return parts, true
}

func noStore(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
}

// handleProjectResourceFiles serves GET /api/projects/{project}/resource-files
// (TS-03.R93): an on-demand metadata catalog of the shared root or one folder.
func (s *Server) handleProjectResourceFiles(w http.ResponseWriter, r *http.Request) {
	noStore(w)
	projectID := r.PathValue("project")
	folder := r.URL.Query().Get("folder")
	refuse := func(apiErr *runtime.APIError) {
		s.log.Info("project files refused", "project", projectID, "op", "catalog", "code", apiErr.Code)
		writeAPIError(w, apiErr)
	}
	var parts []string
	if folder != "" {
		var ok bool
		if parts, ok = splitResourcePath(folder); !ok {
			refuse(apiError(runtime.CodePathRefused, "that folder is outside the project's shared folder"))
			return
		}
	}
	select {
	case resourceScans <- struct{}{}:
		defer func() { <-resourceScans }()
	default:
		refuse(apiError(runtime.CodeResourceScanBusy, "other file lists are being read; try again"))
		return
	}
	root, apiErr, err := s.openProjectResourceRoot(projectID)
	if apiErr != nil {
		refuse(apiErr)
		return
	}
	out := resourceCatalog{ProjectID: projectID, Folder: folder, Entries: []resourceEntry{}}
	if errors.Is(err, errResourceRootMissing) {
		if folder != "" {
			refuse(apiError(runtime.CodeNotFound, "that folder no longer exists"))
			return
		}
		writeJSON(w, http.StatusOK, out)
		return
	}
	dir := root
	if len(parts) > 0 {
		if dir, err = openatNoFollow(root, parts, true); err != nil {
			if errors.Is(err, unix.ENOENT) {
				refuse(apiError(runtime.CodeNotFound, "that folder no longer exists"))
			} else {
				refuse(apiError(runtime.CodeNotAFile, "that path does not name a folder"))
			}
			return
		}
	}
	ctx, cancel := context.WithTimeout(r.Context(), resourceScanTimeout)
	defer cancel()
	scan := resourceScan{ctx: ctx, entries: out.Entries}
	scan.walk(dir, folder, len(parts))
	out.Entries = scan.entries
	out.Partial = scan.limited || scan.unreadable
	switch {
	case scan.limited:
		out.PartialReason = "limit"
	case scan.unreadable:
		out.PartialReason = "unreadable"
	}
	sortResourceEntries(out.Entries)
	writeJSON(w, http.StatusOK, out)
}

type resourceScan struct {
	ctx        context.Context
	entries    []resourceEntry
	bytes      int
	visited    int
	limited    bool
	unreadable bool
}

// walk lists dirfd's entries by metadata only, descending into directories by
// no-follow descriptor. It takes ownership of dirfd. Limits apply while walking,
// and a cancelled or expired scan stops taking work (TS-03.R93, INV §16).
func (sc *resourceScan) walk(dirfd int, rel string, depth int) {
	f := os.NewFile(uintptr(dirfd), rel)
	defer f.Close()
	for {
		names, err := f.Readdirnames(256)
		for _, name := range names {
			if sc.limited {
				return
			}
			if sc.ctx.Err() != nil || sc.visited >= resourceScanMaxEntries {
				sc.limited = true
				return
			}
			sc.visited++
			entryPath := name
			if rel != "" {
				entryPath = rel + "/" + name
			}
			var st unix.Stat_t
			if err := unix.Fstatat(dirfd, name, &st, unix.AT_SYMLINK_NOFOLLOW); err != nil {
				sc.unreadable = true
				continue
			}
			entry := resourceEntry{Path: entryPath, Name: name, Kind: resourceKind(uint32(st.Mode))}
			mod := time.Unix(st.Mtim.Unix()).UTC().Format(time.RFC3339)
			entry.ModTime = &mod
			encoded, _ := json.Marshal(entry)
			if sc.bytes+len(encoded)+1 > resourceScanMaxBytes {
				sc.limited = true
				return
			}
			sc.bytes += len(encoded) + 1
			sc.entries = append(sc.entries, entry)
			if entry.Kind != "directory" {
				continue
			}
			if depth+1 >= resourceScanMaxDepth {
				sc.limited = true
				continue
			}
			child, err := unix.Openat(dirfd, name, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
			if err != nil {
				sc.unreadable = true
				continue
			}
			sc.walk(child, entryPath, depth+1)
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				sc.unreadable = true
			}
			return
		}
	}
}

func resourceKind(mode uint32) string {
	switch mode & unix.S_IFMT {
	case unix.S_IFREG:
		return "file"
	case unix.S_IFDIR:
		return "directory"
	case unix.S_IFLNK:
		return "symlink"
	default:
		return "special"
	}
}

// sortResourceEntries orders rows newest first, then by name and path
// (FS-11.R12). An entry without a time sorts after every dated entry.
func sortResourceEntries(entries []resourceEntry) {
	sort.SliceStable(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		switch {
		case a.ModTime == nil && b.ModTime != nil:
			return false
		case a.ModTime != nil && b.ModTime == nil:
			return true
		case a.ModTime != nil && *a.ModTime != *b.ModTime:
			return *a.ModTime > *b.ModTime
		case a.Name != b.Name:
			return a.Name < b.Name
		default:
			return a.Path < b.Path
		}
	})
}

// handleProjectResourceFile serves GET /api/projects/{project}/resource-file
// (TS-03.R93): one root-relative file through the bounded snapshot decoder,
// opened beneath the shared root by descriptor and never editable.
func (s *Server) handleProjectResourceFile(w http.ResponseWriter, r *http.Request) {
	noStore(w)
	projectID := r.PathValue("project")
	rel := r.URL.Query().Get("path")
	refuse := func(apiErr *runtime.APIError) {
		s.log.Info("project files refused", "project", projectID, "op", "read", "code", apiErr.Code)
		writeAPIError(w, apiErr)
	}
	fd, apiErr := s.openProjectResource(projectID, rel)
	if apiErr != nil {
		refuse(apiErr)
		return
	}
	f := os.NewFile(uintptr(fd), rel)
	defer f.Close()
	out, apiErr := readOpenedFile(f, "", rel, true, false, nil, resourceReadOnly)
	if apiErr != nil {
		refuse(apiErr)
		return
	}
	out.ProjectID = projectID
	writeJSON(w, http.StatusOK, out)
}

// openProjectResource opens one root-relative entry beneath the project's
// shared root without following any component. The caller owns the descriptor
// and must verify its kind before use (TS-05.R30).
func (s *Server) openProjectResource(projectID, rel string) (int, *runtime.APIError) {
	parts, ok := splitResourcePath(rel)
	if !ok {
		return -1, apiError(runtime.CodePathRefused, "that path is outside the project's shared folder")
	}
	root, apiErr, err := s.openProjectResourceRoot(projectID)
	if apiErr != nil {
		return -1, apiErr
	}
	if err != nil {
		return -1, apiError(runtime.CodeNotFound, "that file no longer exists")
	}
	fd, err := openatNoFollow(root, parts, false)
	switch {
	case err == nil:
		return fd, nil
	case errors.Is(err, unix.ENOENT):
		return -1, apiError(runtime.CodeNotFound, "that file no longer exists")
	case errors.Is(err, unix.ELOOP), errors.Is(err, unix.ENOTDIR):
		return -1, apiError(runtime.CodeNotAFile, "that path is a link or special entry and cannot be opened")
	default:
		return -1, apiError(runtime.CodeFileUnreadable, "that file could not be read")
	}
}
