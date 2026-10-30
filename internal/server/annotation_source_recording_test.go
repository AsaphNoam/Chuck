package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AsaphNoam/Chuck/internal/config"
	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
	"github.com/AsaphNoam/Chuck/internal/transcript"
)

// FS-13.A25: real wire requests must distinguish absent source history from
// failed delivery. SQL failure after mail/activation insertion must leave a
// prepared receipt that can recover after reopening the database.
func TestAnnotationOmittedSourceRollbackRestartAndReplay(t *testing.T) {
	srv := testServer(t, true)
	source, target := writeAnnotationPair(t, srv)
	home := srv.configStore.Home()
	body := `{"command_id":"omit-1","omit_source_record":true,"annotations":[{"seq":1,"excerpt":"private side quest","instruction":"review independently"}],"target":{"kind":"agent","agent_id":"a_target"}}`
	post := func() *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		srv.routes().ServeHTTP(rec, newLocalRequest(http.MethodPost, "/api/sessions/"+source.AgentID+"/annotations", strings.NewReader(body)))
		return rec
	}
	// Fail at acceptance, after the mail and activation have been inserted.
	if _, err := srv.stateStore.DB().Exec(`CREATE TRIGGER refuse_annotation_accept BEFORE UPDATE ON annotation_receipts WHEN NEW.state='accepted' BEGIN SELECT RAISE(ABORT, 'accept unavailable'); END`); err != nil {
		t.Fatal(err)
	}
	if rec := post(); rec.Code != http.StatusInternalServerError {
		t.Fatalf("failed transaction = %d %s", rec.Code, rec.Body)
	}
	for _, table := range []string{"messages", "activations"} {
		var count int
		if err := srv.stateStore.DB().QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil || count != 0 {
			t.Fatalf("rollback %s count=%d err=%v", table, count, err)
		}
	}
	receipt, err := srv.stateStore.AnnotationReceipt(source.AgentID, "omit-1")
	if err != nil || receipt.State != "prepared" || receipt.SourceSeq != 0 || receipt.EffectID != "" {
		t.Fatalf("rollback receipt=%+v err=%v", receipt, err)
	}
	if _, err := srv.stateStore.DB().Exec(`DROP TRIGGER refuse_annotation_accept`); err != nil {
		t.Fatal(err)
	}
	reopen := func() {
		t.Helper()
		if err := srv.stateStore.Close(); err != nil {
			t.Fatal(err)
		}
		store, err := state.Open(home)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = store.Close() })
		srv = New(srv.configStore, store, runtime.NewRegistry(store), config.DefaultConfig(), srv.log)
	}
	reopen()
	// An unreadable source log distinguishes intentional omission from the
	// recorded path's mandatory reconciliation; no source scan/append is allowed.
	logPath := filepath.Join(home, "sessions", source.AgentID, transcript.LogFileName)
	if err := os.MkdirAll(logPath, 0o700); err != nil {
		t.Fatal(err)
	}
	assertAccepted := func(rec *httptest.ResponseRecorder) {
		t.Helper()
		var response struct {
			Accepted  bool   `json:"accepted"`
			Seq       *int64 `json:"seq"`
			CommandID string `json:"command_id"`
		}
		if rec.Code != http.StatusAccepted || json.Unmarshal(rec.Body.Bytes(), &response) != nil || !response.Accepted || response.Seq == nil || *response.Seq != 0 || response.CommandID != "omit-1" {
			t.Fatalf("omitted success = %d %s", rec.Code, rec.Body)
		}
	}
	assertAccepted(post())
	receipt, err = srv.stateStore.AnnotationReceipt(source.AgentID, "omit-1")
	if err != nil || receipt.State != "accepted" || receipt.SourceSeq != 0 || receipt.EffectID == "" {
		t.Fatalf("accepted receipt=%+v err=%v", receipt, err)
	}
	reopen() // Lost response, followed by a retry after server restart.
	if _, err := srv.stateStore.DB().Exec(`DELETE FROM running WHERE agent_id=?`, target.AgentID); err != nil {
		t.Fatal(err)
	}
	assertAccepted(post()) // Acceptance must win before recipient revalidation.
	mail, err := srv.stateStore.ListMessages(target.AgentID, true, 10)
	if err != nil || len(mail) != 1 || mail[0].MessageID != receipt.EffectID || !strings.Contains(mail[0].Body, "private side quest") {
		t.Fatalf("mail after replay=%+v err=%v", mail, err)
	}
	if len(srv.activationCh) != 0 {
		t.Fatal("accepted replay repeated activation")
	}
	var activationCount, sourceDocuments int
	if err := srv.stateStore.DB().QueryRow(`SELECT COUNT(*) FROM activations WHERE agent_id=?`, target.AgentID).Scan(&activationCount); err != nil || activationCount != 1 {
		t.Fatalf("durable activation count=%d err=%v", activationCount, err)
	}
	if err := srv.stateStore.DB().QueryRow(`SELECT COUNT(*) FROM sessions_fts WHERE agent_id=? AND document_id <> 'metadata'`, source.AgentID).Scan(&sourceDocuments); err != nil || sourceDocuments != 0 {
		t.Fatalf("omitted source indexed documents=%d err=%v", sourceDocuments, err)
	}
	if err := os.Remove(logPath); err != nil {
		t.Fatal(err)
	}
	events, err := transcript.ReadFile(home, source.AgentID, transcript.ReadOptions{})
	if err != nil || len(events) != 0 {
		t.Fatalf("omitted source transcript=%+v err=%v", events, err)
	}
	body = strings.Replace(body, `"omit_source_record":true`, `"omit_source_record":false`, 1)
	if rec := post(); rec.Code != http.StatusConflict {
		t.Fatalf("changed choice = %d %s", rec.Code, rec.Body)
	}
}

