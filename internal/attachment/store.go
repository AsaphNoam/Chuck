// Package attachment stores immutable, conversation-owned upload bytes.
package attachment

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"unicode/utf8"
)

const (
	MaxFiles      = 10
	MaxFileBytes  = 10 << 20
	MaxTotalBytes = 25 << 20
	MaxPixels     = 40_000_000
	MaxNameBytes  = 255
)

const (
	KindImage = "image"
	KindFile  = "file"
)

var (
	ErrInvalidAgent  = errors.New("attachment: invalid conversation id")
	ErrInvalidID     = errors.New("attachment: invalid attachment id")
	ErrInvalidName   = errors.New("attachment: invalid display name")
	ErrFileTooLarge  = errors.New("attachment: file exceeds size limit")
	ErrTotalTooLarge = errors.New("attachment: message exceeds total size limit")
	ErrTooManyFiles  = errors.New("attachment: message exceeds file count limit")
	ErrInvalidImage  = errors.New("attachment: image content does not match its filename")
)

// Descriptor is safe display metadata; paths are always derived from opaque IDs.
type Descriptor struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	MIMEType string `json:"mime_type"`
	Size     int64  `json:"size"`
	SHA256   string `json:"sha256"`
	Kind     string `json:"kind"`
}

type Store struct {
	root string
	mu   sync.Mutex
}

func New(root string) *Store { return &Store{root: root} }

