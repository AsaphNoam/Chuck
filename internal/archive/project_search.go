package archive

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

const projectSearchDocumentLimit = 4 << 20

var ErrInvalidProjectSearchQuery = errors.New("archive: invalid project search query")

type projectTerm struct {
	text   string
	quoted bool
}

// ProjectSearch executes the opt-in scoped project search. Its context covers
// the FTS query and location enrichment; callers own concurrency admission.
func (a *Archive) ProjectSearch(parent context.Context, q Query) (Response, error) {
	ctx, cancel := context.WithTimeout(parent, 2*time.Second)
	defer cancel()
	if q.Active != nil {
		return Response{}, fmt.Errorf("%w: active filter is not supported", ErrInvalidProjectSearchQuery)
	}
	if err := ValidateProjectSearchQuery(q.Q); err != nil {
		return Response{}, err
	}
	term := strings.TrimSpace(q.Q)
	if q.Limit <= 0 {
		q.Limit = 20
	}
	if q.Limit > 50 {
		q.Limit = 50
	}
	if q.Offset < 0 {
		q.Offset = 0
	}
	if term == "" {
		return Response{Query: term, SearchMode: a.searchMode(), Total: 0, Limit: q.Limit, Offset: q.Offset, Results: []Result{}}, nil
	}
	terms, err := parseProjectTerms(term)
	if err != nil {
		return Response{}, err
	}
	match := projectFTSQuery(terms)
	where, args := archiveWhere(q, "AND")
	countArgs := append([]any{match}, args...)
	var total int
	if err := a.db.QueryRowContext(ctx, `SELECT COUNT(DISTINCT sessions_fts.agent_id) FROM sessions_fts JOIN sessions s ON s.agent_id=sessions_fts.agent_id LEFT JOIN agents a ON a.agent_id=s.agent_id LEFT JOIN running r ON r.agent_id=s.agent_id WHERE sessions_fts MATCH ? AND s.interface='chat'`+where, countArgs...).Scan(&total); err != nil {
		if isFTS5Missing(err) {
			return a.projectSearchFallback(ctx, q, terms, term)
		}
		return Response{}, fmt.Errorf("archive: count project search: %w", err)
	}
	queryArgs := append([]any{match}, args...)
	queryArgs = append(queryArgs, q.Limit, q.Offset)
	rows, err := a.db.QueryContext(ctx, `WITH matching_documents AS (
 SELECT agent_id,document_id,bm25(sessions_fts,0,0,8,4,4,2,1,1,1) score,
        substr(snippet(sessions_fts,8,'','','…',12),1,256) snippet
 FROM sessions_fts WHERE sessions_fts MATCH ?
), ranked_documents AS (
 SELECT agent_id,document_id,score,snippet,
        ROW_NUMBER() OVER(PARTITION BY agent_id ORDER BY score,document_id) doc_rank
 FROM matching_documents
), summary AS (
 SELECT agent_id,MAX(document_id='metadata') meta,MAX(document_id<>'metadata') transcript
 FROM matching_documents GROUP BY agent_id
), transcript_choice AS (
 SELECT agent_id,document_id,snippet,ROW_NUMBER() OVER(PARTITION BY agent_id ORDER BY score,document_id) transcript_rank
 FROM matching_documents WHERE document_id<>'metadata'
)
SELECT s.agent_id,s.name,s.role,s.project,s.backend,s.model,s.effort,s.fast,s.interface,s.grp,
 s.created_at,s.updated_at,s.turn_count,s.files_touched,s.commands_run,(r.agent_id IS NOT NULL),COALESCE(a.archived,0),
 COALESCE(tc.snippet,''),summary.meta,summary.transcript,COALESCE(tc.document_id,'')
FROM summary JOIN sessions s ON s.agent_id=summary.agent_id LEFT JOIN agents a ON a.agent_id=s.agent_id LEFT JOIN running r ON r.agent_id=s.agent_id
LEFT JOIN transcript_choice tc ON tc.agent_id=summary.agent_id AND tc.transcript_rank=1
JOIN ranked_documents rd ON rd.agent_id=summary.agent_id AND rd.doc_rank=1
WHERE s.interface='chat'`+where+`
ORDER BY CASE WHEN COALESCE(a.archived,0)=1 THEN 2 WHEN r.agent_id IS NOT NULL THEN 0 ELSE 1 END,rd.score,s.updated_at DESC,s.agent_id LIMIT ? OFFSET ?`, queryArgs...)
	if err != nil {
		if isFTS5Missing(err) {
			return a.projectSearchFallback(ctx, q, terms, term)
		}
		return Response{}, fmt.Errorf("archive: project search: %w", err)
	}
	type pending struct {
		result Result
		doc    string
	}
	items := make([]pending, 0, q.Limit)
	for rows.Next() {
		var p pending
		var active, archived, meta, transcript int
		if err := rows.Scan(&p.result.AgentID, &p.result.Name, &p.result.Role, &p.result.Project, &p.result.Backend, &p.result.Model, &p.result.Effort, &p.result.Fast, &p.result.Interface, &p.result.Group, &p.result.CreatedAt, &p.result.UpdatedAt, &p.result.TurnCount, &p.result.FilesTouched, &p.result.CommandsRun, &active, &archived, &p.result.Snippet, &meta, &transcript, &p.doc); err != nil {
			rows.Close()
			return Response{}, fmt.Errorf("archive: scan project result: %w", err)
		}
		p.result.Active = active != 0
		p.result.Archived = archived != 0
		if meta != 0 {
			p.result.MatchedIn = append(p.result.MatchedIn, "metadata")
		}
		if transcript != 0 {
			p.result.MatchedIn = append(p.result.MatchedIn, "transcript")
		}
		if p.doc != "" {
			p.result.LocationStatus = "unavailable"
		} else {
			p.result.LocationStatus = "metadata_only"
		}
		items = append(items, p)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return Response{}, fmt.Errorf("archive: iterate project results: %w", err)
	}
	if err := rows.Close(); err != nil {
		return Response{}, fmt.Errorf("archive: close project results: %w", err)
	}
	// Enrich only after closing the result cursor because SQLite may be configured
	// with a single connection.
	for i := range items {
		if items[i].doc == "" {
			continue
		}
		if err := ctx.Err(); err != nil {
			return Response{}, fmt.Errorf("archive: project search budget: %w", err)
		}
		ref, excerpt, err := a.searchReference(ctx, match, items[i].result.AgentID, items[i].doc)
		if err != nil {
			return Response{}, err
		}
		if ref != nil {
			items[i].result.SearchRef = ref
			items[i].result.LocationStatus = "reference"
			items[i].result.Snippet = excerpt
		}
	}
	results := make([]Result, len(items))
	for i := range items {
		results[i] = items[i].result
	}
	return Response{Query: term, SearchMode: a.searchMode(), Total: total, Limit: q.Limit, Offset: q.Offset, Results: results}, nil
}

