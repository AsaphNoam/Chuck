package index

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"unicode"

	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/transcript"
)

const MaxSearchSpans = 100000
const MaxSearchProvenanceBytes = 8 << 20

type SearchSpan struct {
	StartByte       int   `json:"start_byte"`
	EndByte         int   `json:"end_byte"`
	Seq             int64 `json:"seq"`
	PartIndex       int   `json:"part_index"`
	SourceStartByte int   `json:"source_start_byte"`
	SourceEndByte   int   `json:"source_end_byte"`
}
type SearchSeparator struct {
	StartByte int `json:"start_byte"`
	EndByte   int `json:"end_byte"`
}
type SearchProvenance struct {
	Spans      []SearchSpan      `json:"spans"`
	Separators []SearchSeparator `json:"separators"`
}
type SearchDocument struct {
	Content    string
	Provenance *SearchProvenance
}

func ContentSHA256(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

// SearchDocumentBuilder accepts normalized events in transcript order. Callers
// choose the same turn/annotation/close boundary as the immutable FTS document.
// It retains text, not events; optional location data is dropped at admission
// when its bounds are exceeded. The complete searchable text is always kept.
type SearchDocumentBuilder struct {
	text         strings.Builder
	provenance   SearchProvenance
	unavailable  bool
	encodedBytes int
}

func (b *SearchDocumentBuilder) AddEvent(ev runtime.Event) error {
	p, err := transcript.ProjectEvent(ev)
	if err != nil {
		return err
	}
	// SearchText joins all indexed parts first, then trims the event as a whole.
	// Keep original part positions, including whitespace inside an event.
	var raw strings.Builder
	type partRange struct{ start, end, index int }
	ranges := make([]partRange, 0)
	for i, part := range p.Parts {
		if !part.Indexed {
			continue
		}
		if raw.Len() > 0 {
			raw.WriteByte(' ')
		}
		start := raw.Len()
		raw.WriteString(part.Text)
		if !b.unavailable {
			if len(ranges) >= MaxSearchSpans {
				b.drop()
				ranges = nil
			} else {
				ranges = append(ranges, partRange{start, raw.Len(), i})
			}
		}
	}
	s := raw.String()
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return nil
	}
	left := len(s) - len(strings.TrimLeftFunc(s, unicode.IsSpace))
	right := left + len(trimmed)
	offset := b.text.Len()
	if offset > 0 {
		b.text.WriteByte('\n')
		b.separator(offset, offset+1)
		offset++
	}
	b.text.WriteString(trimmed)
	if b.unavailable {
		return nil
	}
	cursor := left
	for _, r := range ranges {
		start, end := max(left, r.start), min(right, r.end)
		if end <= start {
			continue
		}
		if start > cursor {
			b.separator(offset+cursor-left, offset+start-left)
		}
		span := SearchSpan{StartByte: offset + start - left, EndByte: offset + end - left, Seq: ev.Seq, PartIndex: r.index, SourceStartByte: start - r.start, SourceEndByte: end - r.start}
		encoded, _ := json.Marshal(span)
		if len(b.provenance.Spans) >= MaxSearchSpans || b.encodedBytes+len(encoded)+1 > MaxSearchProvenanceBytes-64 {
			b.drop()
			break
		}
		b.encodedBytes += len(encoded) + 1
		b.provenance.Spans = append(b.provenance.Spans, span)
		cursor = end
	}
	if cursor < right {
		b.separator(offset+cursor-left, offset+right-left)
	}
	return nil
}

func (b *SearchDocumentBuilder) drop() { b.unavailable = true; b.provenance = SearchProvenance{} }

func (b *SearchDocumentBuilder) appendDocument(d SearchDocument) {
	if d.Content == "" {
		return
	}
	offset := b.text.Len()
	if offset > 0 {
		b.text.WriteByte('\n')
		b.separator(offset, offset+1)
		offset++
	}
	b.text.WriteString(d.Content)
	if b.unavailable {
		return
	}
	if d.Provenance == nil {
		b.drop()
		return
	}
	for _, s := range d.Provenance.Spans {
		s.StartByte += offset
		s.EndByte += offset
		raw, _ := json.Marshal(s)
		if len(b.provenance.Spans) >= MaxSearchSpans || b.encodedBytes+len(raw)+1 > MaxSearchProvenanceBytes-64 {
			b.drop()
			return
		}
		b.encodedBytes += len(raw) + 1
		b.provenance.Spans = append(b.provenance.Spans, s)
	}
	for _, s := range d.Provenance.Separators {
		b.separator(s.StartByte+offset, s.EndByte+offset)
	}
}
func (b *SearchDocumentBuilder) separator(start, end int) {
	if b.unavailable || end <= start {
		return
	}
	s := SearchSeparator{start, end}
	raw, _ := json.Marshal(s)
	if len(b.provenance.Separators) >= MaxSearchSpans || b.encodedBytes+len(raw)+1 > MaxSearchProvenanceBytes-64 {
		b.drop()
		return
	}
	b.encodedBytes += len(raw) + 1
	b.provenance.Separators = append(b.provenance.Separators, s)
}

func (b *SearchDocumentBuilder) Document() SearchDocument {
	d := SearchDocument{Content: b.text.String()}
	if !b.unavailable {
		p := b.provenance
		if p.Spans == nil {
			p.Spans = []SearchSpan{}
		}
		if p.Separators == nil {
			p.Separators = []SearchSeparator{}
		}
		d.Provenance = &p
	}
	return d
}

func BuildSearchDocument(events []runtime.Event) (SearchDocument, error) {
	var b SearchDocumentBuilder
	for _, ev := range events {
		if err := b.AddEvent(ev); err != nil {
			return SearchDocument{}, err
		}
	}
	return b.Document(), nil
}