// Put streams an upload into a private temporary file, validates it, then
// atomically publishes its immutable blob and manifest.
func (s *Store) Put(agentID, name string, r io.Reader) (Descriptor, error) {
	if !validAgentID(agentID) {
		return Descriptor{}, ErrInvalidAgent
	}
	if err := validateName(name); err != nil {
		return Descriptor{}, err
	}
	if r == nil {
		return Descriptor{}, errors.New("attachment: missing upload")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	dir := s.agentDir(agentID)
	if err := ensureDirChain(s.root, filepath.Join("sessions", agentID, "attachments")); err != nil {
		return Descriptor{}, err
	}
	tmp, err := os.CreateTemp(dir, ".upload-*")
	if err != nil {
		return Descriptor{}, fmt.Errorf("attachment: create temporary file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return Descriptor{}, err
	}
	h := sha256.New()
	limited := &io.LimitedReader{R: r, N: MaxFileBytes + 1}
	n, copyErr := io.Copy(io.MultiWriter(tmp, h), limited)
	if copyErr != nil {
		tmp.Close()
		return Descriptor{}, fmt.Errorf("attachment: write upload: %w", copyErr)
	}
	if n > MaxFileBytes {
		tmp.Close()
		return Descriptor{}, ErrFileTooLarge
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return Descriptor{}, err
	}
	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		tmp.Close()
		return Descriptor{}, err
	}
	meta, err := inspect(tmp, name)
	if err != nil {
		tmp.Close()
		return Descriptor{}, err
	}
	if err := tmp.Close(); err != nil {
		return Descriptor{}, err
	}

	id, err := newID()
	if err != nil {
		return Descriptor{}, err
	}
	d := Descriptor{ID: id, Name: name, MIMEType: meta.mime, Size: n, SHA256: hex.EncodeToString(h.Sum(nil)), Kind: meta.kind}
	stage, err := os.MkdirTemp(dir, ".stage-*")
	if err != nil {
		return Descriptor{}, fmt.Errorf("attachment: stage upload: %w", err)
	}
	defer os.RemoveAll(stage)
	if err := os.Chmod(stage, 0o700); err != nil {
		return Descriptor{}, err
	}
	if err := os.Rename(tmpName, filepath.Join(stage, "blob")); err != nil {
		return Descriptor{}, err
	}
	if err := writeManifest(filepath.Join(stage, "manifest.json"), d); err != nil {
		return Descriptor{}, err
	}
	if err := writePending(stage); err != nil {
		return Descriptor{}, err
	}
	if err := syncDir(stage); err != nil {
		return Descriptor{}, err
	}
	if err := os.Rename(stage, filepath.Join(dir, id)); err != nil {
		return Descriptor{}, fmt.Errorf("attachment: publish upload: %w", err)
	}
	if err := syncDir(dir); err != nil {
		_ = os.RemoveAll(filepath.Join(dir, id))
		_ = syncDir(dir)
		return Descriptor{}, err
	}
	return d, nil
}

func (s *Store) Open(agentID, id string) (*os.File, Descriptor, error) {
	if !validAgentID(agentID) {
		return nil, Descriptor{}, ErrInvalidAgent
	}
	if !validID(id) {
		return nil, Descriptor{}, ErrInvalidID
	}
	base := filepath.Join(s.agentDir(agentID), id)
	if err := checkDirChain(s.root, filepath.Join("sessions", agentID, "attachments", id)); err != nil {
		return nil, Descriptor{}, err
	}
	d, err := readManifest(filepath.Join(base, "manifest.json"))
	if err != nil {
		return nil, Descriptor{}, err
	}
	if d.ID != id {
		return nil, Descriptor{}, errors.New("attachment: manifest identity mismatch")
	}
	f, err := safeOpen(filepath.Join(base, "blob"))
	if err != nil {
		return nil, Descriptor{}, err
	}
	if err := verifyBlob(f, d); err != nil {
		f.Close()
		return nil, Descriptor{}, err
	}
	return f, d, nil
}

func (s *Store) Path(agentID, id string) (string, error) {
	f, _, err := s.Open(agentID, id)
	if err != nil {
		return "", err
	}
	defer f.Close()
	return filepath.Join(s.agentDir(agentID), id, "blob"), nil
}

func (s *Store) Remove(agentID, id string) error {
	if !validAgentID(agentID) {
		return ErrInvalidAgent
	}
	if !validID(id) {
		return ErrInvalidID
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	dir := filepath.Join(s.agentDir(agentID), id)
	if err := checkDirChain(s.root, filepath.Join("sessions", agentID, "attachments", id)); err != nil {
		return err
	}
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	return syncDir(s.agentDir(agentID))
}

// PruneUnowned removes pre-acceptance artifacts abandoned by a crash. Call only
// before uploads/runtime output start, with ownership collected from durable
// transcript events and request receipts. Corrupt manifests fail closed.
func (s *Store) PruneUnowned(agentID string, keep map[string]bool) error {
	if !validAgentID(agentID) {
		return ErrInvalidAgent
	}
	dir := s.agentDir(agentID)
	if err := checkDirChain(s.root, filepath.Join("sessions", agentID, "attachments")); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !validID(entry.Name()) {
			continue
		}
		if keep[entry.Name()] {
			if err := s.Retain(agentID, []string{entry.Name()}); err != nil {
				return err
			}
			continue
		}
		if _, err := os.Lstat(filepath.Join(dir, entry.Name(), "pending")); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return err
		}
		f, _, err := s.Open(agentID, entry.Name())
		if err != nil {
			return err
		}
		f.Close()
		if err := s.Remove(agentID, entry.Name()); err != nil {
			return err
		}
	}
	return nil
}

