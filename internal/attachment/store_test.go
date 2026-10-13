// FS-03.A65: immutable storage enforces file, format and decoding limits.
package attachment

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const agentA = "a_one"
const agentB = "session_two"

func TestPutOpenAndPrivateModes(t *testing.T) {
	s := New(t.TempDir())
	want := []byte("private upload")
	d, err := s.Put(agentA, "notes.txt", bytes.NewReader(want))
	if err != nil {
		t.Fatal(err)
	}
	if d.Kind != KindFile || d.MIMEType == "" || d.Size != int64(len(want)) || len(d.SHA256) != 64 {
		t.Fatalf("descriptor = %+v", d)
	}
	f, got, err := s.Open(agentA, d.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	body, err := io.ReadAll(f)
	if err != nil || !bytes.Equal(body, want) || got != d {
		t.Fatalf("open = %q, %+v, %v", body, got, err)
	}
	for _, p := range []string{filepath.Dir(filepath.Dir(filepath.Dir(s.agentDir(agentA)))), filepath.Dir(s.agentDir(agentA)), s.agentDir(agentA), filepath.Join(s.agentDir(agentA), d.ID)} {
		info, err := os.Stat(p)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o700 {
			t.Errorf("directory mode %s = %o", p, info.Mode().Perm())
		}
	}
	for _, p := range []string{filepath.Join(s.agentDir(agentA), d.ID, "blob"), filepath.Join(s.agentDir(agentA), d.ID, "manifest.json")} {
		info, err := os.Stat(p)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Errorf("file mode %s = %o", p, info.Mode().Perm())
		}
	}
}

func TestPutValidatesBoundsAndNames(t *testing.T) {
	s := New(t.TempDir())
	for _, name := range []string{"", "../x", "a\\b", "a\r\nb", strings.Repeat("x", MaxNameBytes+1), string([]byte{0xff})} {
		if _, err := s.Put(agentA, name, strings.NewReader("x")); !errors.Is(err, ErrInvalidName) {
			t.Errorf("Put(%q) error = %v", name, err)
		}
	}
	if _, err := s.Put("../bad", "x.txt", strings.NewReader("x")); !errors.Is(err, ErrInvalidAgent) {
		t.Fatalf("invalid agent = %v", err)
	}
	if _, err := s.Put(agentA, "large.txt", bytes.NewReader(make([]byte, MaxFileBytes+1))); !errors.Is(err, ErrFileTooLarge) {
		t.Fatalf("oversized = %v", err)
	}
	for i := 0; i < MaxFiles+1; i++ {
		if _, err := s.Put(agentA, "f.txt", strings.NewReader("x")); err != nil {
			t.Fatalf("Put #%d: %v", i, err)
		}
	}
}

func TestPutAllowsConversationHistoryBeyondMessageLimits(t *testing.T) {
	s := New(t.TempDir())
	payload := bytes.Repeat([]byte("x"), MaxFileBytes)
	ids := make([]string, 0, 3)
	for i := 0; i < 3; i++ {
		d, err := s.Put(agentA, "large.txt", bytes.NewReader(payload))
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, d.ID)
	}
	for i := 0; i < MaxFiles+2; i++ {
		d, err := s.Put(agentA, "small.txt", strings.NewReader("x"))
		if err != nil {
			t.Fatalf("historical item #%d: %v", i, err)
		}
		ids = append(ids, d.ID)
	}
	if err := s.Clone(agentA, agentB, ids); err != nil {
		t.Fatalf("clone history exceeding request size limits: %v", err)
	}
}

func TestPutAndOpenAllowZeroByteFiles(t *testing.T) {
	s := New(t.TempDir())
	d, err := s.Put(agentA, "empty.txt", strings.NewReader(""))
	if err != nil || d.Size != 0 || d.Kind != KindFile {
		t.Fatalf("Put empty = %+v, %v", d, err)
	}
	f, _, err := s.Open(agentA, d.ID)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
}

func TestImageSniffAndPixelLimit(t *testing.T) {
	s := New(t.TempDir())
	var valid bytes.Buffer
	if err := png.Encode(&valid, image.NewRGBA(image.Rect(0, 0, 3, 2))); err != nil {
		t.Fatal(err)
	}
	d, err := s.Put(agentA, "image.png", bytes.NewReader(valid.Bytes()))
	if err != nil || d.Kind != KindImage || d.MIMEType != "image/png" {
		t.Fatalf("valid image = %+v, %v", d, err)
	}
	if _, err := s.Put(agentA, "spoof.jpg", strings.NewReader("plain text")); !errors.Is(err, ErrInvalidImage) {
		t.Fatalf("spoofed image = %v", err)
	}
	if _, err := s.Put(agentA, "ordinary.bin", strings.NewReader("<svg></svg>")); err != nil {
		t.Fatalf("ordinary file: %v", err)
	}
	if _, _, err := decodeWebPConfig(bytes.NewReader([]byte("not webp"))); err == nil {
		t.Fatal("accepted malformed WebP")
	}
}

