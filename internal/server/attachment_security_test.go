package server

import (
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"
	"testing"
)

func TestPhoneImageOpenRejectsSymlinkComponentsAndTraversal(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.png"), []byte("private outside bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "linked")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "secret.png"), filepath.Join(root, "leaf.png")); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{filepath.Join(root, "linked", "secret.png"), filepath.Join(root, "leaf.png"), filepath.Join(outside, "secret.png")} {
		if fd, err := openWorkspaceImage(root, path); err == nil {
			unix.Close(fd)
			t.Fatalf("opened forbidden path %s", path)
		}
	}
	if err := os.Mkdir(filepath.Join(root, "images"), 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "images", "regular.png")
	if err := os.WriteFile(path, []byte("bounded fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	fd, err := openWorkspaceImage(root, path)
	if err != nil {
		t.Fatal(err)
	}
	unix.Close(fd)
}
