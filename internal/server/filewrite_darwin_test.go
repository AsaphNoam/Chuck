package server

import (
	"os/exec"
	"path/filepath"
	"testing"
)

// macOS ACLs are separate from listxattr, so the real kernel ACL is the oracle.
func TestFileViewerDarwinACLRefused(t *testing.T) {
	path := filepath.Join(t.TempDir(), "acl.txt")
	writeFile(t, path, "original")
	if out, err := exec.Command("/bin/chmod", "+a", "everyone allow read", path).CombinedOutput(); err != nil {
		t.Fatalf("create ACL: %v %s", err, out)
	}
	got, ae := readLocalFile(path, "acl.txt")
	if ae != nil {
		t.Fatal(ae)
	}
	if got.Editable || got.EditRefusal != "unsupported_metadata" {
		t.Fatalf("ACL admission: %+v", got)
	}
	_, ae = saveLocalFile(path, "acl.txt", got.Revision, []byte("replacement"), nil)
	if ae == nil || ae.Code != "file_not_editable" {
		t.Fatalf("ACL save: %v", ae)
	}
}