func TestAnnotationSourceRecordingWireValidation(t *testing.T) {
	srv := testServer(t, true)
	source, target := writeAnnotationPair(t, srv)
	base := "/api/sessions/" + source.AgentID + "/annotations"
	for _, tc := range []struct {
		name, suffix, option, command, target string
		status                                int
	}{
		{"self refusal", "", "true", `"command_id":"self-omit",`, `{"kind":"self"}`, 422},
		{"same agent refusal", "", "true", `"command_id":"same-omit",`, `{"kind":"agent","agent_id":"a_source"}`, 422},
		{"identity required", "", "true", "", `{"kind":"agent","agent_id":"a_target"}`, 422},
		{"missing recipient", "", "true", `"command_id":"missing-omit",`, `{"kind":"agent","agent_id":"absent"}`, 422},
		{"boolean required", "", `"true"`, `"command_id":"type-omit",`, `{"kind":"agent","agent_id":"a_target"}`, 422},
		{"null refused", "", "null", `"command_id":"null-omit",`, `{"kind":"agent","agent_id":"a_target"}`, 422},
		{"preflight type", "/validate", "1", "", `{"kind":"self"}`, 422},
		{"preflight before launch", "/validate", "true", "", `{"kind":"self"}`, 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := fmt.Sprintf(`{%s"omit_source_record":%s,"annotations":[{"seq":1,"excerpt":"x","instruction":"review"}],"target":%s}`, tc.command, tc.option, tc.target)
			rec := httptest.NewRecorder()
			srv.routes().ServeHTTP(rec, newLocalRequest(http.MethodPost, base+tc.suffix, strings.NewReader(body)))
			if rec.Code != tc.status || (tc.status == 422 && errorCode(t, rec) != "validation") {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body)
			}
		})
	}
	mail, err := srv.stateStore.ListMessages(target.AgentID, true, 10)
	if err != nil || len(mail) != 0 {
		t.Fatalf("invalid/preflight mail=%+v err=%v", mail, err)
	}
	events, err := transcript.ReadFile(srv.configStore.Home(), source.AgentID, transcript.ReadOptions{})
	if err != nil || len(events) != 0 {
		t.Fatalf("invalid/preflight events=%+v err=%v", events, err)
	}
}

