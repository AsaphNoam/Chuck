package contextref

import (
	"context"
	"errors"
	"math"
	"strconv"
	"strings"

	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
	"github.com/AsaphNoam/Chuck/internal/transcript"
)

// PeerReadResult is one bounded page of a direct peer transcript read
// (FS-06.R39, TS-04.R93). Text is the same rendering read_context_link
// produces for a transcript span.
type PeerReadResult struct {
	AgentID         string             `json:"agent_id"`
	Source          state.HistoryAgent `json:"source"`
	SnapshotLastSeq int64              `json:"snapshot_last_seq"`
	Text            string             `json:"text"`
	Complete        bool               `json:"complete"`
	NextCursor      string             `json:"next_cursor,omitempty"`
}

// peerPosition is everything a peer cursor carries: the fixed read boundary
// and the rendered byte offset. It names a position, never authority.
type peerPosition struct {
	since   int64
	lastSeq int64
	prefix  int64
	offset  int
}

const peerCursorVersion = "p1"

// ReadPeerTranscript renders one page of another agent's recorded
// conversation. Authority is the caller's live MCP session (TS-05.R27): no
// grant is consulted or created, and the read wakes nothing, sends no mail,
// and changes no read state. The first page fixes the boundary at the
// transcript's current complete prefix; later activity needs a new read.
func (s *Service) ReadPeerTranscript(ctx context.Context, agentID string, sinceSeq int64, cursor string) (PeerReadResult, error) {
	if strings.TrimSpace(agentID) == "" {
		return PeerReadResult{}, failf(CodeValidation, "agent_id is required.")
	}
	if sinceSeq < 0 {
		return PeerReadResult{}, failf(CodeValidation, "since_seq must not be negative.")
	}
	if cursor != "" && sinceSeq != 0 {
		return PeerReadResult{}, failf(CodeValidation, "since_seq cannot accompany a cursor.")
	}
	source, err := s.store.ReadHistoryAgent(agentID)
	if errors.Is(err, state.ErrNotFound) {
		return PeerReadResult{}, &Error{Code: CodeAgentNotFound, Message: "No Chuck agent has that agent_id."}
	}
	if err != nil {
		return PeerReadResult{}, unavailable(err)
	}
	select {
	case s.peerScans <- struct{}{}:
		defer func() { <-s.peerScans }()
	default:
		return PeerReadResult{}, failf(CodePeerBusy, "Too many peer transcript reads are in progress.")
	}

	var pos peerPosition
	if cursor != "" {
		var cerr *Error
		if pos, cerr = decodePeerCursor(agentID, cursor); cerr != nil {
			return PeerReadResult{}, cerr
		}
	} else {
		prefix, err := transcript.CompletePrefix(s.home, agentID)
		if err != nil {
			return PeerReadResult{}, transcriptUnavailable()
		}
		pos = peerPosition{since: sinceSeq, lastSeq: -1, prefix: prefix}
	}

	out := newPageWriter(pos.offset, MaxPageBytes)
	// The byte prefix, not a sequence, bounds the read, so every page renders
	// the identical stream and offsets stay stable while the source grows.
	renderer := newTranscriptRenderer(state.ContextSource{FirstSeq: pos.since + 1, LastSeq: math.MaxInt64}, out)
	var maxSeq int64
	opts := transcript.ReadOptions{IncludeMeta: true, OnOversized: renderer.oversized}
	err = transcript.ForEachFilePrefix(s.home, agentID, pos.prefix, opts, func(ev runtime.Event) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if ev.Seq > maxSeq {
			maxSeq = ev.Seq
		}
		return renderer.event(ev)
	})
	if err != nil {
		return PeerReadResult{}, transcriptUnavailable()
	}
	renderer.done()
	if pos.since == 0 && renderer.adjacent {
		// A skipped record at the very end of a whole-history read has no later
		// record to anchor it, but it is still inside the read (INV §7).
		renderer.emit(OversizedRecordMarker + "\n\n")
	}
	if pos.lastSeq >= 0 && maxSeq != pos.lastSeq {
		return PeerReadResult{}, transcriptUnavailable()
	}
	if !out.validOffset() {
		return PeerReadResult{}, failf(CodeInvalidCursor, "Cursor does not name a position in this transcript.")
	}
	text, next, complete := out.page()
	result := PeerReadResult{
		AgentID:         agentID,
		Source:          source,
		SnapshotLastSeq: maxSeq,
		Text:            text,
		Complete:        complete,
	}
	if !complete {
		pos.lastSeq, pos.offset = maxSeq, next
		result.NextCursor = encodePeerCursor(agentID, pos)
	}
	return result, nil
}

func transcriptUnavailable() *Error {
	return &Error{Code: CodeTranscriptUnavailable,
		Message: "Chuck could not read that agent's recorded transcript."}
}

func encodePeerCursor(agentID string, p peerPosition) string {
	return encodeOpaque(strings.Join([]string{peerCursorVersion, agentID,
		strconv.FormatInt(p.since, 10), strconv.FormatInt(p.lastSeq, 10),
		strconv.FormatInt(p.prefix, 10), strconv.Itoa(p.offset)}, "\x00"))
}

func decodePeerCursor(agentID, cursor string) (peerPosition, *Error) {
	malformed := failf(CodeInvalidCursor, "Cursor is malformed.")
	if len(cursor) > MaxCursorBytes {
		return peerPosition{}, malformed
	}
	raw, ok := decodeOpaque(cursor)
	if !ok {
		return peerPosition{}, malformed
	}
	parts := strings.Split(raw, "\x00")
	if len(parts) != 6 || parts[0] != peerCursorVersion {
		return peerPosition{}, malformed
	}
	if parts[1] != agentID {
		return peerPosition{}, failf(CodeInvalidCursor, "Cursor does not belong to this agent's transcript.")
	}
	var nums [4]int64
	for i, part := range parts[2:] {
		n, err := strconv.ParseInt(part, 10, 64)
		if err != nil || n < 0 {
			return peerPosition{}, malformed
		}
		nums[i] = n
	}
	// Every issued cursor names a non-empty prefix with content after a
	// positive offset; anything else was forged or corrupted.
	if nums[2] == 0 || nums[3] == 0 || nums[3] > math.MaxInt32 {
		return peerPosition{}, malformed
	}
	return peerPosition{since: nums[0], lastSeq: nums[1], prefix: nums[2], offset: int(nums[3])}, nil
}
