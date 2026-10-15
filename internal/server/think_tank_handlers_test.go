package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/attachment"
	"github.com/AsaphNoam/Chuck/internal/config"
	"github.com/AsaphNoam/Chuck/internal/state"
)

func roomRESTServer(t *testing.T) (*Server, http.Handler) {
	t.Helper()
	srv := testServer(t, true)
	for _, p := range []string{"alpha", "beta", "old"} {
		if err := srv.configStore.WriteProject(p, config.Project{Title: p, Cwd: t.TempDir(), Archived: p == "old"}); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now().UTC()
	agents := []state.Agent{
		{AgentID: "a_one", Name: "Ari", Role: "impl", Project: "alpha", Backend: "claude", Interface: "chat", CreatedAt: now},
		{AgentID: "a_two", Name: "Bea", Role: "impl", Project: "beta", Backend: "claude", Interface: "chat", CreatedAt: now},
		{AgentID: "a_term", Name: "Term", Role: "impl", Project: "alpha", Backend: "claude", Interface: "terminal", CreatedAt: now},
		{AgentID: "a_arch", Name: "Arc", Role: "impl", Project: "alpha", Backend: "claude", Interface: "chat", CreatedAt: now, Archived: true},
		{AgentID: "a_old", Name: "Olde", Role: "impl", Project: "old", Backend: "claude", Interface: "chat", CreatedAt: now},
	}
	for _, a := range agents {
		if err := srv.stateStore.WriteAgent(a); err != nil {
			t.Fatal(err)
		}
	}
	return srv, srv.routes()
}

func roomBody(participants ...string) string {
	parts := []string{}
	for _, p := range participants {
		parts = append(parts, `{"agent_id":"`+p+`","limit":2}`)
	}
	return `{"command_id":"cmd-1","goal":"Pick a cache","origin_project":"alpha","participants":[` + strings.Join(parts, ",") + `]}`
}

// FS-21.A14, A24: setup validation over REST, mixed projects, defaults, and
// an exact replay returning the original room.
func TestThinkTankCreateOverREST(t *testing.T) {
	_, h := roomRESTServer(t)
	for name, body := range map[string]string{
		"duplicate":    roomBody("a_one", "a_one"),
		"terminal":     roomBody("a_one", "a_term"),
		"archived":     roomBody("a_one", "a_arch"),
		"old project":  roomBody("a_one", "a_old"),
		"insufficient": roomBody("a_one"),
		"missing":      roomBody("a_one", "a_nope"),
	} {
		if rec := doJSON(t, h, http.MethodPost, "/api/think-tanks", body); rec.Code == http.StatusCreated {
			t.Fatalf("%s: created %s", name, rec.Body.String())
		}
	}
	if rec := doJSON(t, h, http.MethodPost, "/api/think-tanks", strings.Replace(roomBody("a_one", "a_two"), `"alpha"`, `"old"`, 1)); rec.Code == http.StatusCreated {
		t.Fatal("archived origin project accepted")
	}
	rec := doJSON(t, h, http.MethodPost, "/api/think-tanks", roomBody("a_one", "a_two"))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create = %d %s", rec.Code, rec.Body.String())
	}
	var room thinkTankDetailWire
	if err := json.Unmarshal(rec.Body.Bytes(), &room); err != nil {
		t.Fatal(err)
	}
	if room.Phase != state.ThinkTankPhaseDiscussion || room.Openings || room.Judge.Enabled || len(room.Members) != 2 ||
		!room.Members[0].MayLeave || room.Members[1].Project != "beta" {
		t.Fatalf("defaults = %+v", room)
	}
	again := doJSON(t, h, http.MethodPost, "/api/think-tanks", roomBody("a_one", "a_two"))
	var replay thinkTankDetailWire
	_ = json.Unmarshal(again.Body.Bytes(), &replay)
	if again.Code != http.StatusCreated || replay.RoomID != room.RoomID {
		t.Fatalf("replay = %d %s", again.Code, replay.RoomID)
	}
	list := doJSON(t, h, http.MethodGet, "/api/think-tanks?project=alpha", "")
	if !strings.Contains(list.Body.String(), room.RoomID) {
		t.Fatalf("project list = %s", list.Body.String())
	}
	if other := doJSON(t, h, http.MethodGet, "/api/think-tanks?project=beta", ""); strings.Contains(other.Body.String(), room.RoomID) {
		t.Fatal("room listed under a participant's project rather than its origin")
	}
}