func TestInspectValidatesWithoutPersisting(t *testing.T) {
	var img bytes.Buffer
	if err := png.Encode(&img, image.NewRGBA(image.Rect(0, 0, 3, 2))); err != nil {
		t.Fatal(err)
	}
	input := bytes.NewReader(img.Bytes())
	if _, err := input.Seek(5, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	d, err := Inspect(input, "local.png")
	if err != nil || d.ID != "" || d.Kind != KindImage || d.MIMEType != "image/png" {
		t.Fatalf("Inspect = %+v, %v", d, err)
	}
	pos, _ := input.Seek(0, io.SeekCurrent)
	if pos != 5 {
		t.Fatalf("Inspect changed input position to %d", pos)
	}
	if _, err := Inspect(bytes.NewReader(make([]byte, MaxFileBytes+1)), "local.bin"); !errors.Is(err, ErrFileTooLarge) {
		t.Fatalf("oversized Inspect = %v", err)
	}
}

func TestOpenRejectsSymlinkSubstitution(t *testing.T) {
	s := New(t.TempDir())
	d, err := s.Put(agentA, "secret.txt", strings.NewReader("owned"))
	if err != nil {
		t.Fatal(err)
	}
	blob := filepath.Join(s.agentDir(agentA), d.ID, "blob")
	if err := os.Remove(blob); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(outside, []byte("outside"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, blob); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, _, err := s.Open(agentA, d.ID); err == nil {
		t.Fatal("opened substituted symlink")
	}
}

func TestOpenRejectsSymlinkedConversationParent(t *testing.T) {
	root := t.TempDir()
	s := New(root)
	d, err := s.Put(agentA, "secret.txt", strings.NewReader("owned"))
	if err != nil {
		t.Fatal(err)
	}
	sessions := filepath.Join(root, "sessions")
	backup := filepath.Join(root, "sessions-backup")
	if err := os.Rename(sessions, backup); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(backup, sessions); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, _, err := s.Open(agentA, d.ID); err == nil {
		t.Fatal("opened through substituted conversation parent")
	}
	if err := s.Remove(agentA, d.ID); err == nil {
		t.Fatal("removed through substituted conversation parent")
	}
	if _, err := os.Stat(filepath.Join(backup, agentA, "attachments", d.ID, "blob")); err != nil {
		t.Fatalf("substituted source was affected: %v", err)
	}
}

func TestRejectsSymlinkedStorageParent(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "sessions")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := New(root).Put(agentA, "x.txt", strings.NewReader("x")); err == nil {
		t.Fatal("wrote through symlinked sessions directory")
	}
}

func TestCleanupTempsRemovesOnlyStagingEntries(t *testing.T) {
	s := New(t.TempDir())
	if _, err := s.Put(agentA, "kept.txt", strings.NewReader("keep")); err != nil {
		t.Fatal(err)
	}
	dir := s.agentDir(agentA)
	for _, name := range []string{".upload-abandoned", ".stage-abandoned", ".clone-abandoned"} {
		if strings.HasPrefix(name, ".upload-") {
			if err := os.WriteFile(filepath.Join(dir, name), []byte("temp"), 0o600); err != nil {
				t.Fatal(err)
			}
		} else if err := os.Mkdir(filepath.Join(dir, name), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	external := filepath.Join(t.TempDir(), "keep")
	if err := os.WriteFile(external, []byte("external"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, filepath.Join(dir, ".stage-link")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if err := s.CleanupTemps(); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries after cleanup = %v", entries)
	}
	if _, err := os.Stat(external); err != nil {
		t.Fatalf("cleanup followed staging symlink: %v", err)
	}
}

func TestCloneCopiesOnlySelectedBlobsIndependently(t *testing.T) {
	s := New(t.TempDir())
	selected, err := s.Put(agentA, "chosen.txt", strings.NewReader("selected bytes"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Put(agentA, "omitted.txt", strings.NewReader("do not copy")); err != nil {
		t.Fatal(err)
	}
	if err := s.Clone(agentA, agentB, []string{selected.ID}); err != nil {
		t.Fatal(err)
	}
	clonePath, err := s.Path(agentB, selected.ID)
	if err != nil {
		t.Fatal(err)
	}
	if sourcePath, _ := s.Path(agentA, selected.ID); sourcePath == clonePath {
		t.Fatal("clone shares source path")
	}
	if _, _, err := s.Open(agentB, "00000000000000000000000000000000"); err == nil {
		t.Fatal("opened unowned attachment")
	}
	if err := s.Remove(agentA, selected.ID); err != nil {
		t.Fatal(err)
	}
	f, cloned, err := s.Open(agentB, selected.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	body, err := io.ReadAll(f)
	if err != nil || string(body) != "selected bytes" || cloned.ID != selected.ID {
		t.Fatalf("clone bytes = %q, %+v, %v", body, cloned, err)
	}
	if err := s.Clone(agentA, agentB, []string{selected.ID}); err == nil {
		t.Fatal("clone from missing source succeeded")
	}
}
