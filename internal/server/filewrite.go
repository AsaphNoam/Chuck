package server

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/AsaphNoam/Chuck/internal/runtime"
	"golang.org/x/sys/unix"
)

// Neither the observation nor the text precondition is authority to open a path.
func opaqueFileRevision(stamp string) string {
	h := sha256.Sum256([]byte(stamp))
	return hex.EncodeToString(h[:])
}

func textFileRevision(stamp string, body []byte) string {
	h := sha256.New()
	h.Write([]byte(stamp))
	h.Write([]byte{0})
	h.Write(body)
	return hex.EncodeToString(h.Sum(nil))
}

func fileDescriptorStamp(f *os.File) (string, error) {
	var st unix.Stat_t
	if err := unix.Fstat(int(f.Fd()), &st); err != nil {
		return "", err
	}
	return fmt.Sprintf("%d:%d:%d:%d:%d:%d:%d:%d:%d:%d:%d:%s", st.Dev, st.Ino, st.Size,
		st.Mtim.Sec, st.Mtim.Nsec, st.Ctim.Sec, st.Ctim.Nsec, st.Mode, st.Uid, st.Gid, st.Nlink, fileExtraStamp(st)), nil
}

type fileProbe struct {
	Path             string `json:"path"`
	ObservedRevision string `json:"observed_revision"`
}

// Probe deliberately does not read content (including potentially large images).
// It uses the same no-follow regular-descriptor admission as remote GET.
func probeLocalFile(path, display string, noFollow bool) (fileProbe, *runtime.APIError) {
	out, ae := readLocalFileSnapshot(path, display, nil, noFollow, true, nil)
	return fileProbe{out.Path, out.ObservedRevision}, ae
}

func fileOpenError(err error) *runtime.APIError {
	if os.IsNotExist(err) {
		return apiError(runtime.CodeNotFound, "that file no longer exists")
	}
	return apiError(runtime.CodeFileUnreadable, "that file could not be read")
}

// Unsupported access metadata is refused rather than silently discarded. This
// intentionally conservative first implementation preserves ordinary Unix modes
// and ownership, and refuses files/directories with extended access metadata.
func descriptorEditRefusal(f *os.File) string {
	var st unix.Stat_t
	if unix.Fstat(int(f.Fd()), &st) != nil {
		return "metadata_unavailable"
	}
	if st.Mode&unix.S_IFMT != unix.S_IFREG {
		return "not_regular"
	}
	if st.Nlink != 1 {
		return "hard_link"
	}
	if st.Uid != uint32(os.Geteuid()) {
		return "not_owned"
	}
	if st.Mode&0200 == 0 {
		return "not_writable"
	}
	if st.Mode&07000 != 0 {
		return "unsupported_metadata"
	}
	if !fileMetadataSupported(int(f.Fd()), st) {
		return "unsupported_metadata"
	}
	return ""
}

func fileEditRefusal(path string, f *os.File) string {
	info, err := os.Lstat(path)
	if err != nil {
		return "metadata_unavailable"
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "symlink"
	}
	if reason := descriptorEditRefusal(f); reason != "" {
		return reason
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(path))
	if err != nil {
		return "parent_unavailable"
	}
	pfd, err := unix.Open(parent, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return "parent_unavailable"
	}
	defer unix.Close(pfd)
	if unix.Faccessat(pfd, ".", unix.W_OK|unix.X_OK, 0) != nil {
		return "parent_not_writable"
	}
	fd, err := unix.Openat(pfd, filepath.Base(path), unix.O_WRONLY|unix.O_NONBLOCK|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return "not_writable"
	}
	defer unix.Close(fd)
	var got, want unix.Stat_t
	if unix.Fstat(fd, &got) != nil || unix.Fstat(int(f.Fd()), &want) != nil || got.Dev != want.Dev || got.Ino != want.Ino {
		return "file_changing"
	}
	return ""
}

type fileSaveRequest struct {
	Path         *string `json:"path"`
	BaseRevision *string `json:"base_revision"`
	Content      *string `json:"content"`
}

func (s *Server) serveFileSave(w http.ResponseWriter, r *http.Request, cwd, agentID string) {
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 4*1024*1024))
	if err != nil || !utf8.Valid(raw) {
		writeAPIError(w, apiError(runtime.CodeValidation, "invalid or oversized save request"))
		return
	}
	var req fileSaveRequest
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if dec.Decode(&req) != nil || dec.Decode(new(any)) != io.EOF || req.Path == nil || req.BaseRevision == nil || req.Content == nil {
		writeAPIError(w, apiError(runtime.CodeValidation, "path, base_revision and content are required"))
		return
	}
	if _, err := hex.DecodeString(*req.BaseRevision); err != nil || len(*req.BaseRevision) != 64 || len(*req.Content) > fileReadLimit || !utf8.ValidString(*req.Content) {
		writeAPIError(w, apiError(runtime.CodeValidation, "invalid revision or text content"))
		return
	}
	path, display, ae := resolveFileReadPath(cwd, *req.Path)
	if ae != nil {
		writeAPIError(w, ae)
		return
	}
	out, ae := saveLocalFile(path, display, *req.BaseRevision, []byte(*req.Content), nil)
	if ae != nil {
		s.log.Info("file save refused", "agent_id", agentID, "path", display, "code", ae.Code, "reason", ae.Details["reason"])
		writeAPIError(w, ae)
		return
	}
	out.AgentID = agentID
	s.log.Info("file saved", "agent_id", agentID, "path", display)
	writeJSON(w, http.StatusOK, out)
}