func TestThinkTankCreateReplaysReservedParticipants(t *testing.T) {
	for _, participants := range []string{
		`{"agent_id":"a_one","limit":2},{"new":{"name":"New","project":"alpha","backend":"claude"},"limit":3}`,
		`{"new":{"name":"First","project":"alpha","backend":"claude"},"limit":2},{"new":{"name":"Second","project":"beta","backend":"claude"},"limit":3}`,
	} {
		t.Run(participants, func(t *testing.T) {
			_, h := roomRESTServer(t)
			body := `{"command_id":"new-command","goal":"Choose","origin_project":"alpha","participants":[` + participants + `]}`
			first := doJSON(t, h, http.MethodPost, "/api/think-tanks", body)
			if first.Code != http.StatusCreated {
				t.Fatalf("create: %d %s", first.Code, first.Body.String())
			}
			var room thinkTankDetailWire
			_ = json.Unmarshal(first.Body.Bytes(), &room)
			second := doJSON(t, h, http.MethodPost, "/api/think-tanks", body)
			var replay thinkTankDetailWire
			_ = json.Unmarshal(second.Body.Bytes(), &replay)
			if second.Code != http.StatusCreated || replay.RoomID != room.RoomID || len(replay.Members) != 2 {
				t.Fatalf("replay: %d %s", second.Code, second.Body.String())
			}
			for i := range room.Members {
				if replay.Members[i].AgentID != room.Members[i].AgentID {
					t.Fatal("replay replaced reserved identity")
				}
			}
			changed := strings.Replace(body, `"goal":"Choose"`, `"goal":"Different"`, 1)
			if got := doJSON(t, h, http.MethodPost, "/api/think-tanks", changed); got.Code != http.StatusConflict {
				t.Fatalf("changed intent: %d %s", got.Code, got.Body.String())
			}
		})
	}
}

// FS-21.A9, A30: shared input, pause/resume/end and guarded deletion over
// REST; deletion leaves participant agents intact.
func TestThinkTankControlsOverREST(t *testing.T) {
	srv, h := roomRESTServer(t)
	rec := doJSON(t, h, http.MethodPost, "/api/think-tanks", roomBody("a_one", "a_two"))
	var room thinkTankDetailWire
	_ = json.Unmarshal(rec.Body.Bytes(), &room)
	base := "/api/think-tanks/" + room.RoomID

	if rec := doJSON(t, h, http.MethodPost, base+"/messages", `{"command_id":"m1","body":"Consider eviction"}`); rec.Code != http.StatusOK {
		t.Fatalf("message = %d %s", rec.Code, rec.Body.String())
	}
	entries := doJSON(t, h, http.MethodGet, base+"/entries", "")
	if !strings.Contains(entries.Body.String(), "Consider eviction") || !strings.Contains(entries.Body.String(), `"complete":true`) {
		t.Fatalf("entries = %s", entries.Body.String())
	}
	if rec := doJSON(t, h, http.MethodDelete, base, ""); rec.Code != http.StatusConflict {
		t.Fatalf("delete while running = %d", rec.Code)
	}
	if rec := doJSON(t, h, http.MethodPost, base+"/pause", ""); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"control":"paused"`) {
		t.Fatalf("pause = %d %s", rec.Code, rec.Body.String())
	}
	if rec := doJSON(t, h, http.MethodPost, base+"/resume", ""); rec.Code != http.StatusOK {
		t.Fatalf("resume = %d", rec.Code)
	}
	if rec := doJSON(t, h, http.MethodPost, base+"/end", ""); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"end_reason":"operator"`) {
		t.Fatalf("end = %d %s", rec.Code, rec.Body.String())
	}
	if rec := doJSON(t, h, http.MethodPost, base+"/messages", `{"command_id":"m2","body":"late"}`); rec.Code != http.StatusConflict {
		t.Fatalf("room message after end = %d", rec.Code)
	}
	if rec := doJSON(t, h, http.MethodPost, base+"/retry", `{"target":"bogus"}`); rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("bad retry target = %d", rec.Code)
	}
	if rec := doJSON(t, h, http.MethodDelete, base, ""); rec.Code != http.StatusNoContent {
		t.Fatalf("delete = %d %s", rec.Code, rec.Body.String())
	}
	if rec := doJSON(t, h, http.MethodGet, base, ""); rec.Code != http.StatusNotFound {
		t.Fatalf("deleted room = %d", rec.Code)
	}
	for _, id := range []string{"a_one", "a_two"} {
		if _, err := srv.stateStore.ReadAgent(id); err != nil {
			t.Fatalf("participant %s deleted with the room: %v", id, err)
		}
	}
}

