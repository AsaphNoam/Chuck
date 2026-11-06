//go:build sqlite_fts5

package archive

import (
	"context"
	"errors"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestProjectSearchPrefixAndReference(t *testing.T) {
	db, cleanup := openArchiveTestDB(t)
	defer cleanup()
	resp, err := New(db).ProjectSearch(context.Background(), Query{Project: "my-app", Q: "distinct", ProjectSearch: true})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Total != 1 || len(resp.Results) != 1 {
		t.Fatalf("results=%+v total=%d", resp.Results, resp.Total)
	}
	r := resp.Results[0]
	if r.AgentID != "a_active" || r.LocationStatus != "reference" || r.SearchRef == nil {
		t.Fatalf("result=%+v", r)
	}
	if r.SearchRef.DocumentID != "turn:1" || r.SearchRef.StartByte >= r.SearchRef.EndByte || r.Snippet == "" {
		t.Fatalf("ref=%+v snippet=%q", r.SearchRef, r.Snippet)
	}
}

func TestProjectSearchLiteralQueryParsing(t *testing.T) {
	if err := ValidateProjectSearchQuery(`auth "exact phrase" OR (x)`); err != nil {
		t.Fatal(err)
	}
	terms, err := parseProjectTerms(`auth "exact phrase" OR (x)`)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := projectFTSQuery(terms), `"auth"* "exact phrase" "OR"* "(x)"*`; got != want {
		t.Fatalf("FTS query=%q want %q", got, want)
	}
	if err := ValidateProjectSearchQuery(string(make([]byte, 513))); !errors.Is(err, ErrInvalidProjectSearchQuery) {
		t.Fatalf("oversize error=%v", err)
	}
}

func TestProjectSearchLifecycleBeforePagingAndScope(t *testing.T) {
	db, cleanup := openArchiveTestDB(t)
	defer cleanup()
	for _, row := range []struct{ id, project, updated, text string }{
		{"a_running", "project-a", "2026-06-28T10:00:00Z", "a distinctive match"},
		{"a_stopped", "project-a", "2026-06-28T10:09:00Z", "a distinctive match"},
		{"a_archived", "project-a", "2026-06-28T10:11:00Z", "a distinctive match"},
		{"b_running", "project-b", "2026-06-28T10:12:00Z", "a distinctive match"},
		{"t_running", "project-a", "2026-06-28T10:13:00Z", "a distinctive match"},
	} {
		if err := insertSession(db, row.id, row.id, "worker", row.project, "", row.updated, row.text); err != nil {
			t.Fatal(err)
		}
		archived := 0
		if row.id == "a_archived" {
			archived = 1
		}
		if _, err := db.Exec(`INSERT INTO agents(agent_id,name,role,project,backend,model,interface,created_at,archived) VALUES (?,?,? ,?,'claude','sonnet','chat','now',?)`, row.id, row.id, "worker", row.project, archived); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`INSERT INTO running(agent_id,pid,session_id,interface,started_at) VALUES ('a_running',1,'s1','chat','now'),('b_running',2,'s2','chat','now'),('t_running',3,'s3','chat','now')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE sessions SET interface='terminal' WHERE agent_id='t_running'`); err != nil {
		t.Fatal(err)
	}
	a := New(db)
	for offset, want := range []string{"a_running", "a_stopped", "a_archived"} {
		resp, err := a.ProjectSearch(context.Background(), Query{Project: "project-a", Q: "distinct", Limit: 1, Offset: offset, ProjectSearch: true})
		if err != nil {
			t.Fatal(err)
		}
		if resp.Total != 3 || len(resp.Results) != 1 || resp.Results[0].AgentID != want {
			t.Fatalf("offset %d results=%+v total=%d want %s", offset, resp.Results, resp.Total, want)
		}
	}
	// The legacy Archive entry point retains exact-token matching.
	ordinary, err := a.Search(Query{Project: "project-a", Q: "distinct", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if ordinary.Total != 0 {
		t.Fatalf("ordinary archive prefix total=%d, want 0", ordinary.Total)
	}
}

func TestProjectSearchFirstUnicodeHighlightUsesUTF8Offsets(t *testing.T) {
	db, cleanup := openArchiveTestDB(t)
	defer cleanup()
	content := "éclair distinctive passage and distinctive again"
	if err := insertSession(db, "unicode_hit", "unicode_hit", "worker", "project-a", "", "2026-06-28T10:00:00Z", content); err != nil {
		t.Fatal(err)
	}
	resp, err := New(db).ProjectSearch(context.Background(), Query{Project: "project-a", Q: "distinct", ProjectSearch: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Results) != 1 || resp.Results[0].SearchRef == nil {
		t.Fatalf("results=%+v", resp.Results)
	}
	ref := resp.Results[0].SearchRef
	want := len([]byte("éclair "))
	wantEnd := want + len([]byte("distinctive"))
	if ref.StartByte != want || ref.EndByte != wantEnd {
		t.Fatalf("ref=%+v want exact byte range [%d,%d)", ref, want, wantEnd)
	}
}

func TestProjectSearchBlankAndBounds(t *testing.T) {
	db, cleanup := openArchiveTestDB(t)
	defer cleanup()
	a := New(db)
	resp, err := a.ProjectSearch(context.Background(), Query{Project: "my-app", Q: "  ", ProjectSearch: true})
	if err != nil || resp.Total != 0 || resp.Results == nil {
		t.Fatalf("resp=%+v err=%v", resp, err)
	}
	if _, err := a.ProjectSearch(context.Background(), Query{Project: "my-app", Q: string(make([]byte, 513)), ProjectSearch: true}); err == nil {
		t.Fatal("oversize query accepted")
	}
	if _, err := a.ProjectSearch(context.Background(), Query{Project: "my-app", Q: `"unfinished`, ProjectSearch: true}); err == nil {
		t.Fatal("unmatched quote accepted")
	}
}

// SQLite snippets count tokens, not characters: one oversized token must not
// escape the document admission bound through its fallback preview.
func TestProjectSearchOversizedDocumentKeepsBoundedPreview(t *testing.T) {
	db, cleanup := openArchiveTestDB(t)
	defer cleanup()
	if err := insertSession(db, "a_oversized", "Oversized", "worker", "large-project", "", "2026-11-06T10:00:00Z", strings.Repeat("z", projectSearchDocumentLimit+1)); err != nil {
		t.Fatal(err)
	}
	response, err := New(db).ProjectSearch(context.Background(), Query{Project: "large-project", Q: "zz", ProjectSearch: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Results) != 1 {
		t.Fatalf("results=%+v", response.Results)
	}
	row := response.Results[0]
	if row.LocationStatus != "unavailable" || row.SearchRef != nil || utf8.RuneCountInString(row.Snippet) > 256 {
		t.Fatalf("location=%s reference=%+v preview characters=%d", row.LocationStatus, row.SearchRef, utf8.RuneCountInString(row.Snippet))
	}
}