// Locks live only as long as an owner or waiter. Admission also bounds distinct
// concurrent parent directories, rather than retaining an attacker-controlled path map.
var fileSaveLocks = struct {
	sync.Mutex
	entries map[string]*fileSaveLock
}{entries: make(map[string]*fileSaveLock)}

type fileSaveLock struct {
	sync.Mutex
	refs int
}

func lockFileSave(key string) (func(), bool) {
	fileSaveLocks.Lock()
	l := fileSaveLocks.entries[key]
	if l == nil {
		if len(fileSaveLocks.entries) >= 128 {
			fileSaveLocks.Unlock()
			return nil, false
		}
		l = &fileSaveLock{}
		fileSaveLocks.entries[key] = l
	}
	l.refs++
	fileSaveLocks.Unlock()
	l.Lock()
	return func() {
		l.Unlock()
		fileSaveLocks.Lock()
		l.refs--
		if l.refs == 0 {
			delete(fileSaveLocks.entries, key)
		}
		fileSaveLocks.Unlock()
	}, true
}

// A hook at the prepared boundary is used only by deterministic adversarial
// tests. Production has no callback or configurable filesystem behavior.
func saveLocalFile(path, display, base string, content []byte, prepared func()) (fileContent, *runtime.APIError) {
	return saveLocalFileWithHooks(path, display, base, content, prepared, nil)
}

