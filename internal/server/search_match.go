package server

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/AsaphNoam/Chuck/internal/index"
	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/transcript"
)

const maxSearchDocumentBytes = 4 << 20
const maxSearchSourceBytes = 32 << 20

// Admission is process-wide, non-blocking and separate for the two read paths.
var projectSearchReads = make(chan struct{}, 2)
var searchMatchReads = make(chan struct{}, 2)

type searchSourceRange struct {
	Seq       int64 `json:"seq"`
	PartIndex int   `json:"part_index"`
	StartByte int   `json:"start_byte"`
	EndByte   int   `json:"end_byte"`
}

type searchMatchResponse struct {
	Status string              `json:"status"`
	Reason string              `json:"reason,omitempty"`
	Ranges []searchSourceRange `json:"ranges"`
}

func unavailableSearchMatch(reason string) searchMatchResponse {
	return searchMatchResponse{Status: "unavailable", Reason: reason, Ranges: []searchSourceRange{}}
}

func (s *Server) handleSearchMatch(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	q := r.URL.Query()
	doc, digest := q.Get("document_id"), q.Get("sha256")
	start, e1 := strconv.Atoi(q.Get("start"))
	end, e2 := strconv.Atoi(q.Get("end"))
	_, hashErr := hex.DecodeString(digest)
	_, _, knownDoc := searchDocumentBoundary(doc)
	if !knownDoc || len(doc) > 128 || !utf8.ValidString(doc) || len(digest) != 64 || hashErr != nil || e1 != nil || e2 != nil || start < 0 || end <= start {
		writeAPIError(w, apiError(runtime.CodeValidation, "invalid search passage reference"))
		return
	}
	select {
	case searchMatchReads <- struct{}{}:
		defer func() { <-searchMatchReads }()
	default:
		writeAPIError(w, apiError(runtime.CodeInternal, "Passage lookup is busy. Retry shortly."))
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	// Establish identity from authoritative state before any filesystem access,
	// under the same cancellation/read budget as the selected document.
	var known int
	if err := s.stateStore.DB().QueryRowContext(ctx, `SELECT 1 FROM sessions WHERE agent_id=?`, id).Scan(&known); errors.Is(err, sql.ErrNoRows) {
		writeAPIError(w, apiError(runtime.CodeNotFound, "chat not found"))
		return
	} else if err != nil {
		writeAPIError(w, apiError(runtime.CodeInternal, "cannot read chat"))
		return
	}
	resp, validation, err := s.resolveSearchMatch(ctx, id, doc, strings.ToLower(digest), start, end)
	if validation {
		writeAPIError(w, apiError(runtime.CodeValidation, "invalid search passage offsets"))
	} else if err != nil {
		writeAPIError(w, apiError(runtime.CodeInternal, "Cannot read the matching passage. Retry."))
	} else {
		writeJSON(w, http.StatusOK, resp)
	}
}

func (s *Server) resolveSearchMatch(ctx context.Context, id, doc, digest string, start, end int) (searchMatchResponse, bool, error) {
	db := s.stateStore.DB()
	var size int
	err := db.QueryRowContext(ctx, `SELECT length(CAST(content AS BLOB)) FROM sessions_fts WHERE agent_id=? AND document_id=?`, id, doc).Scan(&size)
	if errors.Is(err, sql.ErrNoRows) {
		return unavailableSearchMatch("The indexed passage is no longer available."), false, nil
	}
	if err != nil {
		return searchMatchResponse{}, false, err
	}
	if end > size {
		return searchMatchResponse{}, true, nil
	}
	if size > maxSearchDocumentBytes {
		return unavailableSearchMatch("This passage exceeds the lookup size limit."), false, nil
	}
	var content string
	if err := db.QueryRowContext(ctx, `SELECT content FROM sessions_fts WHERE agent_id=? AND document_id=?`, id, doc).Scan(&content); err != nil {
		return searchMatchResponse{}, false, err
	}
	if end > len(content) || !utf8.ValidString(content[:start]) || !utf8.ValidString(content[:end]) {
		return searchMatchResponse{}, true, nil
	}
	if index.ContentSHA256(content) != digest {
		return unavailableSearchMatch("The indexed passage has changed. Search again."), false, nil
	}
	var version, provenanceSize int
	var storedDigest string
	err = db.QueryRowContext(ctx, `SELECT version,content_sha256,length(CAST(spans_json AS BLOB)) FROM session_search_spans WHERE agent_id=? AND document_id=?`, id, doc).Scan(&version, &storedDigest, &provenanceSize)
	var provenance *index.SearchProvenance
	if err == nil {
		if version != 1 || storedDigest != digest || provenanceSize > index.MaxSearchProvenanceBytes {
			return unavailableSearchMatch("The stored passage location is incompatible."), false, nil
		}
		var raw string
		if err := db.QueryRowContext(ctx, `SELECT spans_json FROM session_search_spans WHERE agent_id=? AND document_id=?`, id, doc).Scan(&raw); err != nil {
			return searchMatchResponse{}, false, err
		}
		var p index.SearchProvenance
		if decodeSearchProvenance(raw, &p) != nil {
			return unavailableSearchMatch("The stored passage location is invalid."), false, nil
		}
		provenance = &p
	} else if !errors.Is(err, sql.ErrNoRows) {
		return searchMatchResponse{}, false, err
	}
	return s.resolveSearchSource(ctx, id, doc, content, provenance, start, end)
}

var errSearchSourceLimit = errors.New("search source limit")
var errSearchSourceDone = errors.New("search source complete")

type searchSourceBudget struct {
	ctx       context.Context
	remaining int
}

func (b *searchSourceBudget) Write(p []byte) (int, error) {
	if err := b.ctx.Err(); err != nil {
		return 0, err
	}
	if len(p) > b.remaining {
		return 0, errSearchSourceLimit
	}
	b.remaining -= len(p)
	return len(p), nil
}

// readSearchSource uses the canonical bounded record reader and a captured
// complete prefix; no source path or query text comes from the reference.
func (s *Server) readSearchSource(ctx context.Context, id string, visit func(runtime.Event) error) error {
	budget := &searchSourceBudget{ctx: ctx, remaining: maxSearchSourceBytes}
	prefix, err := transcript.CompletePrefixWithBudget(s.configStore.Home(), id, budget)
	if err != nil {
		return err
	}
	if prefix == 0 {
		return transcript.ErrPrefixUnavailable
	}
	return transcript.ForEachFilePrefix(s.configStore.Home(), id, prefix, transcript.ReadOptions{
		IncludeMeta: true, Tee: budget,
		OnOversized: func(int64) error { return errSearchSourceLimit },
	}, func(ev runtime.Event) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		return visit(ev)
	})
}

