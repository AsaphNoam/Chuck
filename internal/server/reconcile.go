package server

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/transcript"
)

const reconcileInterval = 30 * time.Second
const reconcileDebounce = 250 * time.Millisecond

func (s *Server) startReconciliationSweep(ctx context.Context) {
	go func() {
		sessionsDir := filepath.Join(s.configStore.Home(), "sessions")
		_ = os.MkdirAll(sessionsDir, 0o700)
		s.reconcileSessionsOnce(time.Now().Add(-reconcileInterval))
		s.pruneStaleRunning()

		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			s.log.Warn("reconcile: fsnotify unavailable", "err", err)
			s.reconcileSessionsTimer(ctx)
			return
		}
		defer watcher.Close()
		if err := watcher.Add(sessionsDir); err != nil {
			s.log.Warn("reconcile: watch sessions", "err", err)
		}

		ticker := time.NewTicker(reconcileInterval)
		defer ticker.Stop()
		var debounce *time.Timer
		var debounceC <-chan time.Time
		pending := make(map[string]struct{})
		for {
			select {
			case <-ctx.Done():
				return
			case ev := <-watcher.Events:
				if ev.Has(fsnotify.Create) || ev.Has(fsnotify.Write) {
					if info, err := os.Stat(ev.Name); err == nil && info.IsDir() {
						_ = watcher.Add(ev.Name)
					} else {
						pending[ev.Name] = struct{}{}
					}
					if debounce == nil {
						debounce = time.NewTimer(reconcileDebounce)
					} else {
						if !debounce.Stop() {
							select {
							case <-debounce.C:
							default:
							}
						}
						debounce.Reset(reconcileDebounce)
					}
					debounceC = debounce.C
				}
			case <-debounceC:
				staleBefore := time.Now().Add(-reconcileInterval)
				for path := range pending {
					s.reconcileSessionPath(path, staleBefore)
					delete(pending, path)
				}
				s.pruneStaleRunning()
				debounceC = nil
			case err := <-watcher.Errors:
				s.log.Warn("reconcile: watcher error", "err", err)
			case <-ticker.C:
				s.reconcileSessionsOnce(time.Now().Add(-reconcileInterval))
				s.pruneStaleRunning()
			}
		}
	}()
}

func (s *Server) reconcileSessionPath(path string, staleBefore time.Time) bool {
	sessionsDir := filepath.Join(s.configStore.Home(), "sessions")
	agentID := agentIDFromSessionPath(sessionsDir, path)
	if agentID == "" {
		return false
	}
	preview, err := lastAssistantPreviewFrom(path, nil)
	if err != nil {
		s.log.Warn("reconcile: read transcript", "agent", agentID, "err", err)
		return false
	}
	if _, ok, err := s.stateMgr.ApplyStaleCorrection(agentID, preview, staleBefore); err != nil {
		s.log.Debug("reconcile: correction skipped", "agent", agentID, "err", err)
		return false
	} else {
		return ok
	}
}

func (s *Server) reconcileSessionsTimer(ctx context.Context) {
	ticker := time.NewTicker(reconcileInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.reconcileSessionsOnce(time.Now().Add(-reconcileInterval))
			s.pruneStaleRunning()
		}
	}
}

func (s *Server) reconcileSessionsOnce(staleBefore time.Time) int {
	sessionsDir := filepath.Join(s.configStore.Home(), "sessions")
	applied := 0
	_ = filepath.WalkDir(sessionsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if s.reconcileSessionPath(path, staleBefore) {
			applied++
		}
		return nil
	})
	return applied
}

// agentIDFromSessionPath accepts only the canonical sessions/{a_id}/transcript.ndjson.
// Attachments and other session artifacts are never transcript truth (TS-01.R8).
func agentIDFromSessionPath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return ""
	}
	parts := strings.Split(rel, string(filepath.Separator))
	if len(parts) != 2 || parts[1] != transcript.LogFileName || !strings.HasPrefix(parts[0], "a_") {
		return ""
	}
	return parts[0]
}

// lastAssistantPreview derives a bounded, human-readable status preview from the
// assistant's text in the last turn of the NDJSON transcript. Each line is a
// normalized runtime.Event; assistant_text deltas are concatenated and reset at
// each turn boundary, so the result is the final assistant message (clipped to
// the last detailPreviewLimit runes) rather than a raw event envelope (§6.4/§13
// "last output line"). The tail is kept, not the head, so a card preview shows
// the most recently produced text of a long or still-streaming message — the
// same end the live client-side preview (`ui/src/store/transcriptStore.ts`)
// keeps, so a reload or reconcile sweep never flips which end of the message a
// card shows (INV §2). Returns "" when the transcript carries no assistant text, which
// leaves the existing status detail untouched (a non-text envelope is never a
// meaningful preview and must not clobber a healthy card — the BLOCKING fix).
// It streams the log with the shared tolerant reader and retains only a bounded
// tail of each reply (INV §16); retained reports that size for tests. A missing
// log is an empty preview; any other read failure is returned (INV §7).
func lastAssistantPreviewFrom(path string, retained func(int)) (string, error) {
	var completed, current string
	err := transcript.NewReader(path).ForEach(transcript.ReadOptions{}, func(ev runtime.Event) error {
		// A native child's text is not the agent's reply (FS-03.R58).
		if ev.ActivityID != "" {
			return nil
		}
		switch ev.Type {
		case runtime.EvAssistantText:
			var d runtime.AssistantTextData
			if json.Unmarshal(ev.Data, &d) == nil {
				current = keepTailBytes(current+d.Delta, previewRetainBytes)
				if retained != nil {
					retained(len(current))
				}
			}
		case runtime.EvTurnEnd:
			if strings.TrimSpace(current) != "" {
				completed = current
			}
			current = ""
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	preview := current
	if strings.TrimSpace(preview) == "" {
		preview = completed
	}
	return clipPreview(strings.TrimSpace(preview), detailPreviewLimit), nil
}

// previewRetainBytes bounds the reply tail a preview keeps while streaming: ample
// for detailPreviewLimit runes plus trailing whitespace trimmed before clipping.
const previewRetainBytes = 4096

// detailPreviewLimit bounds a reconciled status-detail preview (mirrors the
// chat runtime's clip(tail, 120) crash-path convention).
const detailPreviewLimit = 120

// clipPreview keeps at most the last n runes of s, so a status detail written
// from a transcript never grows unbounded, never splits a multi-byte rune
// (which would produce invalid UTF-8 in the status JSON), and shows the most
// recently produced text rather than a fixed prefix that never advances while
// a message is still streaming.
func clipPreview(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[len(r)-n:])
}

// pruneStaleRunning shares server start's exited-row reconciliation so both
// write FS-01.R20's outcome from one identity check (INV §2/§9).
func (s *Server) pruneStaleRunning() int {
	pruned, err := runtime.ReconcileStale(s.stateStore)
	if err != nil {
		s.log.Debug("liveness: reconcile running", "err", err)
	}
	for _, id := range pruned {
		if _, err := s.stateMgr.Touch(id); err != nil {
			s.log.Debug("liveness: touch", "agent", id, "err", err)
		}
	}
	return len(pruned)
}