func writePending(dir string) error {
	f, err := os.OpenFile(filepath.Join(dir, "pending"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	err = f.Sync()
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	return err
}

// Retain follows durable receipt/event publication. A missing pending marker is
// already retained; startup can only prune still-provisional unreferenced blobs.
func (s *Store) Retain(agentID string, ids []string) error {
	for _, id := range ids {
		f, _, err := s.Open(agentID, id)
		if err != nil {
			return err
		}
		f.Close()
		dir := filepath.Join(s.agentDir(agentID), id)
		if err := os.Remove(filepath.Join(dir, "pending")); err != nil && !os.IsNotExist(err) {
			return err
		}
		if err := syncDir(dir); err != nil {
			return err
		}
	}
	return nil
}

// DeleteConversation removes only conversation-owned copies. Original uploads
// and project files never belong to this directory.
func (s *Store) DeleteConversation(agentID string) error {
	if !validAgentID(agentID) {
		return ErrInvalidAgent
	}
	if err := checkDirChain(s.root, filepath.Join("sessions", agentID, "attachments")); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if err := os.RemoveAll(s.agentDir(agentID)); err != nil {
		return err
	}
	return syncDir(filepath.Join(s.root, "sessions", agentID))
}

// CleanupTemps removes abandoned upload and clone staging entries after a
// process restart. Call it once during startup, before accepting new uploads.
func (s *Store) CleanupTemps() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sessions := filepath.Join(s.root, "sessions")
	if err := checkDir(s.root); err != nil {
		return err
	}
	if err := checkDir(sessions); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	agents, err := os.ReadDir(sessions)
	if err != nil {
		return err
	}
	for _, agent := range agents {
		if !validAgentID(agent.Name()) {
			continue
		}
		rel := filepath.Join("sessions", agent.Name(), "attachments")
		if err := checkDirChain(s.root, rel); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return err
		}
		dir := filepath.Join(s.root, rel)
		entries, err := os.ReadDir(dir)
		if err != nil {
			return err
		}
		changed := false
		for _, entry := range entries {
			name := entry.Name()
			if !strings.HasPrefix(name, ".upload-") && !strings.HasPrefix(name, ".stage-") && !strings.HasPrefix(name, ".clone-") {
				continue
			}
			if err := os.RemoveAll(filepath.Join(dir, name)); err != nil {
				return err
			}
			changed = true
		}
		if changed {
			if err := syncDir(dir); err != nil {
				return err
			}
		}
	}
	return nil
}