// Keep io's failure distinction visible to callers rather than treating an
// actual read error as a successful unavailable result.
func searchSourceUnavailable(err error) bool {
	return errors.Is(err, errSearchSourceLimit) || errors.Is(err, transcript.ErrPrefixUnavailable) || errors.Is(err, io.EOF)
}

func (s *Server) resolveSearchSource(ctx context.Context, id, doc, content string, p *index.SearchProvenance, start, end int) (searchMatchResponse, bool, error) {
	if p == nil {
		var builder index.SearchDocumentBuilder
		kind, boundary, ok := searchDocumentBoundary(doc)
		if !ok {
			return unavailableSearchMatch("This older passage has no compatible source location."), false, nil
		}
		found := kind == "legacy"
		err := s.readSearchSource(ctx, id, func(ev runtime.Event) error {
			if kind != "legacy" && ev.Seq > boundary {
				return errSearchSourceDone
			}
			if err := builder.AddEvent(ev); err != nil {
				return errSearchSourceLimit
			}
			// Indexed document admission also bounds transient text accumulation.
			if len(builder.Document().Content) > maxSearchDocumentBytes {
				return errSearchSourceLimit
			}
			if kind != "legacy" && ev.Seq == boundary {
				found = true
				return errSearchSourceDone
			}
			if kind != "legacy" && (ev.Type == runtime.EvTurnEnd || ev.Type == runtime.EvAnnotation) {
				builder = index.SearchDocumentBuilder{}
			}
			return nil
		})
		if err != nil && !errors.Is(err, errSearchSourceDone) {
			if searchSourceUnavailable(err) || errors.Is(err, context.DeadlineExceeded) {
				return unavailableSearchMatch("The retained source is unavailable or exceeds the lookup limit."), false, nil
			}
			return searchMatchResponse{}, false, err
		}
		rebuilt := builder.Document()
		if !found || rebuilt.Content != content || rebuilt.Provenance == nil {
			return unavailableSearchMatch("The retained source no longer proves this indexed passage."), false, nil
		}
		p = rebuilt.Provenance
		// The builder validated the source slices in this captured prefix.
		ranges, valid := mapSearchRanges(content, p, start, end)
		if !valid {
			return unavailableSearchMatch("The passage location is invalid."), false, nil
		}
		return searchMatchResponse{Status: "exact", Ranges: ranges}, false, nil
	}
	ranges, valid := mapSearchRanges(content, p, start, end)
	if !valid {
		return unavailableSearchMatch("The passage location is invalid."), false, nil
	}
	needed := make(map[int64][]index.SearchSpan)
	for _, span := range p.Spans {
		if span.StartByte < end && span.EndByte > start {
			needed[span.Seq] = append(needed[span.Seq], span)
		}
	}
	mismatch := false
	err := s.readSearchSource(ctx, id, func(ev runtime.Event) error {
		spans, ok := needed[ev.Seq]
		if !ok {
			return nil
		}
		projection, err := transcript.ProjectEvent(ev)
		if err != nil {
			mismatch = true
			return errSearchSourceDone
		}
		for _, span := range spans {
			if span.PartIndex < 0 || span.PartIndex >= len(projection.Parts) {
				mismatch = true
				break
			}
			part := projection.Parts[span.PartIndex]
			if !part.Indexed || span.SourceStartByte < 0 || span.SourceEndByte > len(part.Text) || span.SourceEndByte < span.SourceStartByte ||
				part.Text[span.SourceStartByte:span.SourceEndByte] != content[span.StartByte:span.EndByte] {
				mismatch = true
				break
			}
		}
		delete(needed, ev.Seq)
		if mismatch || len(needed) == 0 {
			return errSearchSourceDone
		}
		return nil
	})
	if err != nil && !errors.Is(err, errSearchSourceDone) {
		if searchSourceUnavailable(err) || errors.Is(err, context.DeadlineExceeded) {
			return unavailableSearchMatch("The retained source is unavailable or exceeds the lookup limit."), false, nil
		}
		return searchMatchResponse{}, false, err
	}
	if mismatch || len(needed) != 0 {
		return unavailableSearchMatch("The retained source no longer matches this passage."), false, nil
	}
	return searchMatchResponse{Status: "exact", Ranges: ranges}, false, nil
}

