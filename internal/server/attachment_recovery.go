package server

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/AsaphNoam/Chuck/internal/attachment"
	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
	"github.com/AsaphNoam/Chuck/internal/transcript"
)

// Cleanup is a startup-only pass before admission. A failed read preserves that
// conversation's artifacts rather than guessing at ownership and deleting data,
// and is logged rather than disabling attachment delivery for every conversation.
func (s *Server) recoverAttachmentStorage() error {
	store := attachment.New(s.configStore.Home())
	if err := store.CleanupTemps(); err != nil {
		return err
	}
	entries, err := os.ReadDir(filepath.Join(s.configStore.Home(), "sessions"))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		if err := s.recoverConversationAttachments(store, entry.Name()); err != nil {
			s.log.Error("recover conversation attachments; preserving its stored copies", "agent_id", entry.Name(), "err", err)
		}
	}
	return nil
}

func (s *Server) recoverConversationAttachments(store *attachment.Store, id string) error {
	keep := map[string]bool{}
	if strings.HasPrefix(id, "tt-") {
		room, err := s.stateStore.ReadThinkTankByCommandID(strings.TrimPrefix(id, "tt-"))
		if err != nil {
			if !errors.Is(err, state.ErrNotFound) {
				return err
			}
			return store.PruneUnowned(id, keep)
		}
		for _, descriptor := range room.Room.Attachments {
			keep[descriptor.ID] = true
		}
		return store.PruneUnowned(id, keep)
	}
	collect := func(raw []byte) error {
		var value any
		if err := json.Unmarshal(raw, &value); err != nil {
			return err
		}
		collectAttachmentIDs(value, keep)
		return nil
	}
	_, err := s.stateStore.ReadAgent(id)
	if err != nil && !errors.Is(err, state.ErrNotFound) {
		return err
	}
	if err == nil {
		err = transcript.ForEachFile(s.configStore.Home(), id, transcript.ReadOptions{OnOversized: func(int64) error { return errors.New("cannot enumerate oversized attachment ownership record") }}, func(ev runtime.Event) error { return collect(ev.Data) })
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		rows, err := s.stateStore.DB().Query(`SELECT message_json FROM attachment_receipts WHERE agent_id=?`, id)
		if err != nil {
			return err
		}
		for rows.Next() {
			var raw string
			if err = rows.Scan(&raw); err == nil {
				err = collect([]byte(raw))
			}
			if err != nil {
				rows.Close()
				return err
			}
		}
		if err = rows.Err(); err != nil {
			rows.Close()
			return err
		}
		if err = rows.Close(); err != nil {
			return err
		}
	}
	return store.PruneUnowned(id, keep)
}

func collectAttachmentIDs(value any, keep map[string]bool) {
	switch v := value.(type) {
	case map[string]any:
		if id, ok := v["id"].(string); ok {
			if _, ok = v["sha256"]; ok {
				keep[id] = true
			}
		}
		for _, child := range v {
			collectAttachmentIDs(child, keep)
		}
	case []any:
		for _, child := range v {
			collectAttachmentIDs(child, keep)
		}
	}
}
