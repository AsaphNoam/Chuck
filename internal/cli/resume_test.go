package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"testing"
)

func archivePage(t *testing.T, total int, results []archiveResult) []byte {
	t.Helper()
	b, err := json.Marshal(struct {
		Total   int             `json:"total"`
		Results []archiveResult `json:"results"`
	}{total, results})
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// FS-01.R11 requires complete evidence before automatic resume or new launch.
// The HTTP fixture observes the forbidden side effects on lookup uncertainty.
func TestRunLaunchArchiveSelectionBoundary(t *testing.T) {
	tests := []struct {
		name                   string
		mode                   string
		code                   int
		wantLaunch, wantResume bool
	}{
		{name: "http failure", mode: "error", code: 1},
		{name: "late unique", mode: "late", code: 0, wantResume: true},
		{name: "duplicate across pages", mode: "duplicate", code: 1},
		{name: "complete absence", mode: "absence", code: 0, wantLaunch: true},
		{name: "bounded lookup", mode: "bound", code: 1},
		{name: "explicit new", mode: "new", code: 0, wantLaunch: true},
		{name: "incomplete page", mode: "short", code: 1},
		{name: "missing total", mode: "missing", code: 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var launches, resumes, pages int
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/api/archive/projects/app":
					pages++
					if tc.mode == "missing" {
						fmt.Fprint(w, `{"results":[]}`)
						return
					}
					if tc.mode == "error" {
						http.Error(w, "broken", http.StatusInternalServerError)
						return
					}
					total := 201
					if tc.mode == "bound" {
						total = archiveLookupPageSize*archiveLookupMaxPages + 1
					}
					offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
					rows := make([]archiveResult, 0, archiveLookupPageSize)
					if offset == 0 && (tc.mode == "late" || tc.mode == "duplicate" || tc.mode == "absence") {
						rows = make([]archiveResult, archiveLookupPageSize)
					}
					if tc.mode == "duplicate" && offset == 0 {
						rows[0] = archiveResult{AgentID: "first", Role: "impl", Project: "app"}
					}
					if tc.mode == "late" && offset > 0 {
						rows = []archiveResult{{AgentID: "late", Role: "impl", Project: "app"}}
					}
					if tc.mode == "duplicate" && offset > 0 {
						rows = []archiveResult{{AgentID: "second", Role: "impl", Project: "app"}}
					}
					if tc.mode == "absence" && offset > 0 {
						rows = []archiveResult{{AgentID: "other", Role: "qa", Project: "app"}}
					}
					if tc.mode == "bound" {
						rows = make([]archiveResult, archiveLookupPageSize)
					}
					if tc.mode == "short" {
						rows = []archiveResult{{AgentID: "first", Role: "impl", Project: "app"}}
					}
					fmt.Fprint(w, string(archivePage(t, total, rows)))
				case "/api/sessions":
					launches++
					w.WriteHeader(http.StatusCreated)
					fmt.Fprint(w, `{"agent":{"agent_id":"new","name":"New"}}`)
				case "/api/sessions/late/resume", "/api/sessions/first/resume":
					resumes++
					fmt.Fprint(w, `{"agent":{"agent_id":"resumed","name":"Old","role":"impl","project":"app"}}`)
				default:
					t.Errorf("unexpected request: %s %s", r.Method, r.URL)
				}
			}))
			defer server.Close()
			u, _ := url.Parse(server.URL)
			port, _ := strconv.Atoi(u.Port())
			home := t.TempDir()
			t.Setenv("CHUCK_HOME", home)
			if err := writePidfile(home, pidInfo{PID: os.Getpid(), Port: port}); err != nil {
				t.Fatal(err)
			}
			args := []string{"impl@app"}
			if tc.mode == "new" {
				args = append(args, "--new")
			}
			if got := runLaunch(args); got != tc.code {
				t.Fatalf("exit=%d want %d", got, tc.code)
			}
			if launches != boolInt(tc.wantLaunch) || resumes != boolInt(tc.wantResume) {
				t.Fatalf("launches=%d resumes=%d, want %d/%d (pages=%d)", launches, resumes, boolInt(tc.wantLaunch), boolInt(tc.wantResume), pages)
			}
			if tc.mode == "new" && pages != 0 {
				t.Fatalf("--new made %d archive requests", pages)
			}
		})
	}
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

// TestListInactiveSessionsNameFilter guards the advisory finding that bare-form
// auto-resume must respect --name: with a name given, only the session actually
// named that matches; without a name, all role@project inactive sessions match.
func TestListInactiveSessionsNameFilter(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.RequestURI() != "/api/archive/projects/app?active=false&limit=200" {
			t.Errorf("legacy archive wire %s %s", r.Method, r.URL)
		}
		fmt.Fprint(w, `{"total":4,"results":[
			{"agent_id":"a_1","name":"Atlas","role":"impl","project":"app","active":false},
			{"agent_id":"a_2","name":"Borg","role":"impl","project":"app","active":false},
			{"agent_id":"a_3","name":"Atlas","role":"impl","project":"other","active":false},
			{"agent_id":"a_4","name":"Live","role":"impl","project":"app","active":true}
		]}`)
	}))
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	port, _ := strconv.Atoi(u.Port())

	// No name → all inactive impl@app sessions.
	all, err := listInactiveSessions(port, "impl", "app", "")
	if err != nil {
		t.Fatalf("listInactiveSessions: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("no-name match = %d sessions, want 2 (a_1,a_2)", len(all))
	}

	// With --name Atlas → only the impl@app session named Atlas (a_1), not Borg,
	// and not the impl@other Atlas or the active one.
	named, err := listInactiveSessions(port, "impl", "app", "Atlas")
	if err != nil {
		t.Fatalf("listInactiveSessions (named): %v", err)
	}
	if len(named) != 1 || named[0].AgentID != "a_1" {
		t.Fatalf("named match = %+v, want exactly a_1", named)
	}
}
