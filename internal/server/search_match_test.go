package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/AsaphNoam/Chuck/internal/index"
	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
	"github.com/AsaphNoam/Chuck/internal/transcript"
)

// These real HTTP reads must resolve independently known UTF-8 source offsets,
// including repeated text and an artificial event separator, without writes.
func TestSearchMatchHTTPSourceAndLegacy(t *testing.T) {
	s := testServer(t, true)
	ix := index.New(s.stateStore.DB())
	id := "a_search_source"
	meta := runtime.SessionMetaData{Name: "Search source", Project: "my-app", Backend: "claude", Interface: "chat", CreatedAt: "2026-11-06T10:00:00Z"}
	if err := ix.UpsertSessionMeta(id, meta); err != nil {
		t.Fatal(err)
	}
	writer, err := transcript.Open(s.configStore.Home(), id, &meta)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	for i, text := range []string{"  café café  ", "猫 authentication"} {
		raw, _ := json.Marshal(runtime.AssistantTextData{Delta: text})
		ev := runtime.Event{Seq: int64(i + 1), Type: runtime.EvAssistantText, Data: raw}
		if err := writer.Append(ev); err != nil {
			t.Fatal(err)
		}
		if err := ix.OnEvent(id, ev); err != nil {
			t.Fatal(err)
		}
	}
	if err := ix.FlushPendingContent(id, 2); err != nil {
		t.Fatal(err)
	}
	mode, err := state.DetectSessionsFTS(s.stateStore.DB())
	if err != nil {
		t.Fatal(err)
	}
	if mode != state.SessionsFTSFullText {
		// Untagged builds retain authoritative data but never make a text claim.
		rec := doGET(t, s.routes(), "/api/archive/projects/my-app?view=project_search&q=Search")
		if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"location_status":"metadata_only"`) {
			t.Fatalf("fallback=%d %s", rec.Code, rec.Body.String())
		}
		return
	}
	digest := index.ContentSHA256("café café\n猫 authentication")
	url := fmt.Sprintf("/api/sessions/%s/search-match?document_id=close:2&sha256=%s&start=6&end=15", id, digest)
	check := func(want []searchSourceRange) {
		t.Helper()
		rec := doGET(t, s.routes(), url)
		if rec.Code != 200 {
			t.Fatalf("status=%d %s", rec.Code, rec.Body.String())
		}
		var got searchMatchResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if got.Status != "exact" || !reflect.DeepEqual(got.Ranges, want) {
			t.Fatalf("response=%+v want=%+v", got, want)
		}
	}
	// Indexed bytes 6..11 are the SECOND café, then a newline, then 猫.
	want := []searchSourceRange{{Seq: 1, PartIndex: 0, StartByte: 8, EndByte: 13}, {Seq: 2, PartIndex: 0, StartByte: 0, EndByte: 3}}
	check(want)
	if _, err := s.stateStore.DB().Exec(`DELETE FROM session_search_spans WHERE agent_id=?`, id); err != nil {
		t.Fatal(err)
	}
	check(want) // Older rows reconstruct from source without writing provenance.
	var count int
	if err := s.stateStore.DB().QueryRow(`SELECT COUNT(*) FROM session_search_spans WHERE agent_id=?`, id).Scan(&count); err != nil || count != 0 {
		t.Fatalf("GET wrote provenance count=%d err=%v", count, err)
	}
	url = strings.Replace(url, "start=6", "start=7", 1) // still a valid boundary in café
	url = strings.Replace(url, "end=15", "end=14", 1)   // inside 猫
	if rec := doGET(t, s.routes(), url); rec.Code != 422 {
		t.Fatalf("split rune=%d %s", rec.Code, rec.Body.String())
	}
	url = fmt.Sprintf("/api/sessions/%s/search-match?document_id=close:2&sha256=%s&start=0&end=5", id, strings.Repeat("0", 64))
	if rec := doGET(t, s.routes(), url); rec.Code != 200 || !strings.Contains(rec.Body.String(), `"status":"unavailable"`) || !strings.Contains(rec.Body.String(), `"ranges":[]`) {
		t.Fatalf("stale=%d %s", rec.Code, rec.Body.String())
	}
}

func TestProjectSearchHTTPValidationAndAdmission(t *testing.T) {
	s := testServer(t, true)
	for _, suffix := range []string{"view=unknown", "view=project_search&active=", "view=project_search&limit=51", "view=project_search&offset=-1", "view=project_search&q=" + strings.Repeat("x", 513)} {
		if rec := doGET(t, s.routes(), "/api/archive/projects/my-app?"+suffix); rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("%s: %d %s", suffix, rec.Code, rec.Body.String())
		}
	}
	rec := doGET(t, s.routes(), "/api/archive/projects/my-app?view=project_search&q=")
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"results":[]`) || !strings.Contains(rec.Body.String(), `"limit":20`) {
		t.Fatalf("empty=%d %s", rec.Code, rec.Body.String())
	}
	for i := 0; i < cap(projectSearchReads); i++ {
		projectSearchReads <- struct{}{}
	}
	defer func() {
		for i := 0; i < cap(projectSearchReads); i++ {
			<-projectSearchReads
		}
	}()
	if rec := doGET(t, s.routes(), "/api/archive/projects/my-app?view=project_search&q=auth"); rec.Code < 500 {
		t.Fatalf("admission=%d %s", rec.Code, rec.Body.String())
	}
}