func TestAnnotationFalseRetainsLegacyHashAndRecordedReplay(t *testing.T) {
	srv := testServer(t, true)
	source, _ := writeAnnotationPair(t, srv)
	body := `{"command_id":"legacy-false","annotations":[{"seq":1,"excerpt":"x","instruction":"review"}],"target":{"kind":"agent","agent_id":"a_target"}}`
	for _, payload := range []string{body, strings.Replace(body, `"command_id"`, `"omit_source_record":false,"command_id"`, 1)} {
		rec := httptest.NewRecorder()
		srv.routes().ServeHTTP(rec, newLocalRequest(http.MethodPost, "/api/sessions/"+source.AgentID+"/annotations", strings.NewReader(payload)))
		var response struct {
			Seq int64 `json:"seq"`
		}
		if rec.Code != 202 || json.Unmarshal(rec.Body.Bytes(), &response) != nil || response.Seq <= 0 {
			t.Fatalf("legacy/false replay=%d %s", rec.Code, rec.Body)
		}
	}
	events, err := transcript.ReadFile(srv.configStore.Home(), source.AgentID, transcript.ReadOptions{})
	if err != nil || len(events) != 1 {
		t.Fatalf("recorded events=%+v err=%v", events, err)
	}
}

func TestRemoteAnnotationOmittedSourceAndLaunchPreflight(t *testing.T) {
	srv := testServer(t, true)
	source, target := writeAnnotationPair(t, srv)
	eventsCh, unsubscribe := srv.eventBus.Subscribe()
	defer unsubscribe()
	h := srv.remoteRoutes(testDomain, testWhoIs(map[string]string{"100.64.0.2:5000": "n"}))
	token := pairTestDevice(t, srv, "omit-phone", "n")
	base := "/api/sessions/" + source.AgentID + "/annotations"
	body := `{"command_id":"phone-omit","omit_source_record":true,"annotations":[{"seq":1,"excerpt":"phone quest","instruction":"review"}],"target":{"kind":"self"}}`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, phoneRequest(http.MethodPost, base+"/validate", body, token))
	if rec.Code != http.StatusOK {
		t.Fatalf("phone preflight=%d %s", rec.Code, rec.Body)
	}
	body = strings.Replace(body, `"kind":"self"`, `"kind":"agent","agent_id":"`+target.AgentID+`"`, 1)
	for range 2 {
		rec = httptest.NewRecorder()
		h.ServeHTTP(rec, phoneRequest(http.MethodPost, base, body, token))
		var response struct {
			Seq *int64 `json:"seq"`
		}
		if rec.Code != 202 || json.Unmarshal(rec.Body.Bytes(), &response) != nil || response.Seq == nil || *response.Seq != 0 {
			t.Fatalf("phone delivery=%d %s", rec.Code, rec.Body)
		}
	}
	mail, err := srv.stateStore.ListMessages(target.AgentID, true, 10)
	if err != nil || len(mail) != 1 || !strings.Contains(mail[0].Body, "phone quest") {
		t.Fatalf("phone mail=%+v err=%v", mail, err)
	}
	events, err := transcript.ReadFile(srv.configStore.Home(), source.AgentID, transcript.ReadOptions{})
	if err != nil || len(events) != 0 {
		t.Fatalf("phone omitted events=%+v err=%v", events, err)
	}
	for {
		select {
		case event := <-eventsCh:
			if event.Type == "new_message" && event.AgentID != nil && *event.AgentID == source.AgentID {
				t.Fatalf("omitted assignment published a source message: %+v", event)
			}
		default:
			return
		}
	}
}