func saveLocalFileWithHooks(path, display, base string, content []byte, prepared func(), syncDirectory func(int) error) (fileContent, *runtime.APIError) {
	if len(content) > fileReadLimit || !utf8.Valid(content) {
		return fileContent{}, apiError(runtime.CodeValidation, "invalid text content")
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(path))
	if err != nil {
		return fileContent{}, fileOpenError(err)
	}
	name := filepath.Base(path)
	pfd, err := unix.Open(parent, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return fileContent{}, fileOpenError(err)
	}
	defer unix.Close(pfd)
	var parentStat unix.Stat_t
	if unix.Fstat(pfd, &parentStat) != nil {
		return fileContent{}, fileOpenError(err)
	}
	// Serialize the whole pinned directory: distinct names can address the same
	// entry on case-insensitive or normalization-insensitive filesystems.
	unlock, ok := lockFileSave(fmt.Sprintf("%d:%d", parentStat.Dev, parentStat.Ino))
	if !ok {
		return fileContent{}, apiError(runtime.CodeFileChanging, "too many concurrent file saves; try again")
	}
	defer unlock()
	// A preceding save may have created/removed an entry while we waited.
	// Refresh directory metadata under the parent lock; its pinned identity
	// remains the one used for the lock key.
	if unix.Fstat(pfd, &parentStat) != nil {
		return fileContent{}, apiError(runtime.CodeFileUnreadable, "that parent directory could not be read")
	}
	refuse := func(reason string) (fileContent, *runtime.APIError) {
		ae := apiError(runtime.CodeFileNotEditable, "that file cannot be safely edited")
		ae.Details = map[string]any{"reason": reason}
		return fileContent{}, ae
	}
	conflict := func() (fileContent, *runtime.APIError) {
		return fileContent{}, apiError(runtime.CodeFileConflict, "that file changed; compare before saving")
	}
	parentValid := func() bool {
		var now unix.Stat_t
		return unix.Stat(parent, &now) == nil && now.Dev == parentStat.Dev && now.Ino == parentStat.Ino && now.Mode == parentStat.Mode && now.Uid == parentStat.Uid && now.Gid == parentStat.Gid && now.Nlink == parentStat.Nlink && fileExtraStamp(now) == fileExtraStamp(parentStat) && unix.Faccessat(pfd, ".", unix.W_OK|unix.X_OK, 0) == nil
	}
	if !parentValid() {
		return conflict()
	}
	fd, err := unix.Openat(pfd, name, unix.O_RDONLY|unix.O_NONBLOCK|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		if errors.Is(err, unix.ELOOP) {
			return refuse("symlink")
		}
		return fileContent{}, fileOpenError(err)
	}
	f := os.NewFile(uintptr(fd), name)
	defer f.Close()
	if reason := descriptorEditRefusal(f); reason != "" {
		return refuse(reason)
	}
	if unix.Faccessat(pfd, ".", unix.W_OK|unix.X_OK, 0) != nil {
		return refuse("parent_not_writable")
	}
	writeFD, err := unix.Openat(pfd, name, unix.O_WRONLY|unix.O_NONBLOCK|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return refuse("not_writable")
	}
	var writeStat, original unix.Stat_t
	err = unix.Fstat(writeFD, &writeStat)
	unix.Close(writeFD)
	if err != nil || unix.Fstat(fd, &original) != nil || writeStat.Dev != original.Dev || writeStat.Ino != original.Ino {
		return conflict()
	}
	currentRevision := func() (string, error) {
		before, err := fileDescriptorStamp(f)
		if err != nil {
			return "", err
		}
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return "", err
		}
		body, err := io.ReadAll(io.LimitReader(f, fileReadLimit+1))
		if err != nil {
			return "", err
		}
		after, err := fileDescriptorStamp(f)
		var entry unix.Stat_t
		if err != nil || before != after || len(body) > fileReadLimit || !utf8.Valid(body) || unix.Fstatat(pfd, name, &entry, unix.AT_SYMLINK_NOFOLLOW) != nil || entry.Dev != original.Dev || entry.Ino != original.Ino {
			return "", errors.New("file changed")
		}
		return textFileRevision(before, body), nil
	}
	rev, err := currentRevision()
	if err != nil || rev != base {
		return conflict()
	}
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return fileContent{}, apiError(runtime.CodeInternal, "could not prepare file save")
	}
	tmpName := ".chuck-save-" + hex.EncodeToString(nonce[:])
	tfd, err := unix.Openat(pfd, tmpName, unix.O_RDWR|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0600)
	if err != nil {
		return refuse("parent_not_writable")
	}
	tmp := os.NewFile(uintptr(tfd), tmpName)
	defer tmp.Close()
	defer unix.Unlinkat(pfd, tmpName, 0)
	if unix.Fchown(tfd, int(original.Uid), int(original.Gid)) != nil || unix.Fchmod(tfd, uint32(original.Mode)&0777) != nil {
		return refuse("unsupported_metadata")
	}
	var tempStat unix.Stat_t
	if unix.Fstat(tfd, &tempStat) != nil || !fileMetadataSupported(tfd, tempStat) || !fileAccessMetadataEqual(fd, tfd) {
		return refuse("unsupported_metadata")
	}
	if _, err := tmp.Write(content); err != nil {
		return fileContent{}, apiError(runtime.CodeInternal, "could not write file save")
	}
	if tmp.Sync() != nil {
		return fileContent{}, apiError(runtime.CodeInternal, "could not sync file save")
	}
	// APFS includes ordinary directory entries in st_nlink. Account for our
	// own temporary entry before rechecking the parent at commit.
	var preparedParent unix.Stat_t
	if unix.Fstat(pfd, &preparedParent) != nil {
		return conflict()
	}
	parentStat.Nlink = preparedParent.Nlink
	if prepared != nil {
		prepared()
	}
	if !parentValid() || descriptorEditRefusal(f) != "" {
		return conflict()
	}
	rev, err = currentRevision()
	if err != nil || rev != base {
		return conflict()
	}
	if unix.Renameat(pfd, tmpName, pfd, name) != nil {
		// Filesystems with remote backing may report failure after committing
		// the rename. Never invite a blind retry once replacement was attempted.
		return fileContent{}, apiError(runtime.CodeFileSaveUncertain, "file replacement may have succeeded; reread before saving")
	}
	// From this point on a failure must never invite a blind retry.
	if syncDirectory == nil {
		syncDirectory = unix.Fsync
	}
	if syncDirectory(pfd) != nil {
		return fileContent{}, apiError(runtime.CodeFileSaveUncertain, "file replacement may have succeeded; reread before saving")
	}
	stamp, err := fileDescriptorStamp(tmp)
	if _, seekErr := tmp.Seek(0, io.SeekStart); seekErr != nil {
		return fileContent{}, apiError(runtime.CodeFileSaveUncertain, "file replacement may have succeeded; reread before saving")
	}
	actual, readErr := io.ReadAll(io.LimitReader(tmp, fileReadLimit+1))
	afterStamp, afterErr := fileDescriptorStamp(tmp)
	info, statErr := tmp.Stat()
	if err != nil || statErr != nil || readErr != nil || afterErr != nil || stamp != afterStamp || !bytes.Equal(actual, content) {
		return fileContent{}, apiError(runtime.CodeFileSaveUncertain, "file replacement may have succeeded; reread before saving")
	}
	return fileContent{Path: display, Size: int64(len(content)), ModTime: info.ModTime().UTC().Format(time.RFC3339),
		Content: string(content), LineCount: countLines(string(content)), Language: languageForPath(display),
		ObservedRevision: opaqueFileRevision(stamp), Revision: textFileRevision(stamp, content), Editable: true}, nil
}