// Clone copies only the requested owned attachments and preserves IDs while
// creating independent files under the destination conversation.
func (s *Store) Clone(source, target string, ids []string) error {
	if !validAgentID(source) || !validAgentID(target) || source == target {
		return ErrInvalidAgent
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ensureDirChain(s.root, filepath.Join("sessions", target, "attachments")); err != nil {
		return err
	}
	dest := s.agentDir(target)
	created := make([]string, 0, len(ids))
	rollback := func() {
		for _, p := range created {
			_ = os.RemoveAll(p)
		}
	}
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		if !validID(id) || seen[id] {
			rollback()
			return ErrInvalidID
		}
		seen[id] = true
		f, d, err := s.openUnlocked(source, id)
		if err != nil {
			rollback()
			return err
		}
		stage, err := os.MkdirTemp(dest, ".clone-*")
		if err != nil {
			f.Close()
			rollback()
			return err
		}
		_ = os.Chmod(stage, 0o700)
		out, err := os.OpenFile(filepath.Join(stage, "blob"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err == nil {
			copiedHash := sha256.New()
			var copied int64
			copied, err = io.Copy(io.MultiWriter(out, copiedHash), io.LimitReader(f, MaxFileBytes+1))
			if err == nil && (copied != d.Size || hex.EncodeToString(copiedHash.Sum(nil)) != d.SHA256) {
				err = errors.New("attachment: source changed during clone")
			}
		}
		f.Close()
		if err == nil {
			err = out.Sync()
		}
		if out != nil {
			if closeErr := out.Close(); err == nil {
				err = closeErr
			}
		}
		if err != nil {
			os.RemoveAll(stage)
			rollback()
			return err
		}
		if err = writeManifest(filepath.Join(stage, "manifest.json"), d); err != nil {
			os.RemoveAll(stage)
			rollback()
			return err
		}
		if err = writePending(stage); err != nil {
			os.RemoveAll(stage)
			rollback()
			return err
		}
		if err = syncDir(stage); err != nil {
			os.RemoveAll(stage)
			rollback()
			return err
		}
		final := filepath.Join(dest, id)
		if _, err = os.Lstat(final); err == nil {
			os.RemoveAll(stage)
			rollback()
			return errors.New("attachment: clone target already owns this id")
		}
		if err = os.Rename(stage, final); err != nil {
			os.RemoveAll(stage)
			rollback()
			return err
		}
		created = append(created, final)
	}
	if err := syncDir(dest); err != nil {
		rollback()
		return err
	}
	return nil
}

func (s *Store) openUnlocked(agentID, id string) (*os.File, Descriptor, error) {
	if !validAgentID(agentID) {
		return nil, Descriptor{}, ErrInvalidAgent
	}
	if !validID(id) {
		return nil, Descriptor{}, ErrInvalidID
	}
	base := filepath.Join(s.agentDir(agentID), id)
	if err := checkDirChain(s.root, filepath.Join("sessions", agentID, "attachments", id)); err != nil {
		return nil, Descriptor{}, err
	}
	d, err := readManifest(filepath.Join(base, "manifest.json"))
	if err != nil {
		return nil, Descriptor{}, err
	}
	if d.ID != id {
		return nil, Descriptor{}, errors.New("attachment: manifest identity mismatch")
	}
	f, err := safeOpen(filepath.Join(base, "blob"))
	if err != nil {
		return nil, Descriptor{}, err
	}
	if err := verifyBlob(f, d); err != nil {
		f.Close()
		return nil, Descriptor{}, err
	}
	return f, d, nil
}

func verifyBlob(f *os.File, d Descriptor) error {
	st, err := f.Stat()
	if err != nil || !st.Mode().IsRegular() || st.Size() != d.Size {
		return errors.New("attachment: invalid stored blob")
	}
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(f, MaxFileBytes+1))
	if err != nil || n != d.Size || hex.EncodeToString(h.Sum(nil)) != d.SHA256 {
		return errors.New("attachment: stored blob failed integrity check")
	}
	_, err = f.Seek(0, io.SeekStart)
	return err
}

func (s *Store) agentDir(id string) string {
	return filepath.Join(s.root, "sessions", id, "attachments")
}

type imageInfo struct{ mime, kind string }

func inspect(r io.ReadSeeker, name string) (imageInfo, error) {
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return imageInfo{}, err
	}
	buf := make([]byte, 512)
	n, err := r.Read(buf)
	if err != nil && !errors.Is(err, io.EOF) {
		return imageInfo{}, err
	}
	buf = buf[:n]
	contentType := http.DetectContentType(buf)
	imgType := ""
	switch {
	case len(buf) >= 8 && string(buf[:8]) == "\x89PNG\r\n\x1a\n":
		imgType = "image/png"
	case len(buf) >= 3 && buf[0] == 0xff && buf[1] == 0xd8 && buf[2] == 0xff:
		imgType = "image/jpeg"
	case len(buf) >= 12 && string(buf[:4]) == "RIFF" && string(buf[8:12]) == "WEBP":
		imgType = "image/webp"
	case len(buf) >= 6 && (string(buf[:6]) == "GIF87a" || string(buf[:6]) == "GIF89a"):
		imgType = "image/gif"
	}
	ext := strings.ToLower(filepath.Ext(name))
	expected := map[string]string{".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".webp": "image/webp", ".gif": "image/gif"}[ext]
	if expected != "" && expected != imgType {
		return imageInfo{}, ErrInvalidImage
	}
	if imgType == "" {
		return imageInfo{mime: normalizeMIME(contentType), kind: KindFile}, nil
	}
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return imageInfo{}, err
	}
	config, _, err := imageDecodeConfig(imgType, r)
	if err != nil {
		return imageInfo{}, fmt.Errorf("attachment: invalid image: %w", err)
	}
	if config.Width <= 0 || config.Height <= 0 || int64(config.Width)*int64(config.Height) > MaxPixels {
		return imageInfo{}, errors.New("attachment: image exceeds pixel limit")
	}
	return imageInfo{mime: imgType, kind: KindImage}, nil
}