// thinkTankWireFixturePath is the room payload the UI tests read. Go's encoder
// defines its shape (INV §11, §17); regenerate with
// CHUCK_UPDATE_THINK_TANK_FIXTURE=1 after a deliberate change.
const thinkTankWireFixturePath = "../../ui/src/features/thinktank/fixtures/room.json"

func TestThinkTankWireFixtureMatchesServerEncoding(t *testing.T) {
	srv, _ := roomRESTServer(t)
	at := time.Date(2026, 10, 6, 9, 0, 0, 0, time.UTC)
	d := state.ThinkTankDetail{
		Room: state.ThinkTank{RoomID: "tt_fixture", Title: "Cache choice", Goal: "Pick a cache", OriginProject: "alpha",
			Phase: state.ThinkTankPhaseDiscussion, Control: state.ThinkTankRunning, Revision: 7,
			JudgeConfig: `{"role":"impl","project":"alpha","interface":"chat"}`, JudgeStatus: state.ThinkTankJudgeWaiting,
			CreatedAt: at, UpdatedAt: at},
		Members: []state.ThinkTankMember{
			{AgentID: "a_one", AgentName: "Ari", Project: "alpha", Role: state.ThinkTankRoleParticipant, Order: 0, Cap: 3, Completed: 1, MayLeave: true, State: state.ThinkTankMemberActive, SetupState: state.ThinkTankSetupReady},
			{AgentID: "a_gone", AgentName: "Gone", Project: "beta", Role: state.ThinkTankRoleParticipant, Order: 1, Cap: 2, Completed: 2, State: state.ThinkTankMemberExhausted, SetupState: state.ThinkTankSetupReady},
		},
		Pending:  []state.ThinkTankInput{},
		Attempts: []state.ThinkTankAttempt{},
	}
	detail := srv.thinkTankDetailWire(d)
	entries := []thinkTankEntryWire{
		thinkTankEntryFor(state.ThinkTankEntry{Seq: 1, Kind: state.ThinkTankEntryReply, AgentID: "a_one", AgentName: "Ari", Project: "alpha", Body: "Use **LRU**.", AttemptID: "tta_1", CreatedAt: at}),
		thinkTankEntryFor(state.ThinkTankEntry{Seq: 2, Kind: state.ThinkTankEntryUser, Body: "Consider eviction", InputID: "tti_1", CreatedAt: at}),
	}
	fixture := map[string]any{"room": detail, "entries": entries}
	got, err := json.MarshalIndent(fixture, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, '\n')
	if !bytes.Contains(got, []byte(`"pending": []`)) || !bytes.Contains(got, []byte(`"failed": []`)) {
		t.Fatalf("collections must encode as arrays: %s", got)
	}
	if os.Getenv("CHUCK_UPDATE_THINK_TANK_FIXTURE") == "1" {
		if err := os.MkdirAll("../../ui/src/features/thinktank/fixtures", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(thinkTankWireFixturePath, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(thinkTankWireFixturePath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("%s is stale; regenerate with CHUCK_UPDATE_THINK_TANK_FIXTURE=1", thinkTankWireFixturePath)
	}
}

// FS-21.A36: a ceiling increase over REST returns the updated room, replays
// its command and reports a stale expected ceiling as a conflict.
func TestThinkTankTurnLimitOverREST(t *testing.T) {
	_, h := roomRESTServer(t)
	rec := doJSON(t, h, http.MethodPost, "/api/think-tanks", roomBody("a_one", "a_two"))
	var room thinkTankDetailWire
	_ = json.Unmarshal(rec.Body.Bytes(), &room)
	url := "/api/think-tanks/" + room.RoomID + "/participants/a_one/turn-limit"
	body := `{"command_id":"lim","expected_limit":2,"limit":5}`
	for i := 0; i < 2; i++ {
		got := doJSON(t, h, http.MethodPost, url, body)
		if got.Code != http.StatusOK || !strings.Contains(got.Body.String(), `"limit":5`) {
			t.Fatalf("increase %d = %d %s", i, got.Code, got.Body.String())
		}
	}
	if got := doJSON(t, h, http.MethodPost, url, `{"command_id":"lim2","expected_limit":2,"limit":6}`); got.Code != http.StatusConflict {
		t.Fatalf("stale = %d %s", got.Code, got.Body.String())
	}
	if got := doJSON(t, h, http.MethodPost, url, `{"command_id":"lim3","expected_limit":5,"limit":4}`); got.Code != http.StatusUnprocessableEntity {
		t.Fatalf("decrease = %d %s", got.Code, got.Body.String())
	}
}

// FS-21.A37, TS-14.R26: the judge's retained results read through its
// session route; a deleted room is marked unavailable, not dropped.
func TestThinkTankResultsRoute(t *testing.T) {
	srv, h := roomRESTServer(t)
	rec := doJSON(t, h, http.MethodPost, "/api/think-tanks", roomBody("a_one", "a_two"))
	var room thinkTankDetailWire
	_ = json.Unmarshal(rec.Body.Bytes(), &room)
	for i, id := range []string{room.RoomID, "tt_deleted"} {
		if _, err := srv.stateStore.DB().Exec(`INSERT INTO think_tank_results(agent_id, room_id, room_title, entry_seq, attempt_id, body, generation, turn_id, event_seq, completed_at)
VALUES('a_one', ?, 'Title', ?, 'att', 'Exact synthesis', 'g', 't', 7, '2026-10-07T00:00:00Z')`, id, i+1); err != nil {
			t.Fatal(err)
		}
	}
	got := doJSON(t, h, http.MethodGet, "/api/sessions/a_one/think-tank-results", "")
	var body struct {
		Results  []thinkTankResultWire `json:"results"`
		Complete bool                  `json:"complete"`
	}
	_ = json.Unmarshal(got.Body.Bytes(), &body)
	if got.Code != http.StatusOK || !body.Complete || len(body.Results) != 2 || !body.Results[0].RoomAvailable ||
		body.Results[1].RoomAvailable || body.Results[0].Body != "Exact synthesis" || body.Results[0].EventSeq != 7 {
		t.Fatalf("results = %d %s", got.Code, got.Body.String())
	}
	if other := doJSON(t, h, http.MethodGet, "/api/sessions/a_two/think-tank-results", ""); !strings.Contains(other.Body.String(), `"results":[]`) {
		t.Fatalf("other agent results = %s", other.Body.String())
	}
}

// FS-21.A33, TS-14.R27: the membership filter finds a participant's rooms
// from any project, and summaries carry the card roster and allowances.
func TestThinkTankListByMemberWithRoster(t *testing.T) {
	_, h := roomRESTServer(t)
	rec := doJSON(t, h, http.MethodPost, "/api/think-tanks", roomBody("a_one", "a_two"))
	var room thinkTankDetailWire
	_ = json.Unmarshal(rec.Body.Bytes(), &room)
	var list struct {
		Rooms   []thinkTankSummaryWire `json:"rooms"`
		Clipped bool                   `json:"clipped"`
	}
	got := doJSON(t, h, http.MethodGet, "/api/think-tanks?agent_id=a_two", "")
	_ = json.Unmarshal(got.Body.Bytes(), &list)
	if len(list.Rooms) != 1 || list.Clipped || list.Rooms[0].RoomID != room.RoomID || list.Rooms[0].TotalRemaining != 4 ||
		len(list.Rooms[0].Roster) != 2 || list.Rooms[0].Roster[1].Remaining != 2 || !list.Rooms[0].Roster[1].Exists {
		t.Fatalf("member list = %s", got.Body.String())
	}
	if none := doJSON(t, h, http.MethodGet, "/api/think-tanks?agent_id=a_nobody", ""); !strings.Contains(none.Body.String(), `"rooms":[]`) {
		t.Fatalf("non-member list = %s", none.Body.String())
	}
}

func roomMultipart(t *testing.T, h http.Handler, room string, files map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	decl := []string{}
	for i, name := range names {
		decl = append(decl, fmt.Sprintf(`{"part":"file_%d","name":%q}`, i, name))
	}
	meta, _ := w.CreateFormField("room")
	_, _ = meta.Write([]byte(strings.TrimSuffix(room, "}") + `,"files":[` + strings.Join(decl, ",") + `]}`))
	for i, name := range names {
		part, _ := w.CreateFormFile(fmt.Sprintf("file_%d", i), name)
		_, _ = part.Write([]byte(files[name]))
	}
	_ = w.Close()
	req := newLocalRequest(http.MethodPost, "/api/think-tanks", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func roomFileCount(t *testing.T, srv *Server, commandID string) int {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(srv.configStore.Home(), "sessions", state.ThinkTankAttachmentKey(commandID), "attachments"))
	if os.IsNotExist(err) {
		return 0
	}
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), ".") {
			n++
		}
	}
	return n
}

// FS-21.A39, TS-14.R29: multipart setup stores room-owned goal files and
// judge-only instructions; replay keeps the first copies; refusals and
// deletion leave no files behind.
func TestThinkTankCreateWithGoalFilesAndJudgeInstructions(t *testing.T) {
	srv, h := roomRESTServer(t)
	files := map[string]string{"brief.md": "# Brief", "data.csv": "a,b"}
	judged := `{"command_id":"cmd-files","goal":"Pick a cache","origin_project":"alpha","judge":{"project":"alpha","backend":"claude"},` +
		`"judge_instructions":"Rank the options.","participants":[{"agent_id":"a_one","limit":2},{"agent_id":"a_two","limit":2}]}`
	rec := roomMultipart(t, h, judged, files)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create = %d %s", rec.Code, rec.Body.String())
	}
	var room thinkTankDetailWire
	if err := json.Unmarshal(rec.Body.Bytes(), &room); err != nil {
		t.Fatal(err)
	}
	if len(room.Attachments) != 2 || room.Attachments[0].Name != "brief.md" || room.Judge.Instructions != "Rank the options." {
		t.Fatalf("room = %+v", room)
	}
	if again := roomMultipart(t, h, judged, files); again.Code != http.StatusCreated || !strings.Contains(again.Body.String(), room.RoomID) {
		t.Fatalf("replay = %d %s", again.Code, again.Body.String())
	}
	if n := roomFileCount(t, srv, "cmd-files"); n != 2 {
		t.Fatalf("stored files after replay = %d, want 2", n)
	}

	noJudge := `{"command_id":"cmd-nojudge","goal":"Pick","origin_project":"alpha","judge_instructions":"x","participants":[{"agent_id":"a_one","limit":2},{"agent_id":"a_two","limit":2}]}`
	if rec := roomMultipart(t, h, noJudge, files); rec.Code == http.StatusCreated || roomFileCount(t, srv, "cmd-nojudge") != 0 {
		t.Fatalf("instructions without a judge = %d, files %d", rec.Code, roomFileCount(t, srv, "cmd-nojudge"))
	}
	big := `{"command_id":"cmd-big","goal":"Pick","origin_project":"alpha","participants":[{"agent_id":"a_one","limit":2},{"agent_id":"a_two","limit":2}]}`
	if rec := roomMultipart(t, h, big, map[string]string{"huge.txt": strings.Repeat("x", attachment.MaxFileBytes+1)}); rec.Code == http.StatusCreated || roomFileCount(t, srv, "cmd-big") != 0 {
		t.Fatalf("oversized file = %d, files %d", rec.Code, roomFileCount(t, srv, "cmd-big"))
	}

	plain := strings.Replace(big, "cmd-big", "cmd-plain", 1)
	rec = roomMultipart(t, h, plain, files)
	var deletable thinkTankDetailWire
	_ = json.Unmarshal(rec.Body.Bytes(), &deletable)
	if rec.Code != http.StatusCreated || roomFileCount(t, srv, "cmd-plain") != 2 {
		t.Fatalf("plain create = %d %s", rec.Code, rec.Body.String())
	}
	if _, err := srv.stateStore.EndThinkTank(deletable.RoomID); err != nil {
		t.Fatal(err)
	}
	if rec := doJSON(t, h, http.MethodDelete, "/api/think-tanks/"+deletable.RoomID, ""); rec.Code != http.StatusNoContent {
		t.Fatalf("delete = %d %s", rec.Code, rec.Body.String())
	}
	if n := roomFileCount(t, srv, "cmd-plain"); n != 0 {
		t.Fatalf("files after delete = %d", n)
	}
}
