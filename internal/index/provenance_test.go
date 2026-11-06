package index

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
)

// TS-02.R64: a populated v69 database keeps exact legacy documents on
// upgrade, without fabricated source locations, under either SQLite build.
func TestSearchProvenanceMigrationPreservesDocuments(t *testing.T) {
	st, home := openTestDB(t)
	if _, err := st.DB().Exec(`INSERT INTO sessions_fts(agent_id,document_id,name,role,project,grp,model,backend,content) VALUES('old','legacy','','','','','','','café retained'); DROP TABLE session_search_spans; DELETE FROM schema_migrations WHERE version=70`); err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := state.Open(home)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	var text string
	if err := reopened.DB().QueryRow(`SELECT content FROM sessions_fts WHERE agent_id='old' AND document_id='legacy'`).Scan(&text); err != nil || text != "café retained" {
		t.Fatalf("upgrade document=%q %v", text, err)
	}
	var n int
	if err := reopened.DB().QueryRow(`SELECT COUNT(*) FROM session_search_spans`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("upgrade invented provenance: %d %v", n, err)
	}
}

// TS-02.R63: independently known UTF-8 offsets distinguish a constituent
// delta from a turn's final sequence. A real trigger proves publication and
// buffer release are atomic with provenance storage.
func TestSearchProvenancePublication(t *testing.T) {
	st, _ := openTestDB(t)
	ix := New(st.DB())
	if err := ix.UpsertSessionMeta("a_index", meta()); err != nil {
		t.Fatal(err)
	}
	for _, e := range []runtime.Event{
		ev(t, 1, runtime.EvAssistantText, runtime.AssistantTextData{Delta: "  café  "}),
		ev(t, 2, runtime.EvAssistantText, runtime.AssistantTextData{Delta: "猫"}),
	} {
		if err := ix.OnEvent("a_index", e); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := st.DB().Exec(`CREATE TRIGGER fail_spans BEFORE INSERT ON session_search_spans BEGIN SELECT RAISE(ABORT,'span failure'); END`); err != nil {
		t.Fatal(err)
	}
	if err := ix.FlushPendingContent("a_index", 2); err == nil {
		t.Fatal("flush accepted failed provenance write")
	}
	var n int
	if err := st.DB().QueryRow(`SELECT COUNT(*) FROM sessions_fts WHERE document_id='close:2'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("failed transaction published text: %d %v", n, err)
	}
	if _, err := st.DB().Exec(`DROP TRIGGER fail_spans`); err != nil {
		t.Fatal(err)
	}
	if err := ix.FlushPendingContent("a_index", 2); err != nil {
		t.Fatal(err)
	}
	var content, digest, raw string
	if err := st.DB().QueryRow(`SELECT f.content,p.content_sha256,p.spans_json FROM sessions_fts f JOIN session_search_spans p USING(agent_id,document_id) WHERE f.document_id='close:2'`).Scan(&content, &digest, &raw); err != nil {
		t.Fatal(err)
	}
	if content != "café\n猫" || digest != ContentSHA256("café\n猫") {
		t.Fatalf("content/digest=%q %q", content, digest)
	}
	var p SearchProvenance
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		t.Fatal(err)
	}
	want := SearchProvenance{Spans: []SearchSpan{{0, 5, 1, 0, 2, 7}, {6, 9, 2, 0, 0, 3}}, Separators: []SearchSeparator{{5, 6}}}
	if !reflect.DeepEqual(p, want) {
		t.Fatalf("provenance=%+v want %+v", p, want)
	}
	if len(ix.content["a_index"]) != 0 {
		t.Fatal("successful publication retained buffer")
	}
	if err := ix.RemoveAgent("a_index"); err != nil {
		t.Fatal(err)
	}
	if err := st.DB().QueryRow(`SELECT COUNT(*) FROM session_search_spans`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("orphan spans: %d %v", n, err)
	}
}

// Optional source mapping must never truncate text when the span budget runs
// out. This exercises admission incrementally, not an after-the-fact decoder.
func TestSearchProvenanceBudgetKeepsText(t *testing.T) {
	var b SearchDocumentBuilder
	e := ev(t, 1, runtime.EvAssistantText, runtime.AssistantTextData{Delta: "x"})
	for i := 0; i < MaxSearchSpans+1; i++ {
		e.Seq = int64(i + 1)
		if err := b.AddEvent(e); err != nil {
			t.Fatal(err)
		}
	}
	d := b.Document()
	if d.Provenance != nil || d.Content != strings.Repeat("x\n", MaxSearchSpans)+"x" {
		t.Fatalf("budget changed text or retained provenance: bytes=%d", len(d.Content))
	}
}