func searchDocumentBoundary(doc string) (string, int64, bool) {
	if doc == "legacy" {
		return doc, 0, true
	}
	kind, suffix, ok := strings.Cut(doc, ":")
	if !ok || (kind != "turn" && kind != "flush" && kind != "close" && kind != "partial") {
		return "", 0, false
	}
	n, err := strconv.ParseInt(suffix, 10, 64)
	return kind, n, err == nil && n > 0
}

// Account for every byte in the selected interval. Artificial join separators
// contribute no source range; all other bytes must belong to a valid span.
func mapSearchRanges(content string, p *index.SearchProvenance, start, end int) ([]searchSourceRange, bool) {
	if len(p.Spans) > index.MaxSearchSpans || len(p.Separators) > index.MaxSearchSpans || !utf8.ValidString(content) {
		return nil, false
	}
	type interval struct {
		start, end int
		span       *index.SearchSpan
	}
	parts := make([]interval, 0, len(p.Spans)+len(p.Separators))
	last := 0
	var priorSeq int64
	priorPart := -1
	for i := range p.Spans {
		span := &p.Spans[i]
		if span.StartByte < last || span.EndByte <= span.StartByte || span.EndByte > len(content) || span.Seq <= 0 || span.PartIndex < 0 || span.SourceStartByte < 0 || span.SourceEndByte-span.SourceStartByte != span.EndByte-span.StartByte ||
			!searchByteBoundary(content, span.StartByte) || !searchByteBoundary(content, span.EndByte) {
			return nil, false
		}
		if span.Seq < priorSeq || (span.Seq == priorSeq && span.PartIndex <= priorPart) {
			return nil, false
		}
		priorSeq, priorPart = span.Seq, span.PartIndex
		last = span.EndByte
		parts = append(parts, interval{span.StartByte, span.EndByte, span})
	}
	last = 0
	for _, sep := range p.Separators {
		if sep.StartByte < last || sep.EndByte <= sep.StartByte || sep.EndByte > len(content) {
			return nil, false
		}
		if strings.Trim(content[sep.StartByte:sep.EndByte], " \n") != "" {
			return nil, false
		}
		last = sep.EndByte
		parts = append(parts, interval{sep.StartByte, sep.EndByte, nil})
	}
	sort.Slice(parts, func(i, j int) bool { return parts[i].start < parts[j].start })
	ranges := make([]searchSourceRange, 0)
	cursor, priorEnd := start, 0
	for _, part := range parts {
		if part.start < priorEnd {
			return nil, false
		}
		priorEnd = part.end
		if part.end <= start || part.start >= end {
			continue
		}
		lo, hi := max(start, part.start), min(end, part.end)
		if lo != cursor {
			return nil, false
		}
		cursor = hi
		if part.span != nil {
			span := part.span
			ranges = append(ranges, searchSourceRange{span.Seq, span.PartIndex, span.SourceStartByte + lo - span.StartByte, span.SourceStartByte + hi - span.StartByte})
		}
	}
	return ranges, cursor == end && len(ranges) > 0
}