// ValidateProjectSearchQuery validates the public project-search input before
// a handler starts a database operation, allowing it to distinguish 422 input
// errors from storage failures.
func ValidateProjectSearchQuery(raw string) error {
	if len([]byte(raw)) > 512 {
		return fmt.Errorf("%w: query exceeds 512 bytes", ErrInvalidProjectSearchQuery)
	}
	if _, err := parseProjectTerms(strings.TrimSpace(raw)); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidProjectSearchQuery, err)
	}
	return nil
}

func parseProjectTerms(q string) ([]projectTerm, error) {
	return parseQueryTerms(q, true)
}

func projectFTSQuery(terms []projectTerm) string {
	out := make([]string, 0, len(terms))
	for _, t := range terms {
		lit := strings.ReplaceAll(t.text, `"`, `""`)
		atom := `"` + lit + `"`
		if !t.quoted && utf8.RuneCountInString(t.text) >= 2 {
			atom += `*`
		}
		out = append(out, atom)
	}
	return strings.Join(out, " ")
}

func (a *Archive) searchReference(ctx context.Context, match, agent, doc string) (*SearchRef, string, error) {
	var content string
	err := a.db.QueryRowContext(ctx, `SELECT content FROM sessions_fts WHERE agent_id=? AND document_id=? AND sessions_fts MATCH ? AND length(CAST(content AS BLOB))<=?`, agent, doc, match, projectSearchDocumentLimit).Scan(&content)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, "", nil
		}
		return nil, "", fmt.Errorf("archive: read matched document: %w", err)
	}
	// Markers are content-derived and collision checked with a fixed retry cap.
	contentHash := sha256.Sum256([]byte(content))
	seed := hex.EncodeToString(contentHash[:])
	open, close := "", ""
	for i := 0; i < 8; i++ {
		suffix := seed + ":" + string(rune('a'+i))
		candidateOpen, candidateClose := "\x01chuck:"+suffix+":o\x02", "\x01chuck:"+suffix+":c\x02"
		if !strings.Contains(content, candidateOpen) && !strings.Contains(content, candidateClose) {
			open, close = candidateOpen, candidateClose
			break
		}
	}
	if open == "" {
		return nil, "", nil
	}
	var marked string
	if err := a.db.QueryRowContext(ctx, `SELECT highlight(sessions_fts,8,?,?) FROM sessions_fts WHERE agent_id=? AND document_id=? AND sessions_fts MATCH ?`, open, close, agent, doc, match).Scan(&marked); err != nil {
		return nil, "", fmt.Errorf("archive: highlight matched document: %w", err)
	}
	start, end := -1, -1
	plain := strings.Builder{}
	for len(marked) > 0 {
		if strings.HasPrefix(marked, open) {
			if start < 0 {
				start = plain.Len()
			}
			marked = strings.TrimPrefix(marked, open)
			continue
		}
		if strings.HasPrefix(marked, close) {
			if start >= 0 {
				end = plain.Len()
				break
			}
			marked = strings.TrimPrefix(marked, close)
			continue
		}
		_, n := utf8.DecodeRuneInString(marked)
		plain.WriteString(marked[:n])
		marked = marked[n:]
	}
	text := plain.String()
	if start < 0 || end <= start || end > len(text) || !utf8.ValidString(text) {
		return nil, "", nil
	}
	h := sha256.Sum256([]byte(content))
	excerpt := runeExcerpt(content, start, end, 256)
	return &SearchRef{DocumentID: doc, ContentSHA256: hex.EncodeToString(h[:]), StartByte: start, EndByte: end}, excerpt, nil
}