// Inspect validates a bounded input without retaining it. It is useful for an
// explicitly selected local image that should be previewed without storing a
// second copy. The returned descriptor has no storage ID.
func Inspect(r io.ReadSeeker, name string) (Descriptor, error) {
	if err := validateName(name); err != nil {
		return Descriptor{}, err
	}
	if r == nil {
		return Descriptor{}, errors.New("attachment: missing input")
	}
	start, err := r.Seek(0, io.SeekCurrent)
	if err != nil {
		return Descriptor{}, err
	}
	defer r.Seek(start, io.SeekStart)
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return Descriptor{}, err
	}
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(r, MaxFileBytes+1))
	if err != nil {
		return Descriptor{}, err
	}
	if n > MaxFileBytes {
		return Descriptor{}, ErrFileTooLarge
	}
	meta, err := inspect(r, name)
	if err != nil {
		return Descriptor{}, err
	}
	return Descriptor{Name: name, MIMEType: meta.mime, Size: n, SHA256: hex.EncodeToString(h.Sum(nil)), Kind: meta.kind}, nil
}

func imageDecodeConfig(mt string, r io.Reader) (image.Config, string, error) {
	switch mt {
	case "image/png":
		c, err := png.DecodeConfig(r)
		return c, "png", err
	case "image/jpeg":
		c, err := jpeg.DecodeConfig(r)
		return c, "jpeg", err
	case "image/gif":
		c, err := gif.DecodeConfig(r)
		return c, "gif", err
	case "image/webp":
		return decodeWebPConfig(r)
	default:
		return image.Config{}, "", errors.New("unsupported image")
	}
}

func decodeWebPConfig(r io.Reader) (image.Config, string, error) {
	b, err := io.ReadAll(io.LimitReader(r, 30))
	if err != nil {
		return image.Config{}, "", err
	}
	if len(b) < 30 || string(b[:4]) != "RIFF" || string(b[8:12]) != "WEBP" {
		return image.Config{}, "", errors.New("invalid WebP header")
	}
	var w, h uint32
	switch string(b[12:16]) {
	case "VP8X":
		w = 1 + uint32(b[24]) + uint32(b[25])<<8 + uint32(b[26])<<16
		h = 1 + uint32(b[27]) + uint32(b[28])<<8 + uint32(b[29])<<16
	case "VP8 ":
		if b[23] != 0x9d || b[24] != 0x01 || b[25] != 0x2a {
			return image.Config{}, "", errors.New("invalid VP8 frame header")
		}
		w = uint32(b[26]) | uint32(b[27]&0x3f)<<8
		h = uint32(b[28]) | uint32(b[29]&0x3f)<<8
	case "VP8L":
		if b[20] != 0x2f {
			return image.Config{}, "", errors.New("invalid VP8L header")
		}
		w = 1 + uint32(b[21]) + uint32(b[22]&0x3f)<<8
		h = 1 + uint32(b[22]>>6) + uint32(b[23])<<2 + uint32(b[24]&0x0f)<<10
	default:
		return image.Config{}, "", errors.New("unsupported WebP bitstream")
	}
	return image.Config{Width: int(w), Height: int(h)}, "webp", nil
}

func normalizeMIME(mt string) string {
	base, _, err := mime.ParseMediaType(mt)
	if err == nil && base != "" {
		return base
	}
	return "application/octet-stream"
}

func validateName(name string) error {
	if name == "" || !utf8.ValidString(name) || len(name) > MaxNameBytes {
		return ErrInvalidName
	}
	for _, r := range name {
		if r == '/' || r == '\\' || r == '\r' || r == '\n' || r == 0 || r < 0x20 || r == 0x7f {
			return ErrInvalidName
		}
	}
	return nil
}

var agentRE = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)
var idRE = regexp.MustCompile(`^[0-9a-f]{32}$`)