func searchByteBoundary(text string, offset int) bool {
	return offset >= 0 && offset <= len(text) && (offset == len(text) || utf8.RuneStart(text[offset]))
}

// Decode arrays incrementally so malformed persisted data cannot allocate an
// arbitrary number of spans before the item limit is enforced.
func decodeSearchProvenance(raw string, into *index.SearchProvenance) error {
	d := json.NewDecoder(strings.NewReader(raw))
	if token, err := d.Token(); err != nil || token != json.Delim('{') {
		return errors.New("invalid provenance")
	}
	seen := map[string]bool{}
	for d.More() {
		token, err := d.Token()
		if err != nil {
			return err
		}
		key, ok := token.(string)
		if !ok || seen[key] || (key != "spans" && key != "separators") {
			return errors.New("invalid provenance field")
		}
		seen[key] = true
		if token, err = d.Token(); err != nil || token != json.Delim('[') {
			return errors.New("invalid provenance array")
		}
		count := 0
		for d.More() {
			if count >= index.MaxSearchSpans {
				return errSearchSourceLimit
			}
			count++
			if key == "spans" {
				var span index.SearchSpan
				if err := d.Decode(&span); err != nil {
					return err
				}
				into.Spans = append(into.Spans, span)
			} else {
				var sep index.SearchSeparator
				if err := d.Decode(&sep); err != nil {
					return err
				}
				into.Separators = append(into.Separators, sep)
			}
		}
		if _, err := d.Token(); err != nil {
			return err
		}
	}
	if _, err := d.Token(); err != nil {
		return err
	}
	if !seen["spans"] || !seen["separators"] {
		return errors.New("incomplete provenance")
	}
	if _, err := d.Token(); !errors.Is(err, io.EOF) {
		return errors.New("trailing provenance")
	}
	return nil
}