func runeExcerpt(s string, start, end, maxRunes int) string {
	bounds := []int{0}
	for i := range s {
		if i > 0 {
			bounds = append(bounds, i)
		}
	}
	bounds = append(bounds, len(s))
	a, b := 0, 0
	for a+1 < len(bounds) && bounds[a+1] <= start {
		a++
	}
	b = a
	for b+1 < len(bounds) && bounds[b] < end {
		b++
	}
	matchRunes := b - a
	left := a
	if matchRunes >= maxRunes { // Keep the beginning of an overlong match.
		return s[bounds[a]:bounds[min(a+maxRunes, len(bounds)-1)]]
	}
	left = max(0, a-(maxRunes-matchRunes)/2)
	right := min(left+maxRunes, len(bounds)-1)
	if right < b {
		left = max(0, b-maxRunes)
		right = b
	}
	return s[bounds[left]:bounds[right]]
}

func (a *Archive) projectSearchFallback(ctx context.Context, q Query, terms []projectTerm, raw string) (Response, error) {
	where, args := archiveWhere(q, "AND")
	clauses := make([]string, 0, len(terms))
	for range terms {
		clauses = append(clauses, `(instr(lower(s.name),lower(?))>0 OR instr(lower(s.role),lower(?))>0 OR instr(lower(s.project),lower(?))>0 OR instr(lower(s.backend),lower(?))>0 OR instr(lower(s.grp),lower(?))>0 OR instr(lower(s.model),lower(?))>0)`)
	}
	filter := strings.Join(clauses, " AND ")
	base := ` FROM sessions s LEFT JOIN agents a ON a.agent_id=s.agent_id LEFT JOIN running r ON r.agent_id=s.agent_id WHERE s.interface='chat'`
	if where != "" {
		base += where
	}
	base += ` AND ` + filter
	likes := []any{}
	for _, t := range terms {
		v := t.text
		for range 6 {
			likes = append(likes, v)
		}
	}
	countArgs := append(append([]any{}, args...), likes...)
	var total int
	if err := a.db.QueryRowContext(ctx, "SELECT COUNT(*)"+base, countArgs...).Scan(&total); err != nil {
		return Response{}, fmt.Errorf("archive: fallback project count: %w", err)
	}
	queryArgs := append(append([]any{}, args...), likes...)
	queryArgs = append(queryArgs, q.Limit, q.Offset)
	rows, err := a.db.QueryContext(ctx, `SELECT s.agent_id,s.name,s.role,s.project,s.backend,s.model,s.effort,s.fast,s.interface,s.grp,s.created_at,s.updated_at,s.turn_count,s.files_touched,s.commands_run,(r.agent_id IS NOT NULL),COALESCE(a.archived,0),'' FROM sessions s LEFT JOIN agents a ON a.agent_id=s.agent_id LEFT JOIN running r ON r.agent_id=s.agent_id WHERE s.interface='chat'`+where+` AND `+filter+` ORDER BY CASE WHEN COALESCE(a.archived,0)=1 THEN 2 WHEN r.agent_id IS NOT NULL THEN 0 ELSE 1 END,s.updated_at DESC,s.agent_id LIMIT ? OFFSET ?`, queryArgs...)
	if err != nil {
		return Response{}, fmt.Errorf("archive: fallback project search: %w", err)
	}
	defer rows.Close()
	results := make([]Result, 0)
	for rows.Next() {
		var r Result
		var active, arch int
		if err := rows.Scan(&r.AgentID, &r.Name, &r.Role, &r.Project, &r.Backend, &r.Model, &r.Effort, &r.Fast, &r.Interface, &r.Group, &r.CreatedAt, &r.UpdatedAt, &r.TurnCount, &r.FilesTouched, &r.CommandsRun, &active, &arch, &r.Snippet); err != nil {
			return Response{}, err
		}
		r.Active = active != 0
		r.Archived = arch != 0
		r.MatchedIn = []string{"metadata"}
		r.LocationStatus = "metadata_only"
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return Response{}, err
	}
	return Response{Query: strings.TrimSpace(raw), SearchMode: "metadata", Total: total, Limit: q.Limit, Offset: q.Offset, Results: results}, nil
}