func validAgentID(id string) bool { return agentRE.MatchString(id) }
func validID(id string) bool      { return idRE.MatchString(id) }
func newID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func writeManifest(path string, d Descriptor) error {
	b, err := json.Marshal(d)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err = f.Write(b); err == nil {
		err = f.Sync()
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	return err
}

func readManifest(path string) (Descriptor, error) {
	f, err := safeOpen(path)
	if err != nil {
		return Descriptor{}, err
	}
	defer f.Close()
	var d Descriptor
	if err := json.NewDecoder(io.LimitReader(f, 4097)).Decode(&d); err != nil {
		return Descriptor{}, fmt.Errorf("attachment: read manifest: %w", err)
	}
	if !validID(d.ID) || validateName(d.Name) != nil || d.Size < 0 || d.Size > MaxFileBytes || (d.Kind != KindFile && d.Kind != KindImage) || d.MIMEType == "" || len(d.SHA256) != 64 || !validSHA256(d.SHA256) {
		return Descriptor{}, errors.New("attachment: invalid manifest")
	}
	if d.Kind == KindImage && !previewMIME(d.MIMEType) {
		return Descriptor{}, errors.New("attachment: invalid image manifest")
	}
	return d, nil
}

func validSHA256(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size && value == strings.ToLower(value)
}

func previewMIME(value string) bool {
	switch value {
	case "image/png", "image/jpeg", "image/webp", "image/gif":
		return true
	default:
		return false
	}
}

func safeOpen(path string) (*os.File, error) {
	before, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return nil, errors.New("attachment: refusing non-regular or symlink file")
	}
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_NONBLOCK|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fd), path)
	after, err := os.Lstat(path)
	opened, statErr := f.Stat()
	if err != nil || statErr != nil || !opened.Mode().IsRegular() || !os.SameFile(before, after) || !os.SameFile(opened, after) {
		f.Close()
		return nil, errors.New("attachment: file changed during open")
	}
	return f, nil
}

func ensureDirChain(root, relative string) error {
	if root == "" {
		return errors.New("attachment: empty storage root")
	}
	root = filepath.Clean(root)
	if err := ensureOneDir(root); err != nil {
		return err
	}
	cur := root
	for _, part := range strings.Split(relative, string(filepath.Separator)) {
		if part == "" || part == "." || part == ".." {
			return errors.New("attachment: invalid storage path")
		}
		cur = filepath.Join(cur, part)
		if err := ensureOneDir(cur); err != nil {
			return err
		}
	}
	return nil
}

func ensureOneDir(path string) error {
	fi, err := os.Lstat(path)
	if os.IsNotExist(err) {
		if err = os.Mkdir(path, 0o700); err != nil && !os.IsExist(err) {
			return err
		}
		fi, err = os.Lstat(path)
	}
	if err != nil {
		return err
	}
	if fi.Mode()&os.ModeSymlink != 0 || !fi.IsDir() {
		return errors.New("attachment: refusing symlink or non-directory storage path")
	}
	if err := os.Chmod(path, 0o700); err != nil {
		return err
	}
	return nil
}

func checkDir(path string) error {
	fi, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if fi.Mode()&os.ModeSymlink != 0 || !fi.IsDir() {
		return errors.New("attachment: refusing symlink or non-directory storage path")
	}
	return nil
}

func checkDirChain(root, relative string) error {
	if root == "" {
		return errors.New("attachment: empty storage root")
	}
	cur := filepath.Clean(root)
	if err := checkDir(cur); err != nil {
		return err
	}
	for _, part := range strings.Split(relative, string(filepath.Separator)) {
		if part == "" || part == "." || part == ".." {
			return errors.New("attachment: invalid storage path")
		}
		cur = filepath.Join(cur, part)
		if err := checkDir(cur); err != nil {
			return err
		}
	}
	return nil
}

func syncDir(path string) error {
	d, err := os.Open(path)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}