func TestSearchMatchInvalidProvenanceAndReadBudget(t *testing.T) {
	s := testServer(t, true)
	mode, err := state.DetectSessionsFTS(s.stateStore.DB())
	if err != nil {
		t.Fatal(err)
	}
	if mode != state.SessionsFTSFullText {
		t.Skip("FTS-only source documents")
	}
	id := "a_search_limits"
	ix := index.New(s.stateStore.DB())
	if err := ix.UpsertSessionMeta(id, runtime.SessionMetaData{Name: "Limits", Interface: "chat"}); err != nil {
		t.Fatal(err)
	}
	content := "authentication"
	if _, err := s.stateStore.DB().Exec(`INSERT INTO sessions_fts(agent_id,document_id,content) VALUES(?, 'turn:2', ?)`, id, content); err != nil {
		t.Fatal(err)
	}
	digest := index.ContentSHA256(content)
	url := fmt.Sprintf("/api/sessions/%s/search-match?document_id=turn:2&sha256=%s&start=0&end=4", id, digest)
	// A matching digest cannot make forged separator bytes into source text.
	bad := `{"spans":[{"start_byte":0,"end_byte":1,"seq":1,"part_index":0,"source_start_byte":0,"source_end_byte":1}],"separators":[{"start_byte":1,"end_byte":4}]}`
	if _, err := s.stateStore.DB().Exec(`INSERT INTO session_search_spans VALUES(?, 'turn:2',1,?,?)`, id, digest, bad); err != nil {
		t.Fatal(err)
	}
	if rec := doGET(t, s.routes(), url); rec.Code != 200 || !strings.Contains(rec.Body.String(), `"status":"unavailable"`) {
		t.Fatalf("forged=%d %s", rec.Code, rec.Body.String())
	}
	if _, err := s.stateStore.DB().Exec(`DELETE FROM session_search_spans WHERE agent_id=?`, id); err != nil {
		t.Fatal(err)
	}
	if rec := doGET(t, s.routes(), url); rec.Code != 200 || !strings.Contains(rec.Body.String(), `"ranges":[]`) {
		t.Fatalf("missing source=%d %s", rec.Code, rec.Body.String())
	}
	oversize := strings.Repeat("x", maxSearchDocumentBytes+1)
	if _, err := s.stateStore.DB().Exec(`UPDATE sessions_fts SET content=? WHERE agent_id=?`, oversize, id); err != nil {
		t.Fatal(err)
	}
	url = strings.Replace(url, digest, index.ContentSHA256(oversize), 1)
	if rec := doGET(t, s.routes(), url); rec.Code != 200 || !strings.Contains(rec.Body.String(), "size limit") {
		t.Fatalf("document budget=%d %s", rec.Code, rec.Body.String())
	}
	if rec := doGET(t, s.routes(), strings.Replace(url, "turn:2", "made-up", 1)); rec.Code != 422 {
		t.Fatalf("document identity=%d", rec.Code)
	}
	// A real database failure must remain a failed request, not unavailable.
	if _, err := s.stateStore.DB().Exec(`DROP TABLE session_search_spans`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.stateStore.DB().Exec(`UPDATE sessions_fts SET content=? WHERE agent_id=?`, content, id); err != nil {
		t.Fatal(err)
	}
	url = strings.Replace(url, index.ContentSHA256(oversize), digest, 1)
	if rec := doGET(t, s.routes(), url); rec.Code != 500 {
		t.Fatalf("storage failure=%d %s", rec.Code, rec.Body.String())
	}
}

func TestSearchMatchCanonicalSourceBudget(t *testing.T) {
	s := testServer(t, true)
	mode, err := state.DetectSessionsFTS(s.stateStore.DB())
	if err != nil {
		t.Fatal(err)
	}
	if mode != state.SessionsFTSFullText {
		t.Skip("FTS source locator")
	}
	id := "a_source_budget"
	if err = index.New(s.stateStore.DB()).UpsertSessionMeta(id, runtime.SessionMetaData{Name: "Source budget", Interface: "chat"}); err != nil {
		t.Fatal(err)
	}
	content := "needle"
	digest := index.ContentSHA256(content)
	p := index.SearchProvenance{Spans: []index.SearchSpan{{StartByte: 0, EndByte: 6, Seq: 35, PartIndex: 0, SourceStartByte: 0, SourceEndByte: 6}}, Separators: []index.SearchSeparator{}}
	raw, _ := json.Marshal(p)
	if _, err = s.stateStore.DB().Exec(`INSERT INTO sessions_fts(agent_id,document_id,content) VALUES(?,'turn:40',?);`, id, content); err != nil {
		t.Fatal(err)
	}
	if _, err = s.stateStore.DB().Exec(`INSERT INTO session_search_spans VALUES(?,'turn:40',1,?,?)`, id, digest, string(raw)); err != nil {
		t.Fatal(err)
	}
	w, err := transcript.Open(s.configStore.Home(), id, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	padding, _ := json.Marshal(runtime.AssistantTextData{Delta: strings.Repeat("x", 1<<20)})
	for seq := int64(1); seq < 35; seq++ {
		if err = w.Append(runtime.Event{Seq: seq, Type: runtime.EvAssistantText, Data: padding}); err != nil {
			t.Fatal(err)
		}
	}
	last, _ := json.Marshal(runtime.AssistantTextData{Delta: content})
	if err = w.Append(runtime.Event{Seq: 35, Type: runtime.EvAssistantText, Data: last}); err != nil {
		t.Fatal(err)
	}
	rec := doGET(t, s.routes(), fmt.Sprintf("/api/sessions/%s/search-match?document_id=turn:40&sha256=%s&start=0&end=6", id, digest))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"status":"unavailable"`) || !strings.Contains(rec.Body.String(), "lookup limit") {
		t.Fatalf("source budget=%d %s", rec.Code, rec.Body.String())
	}
}
