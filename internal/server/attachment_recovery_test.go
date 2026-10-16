package server

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AsaphNoam/Chuck/internal/attachment"
	"github.com/AsaphNoam/Chuck/internal/state"
)

func TestAttachmentRecoveryKeepsCommittedThinkTankBlobs(t *testing.T) {
	srv := testServer(t, true)
	store := attachment.New(srv.configStore.Home())
	descriptor, err := store.Put(state.ThinkTankAttachmentKey("room-command"), "goal.txt", strings.NewReader("keep me"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := srv.stateStore.CreateThinkTank(state.ThinkTankCreate{
		CommandID: "room-command", Goal: "Decide", OriginProject: "alpha", Attachments: []attachment.Descriptor{descriptor},
		Members: []state.ThinkTankMember{{AgentID: "one", Cap: 1}, {AgentID: "two", Cap: 1}},
	}); err != nil {
		t.Fatal(err)
	}

	orphan, err := store.Put(state.ThinkTankAttachmentKey("orphan-command"), "orphan.txt", strings.NewReader("remove me"))
	if err != nil {
		t.Fatal(err)
	}
	if err := srv.recoverAttachmentStorage(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(srv.configStore.Home(), "sessions", state.ThinkTankAttachmentKey("room-command"), "attachments", descriptor.ID)); err != nil {
		t.Fatalf("committed room blob removed: %v", err)
	}
	f, _, err := store.Open(state.ThinkTankAttachmentKey("room-command"), descriptor.ID)
	if err != nil {
		t.Fatalf("committed room blob unreadable: %v", err)
	}
	contents, err := io.ReadAll(f)
	f.Close()
	if err != nil || string(contents) != "keep me" {
		t.Fatalf("committed room blob contents = %q, err=%v", contents, err)
	}
	if _, err := os.Stat(filepath.Join(srv.configStore.Home(), "sessions", state.ThinkTankAttachmentKey("orphan-command"), "attachments", orphan.ID)); !os.IsNotExist(err) {
		t.Fatalf("orphan room blob still present: %v", err)
	}
}
