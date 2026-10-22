package index

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
	"github.com/AsaphNoam/Chuck/internal/transcript"
	"github.com/mattn/go-sqlite3"
)

// Capture all columns of the four published projections, independently of the
// replay implementation. A retained native ID alone would miss partial damage.
func reindexSnapshot(t *testing.T, db *sql.DB) map[string][][]any {
	t.Helper()
	out := map[string][][]any{}
	for _, table := range []string{"sessions", "sessions_fts", "tracked_files", "tracked_commands"} {
		rows, err := db.Query(`SELECT * FROM ` + table + ` ORDER BY 1, 2`)
		if err != nil {
			t.Fatal(err)
		}
		cols, err := rows.Columns()
		if err != nil {
			t.Fatal(err)
		}
		for rows.Next() {
			values := make([]any, len(cols))
			ptrs := make([]any, len(cols))
			for i := range values {
				ptrs[i] = &values[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				t.Fatal(err)
			}
			for i, v := range values {
				if b, ok := v.([]byte); ok {
					values[i] = string(b)
				}
			}
			out[table] = append(out[table], values)
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
		rows.Close()
	}
	return out
}

func writeRepairSource(t *testing.T, home string) {
	t.Helper()
	path := filepath.Join(home, "sessions", "a_index", "transcript.ndjson")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	for _, e := range fixtureEvents(t) {
		if e.Type == runtime.EvSessionMeta {
			m := meta()
			m.Name = "Rebuilt"
			e.Data, _ = json.Marshal(m)
		}
		if err := json.NewEncoder(&b).Encode(e); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(path, b.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
}

func retainRepairAuthority(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec(`UPDATE sessions SET last_session_id='latest-native', effort='high', fast=1 WHERE agent_id='a_index'`); err != nil {
		t.Fatal(err)
	}
}

// FS-05.A27: missing and unusable sources must preserve every prior projection.
// Late malformed metadata distinguishes rollback from preflight validation only.
func TestReindexPreservesCompleteFailedProjection(t *testing.T) {
	for _, fault := range []string{"missing-root", "missing-directory", "missing-log", "unreadable-log", "empty", "no-metadata", "null-metadata", "unterminated-metadata", "late-invalid-metadata"} {
		t.Run(fault, func(t *testing.T) {
			st, home := openTestDB(t)
			indexFixture(t, st.DB())
			retainRepairAuthority(t, st.DB())
			before := reindexSnapshot(t, st.DB())
			writeRepairSource(t, home)
			path := filepath.Join(home, "sessions", "a_index", "transcript.ndjson")
			var err error
			switch fault {
			case "missing-root":
				err = os.RemoveAll(filepath.Join(home, "sessions"))
			case "missing-directory":
				err = os.RemoveAll(filepath.Dir(path))
			case "missing-log":
				err = os.Remove(path)
			case "unreadable-log":
				if err = os.Remove(path); err == nil {
					err = os.Mkdir(path, 0o700) // Real read-I/O error, including when tests run as root.
				}
			case "empty":
				err = os.WriteFile(path, nil, 0o600)
			case "no-metadata":
				err = os.WriteFile(path, []byte(`{"seq":1,"type":"assistant_text","data":{"delta":"orphan"}}`+"\n"), 0o600)
			case "null-metadata":
				err = os.WriteFile(path, []byte(`{"seq":0,"type":"session_meta","data":null}`+"\n"), 0o600)
			case "unterminated-metadata":
				e := fixtureEvents(t)[0]
				b, _ := json.Marshal(e)
				err = os.WriteFile(path, b, 0o600)
			case "late-invalid-metadata":
				f, openErr := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
				if openErr != nil {
					t.Fatal(openErr)
				}
				_, err = f.WriteString(`{"seq":7,"type":"session_meta","data":"invalid"}` + "\n")
				f.Close()
			}
			if err != nil {
				t.Fatal(err)
			}
			repairErr := Reindex(home, st.DB())
			if got := reindexSnapshot(t, st.DB()); !reflect.DeepEqual(got, before) {
				t.Fatalf("%s changed the prior projection\nbefore=%v\nafter=%v", fault, before, got)
			}
			if repairErr == nil || !strings.Contains(repairErr.Error(), "a_index") || !strings.Contains(repairErr.Error(), "preserv") {
				t.Fatalf("repair error = %v, want identity and preservation diagnostic", repairErr)
			}
			if fault == "unreadable-log" {
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			}
			writeRepairSource(t, home)
			if err := Reindex(home, st.DB()); err != nil {
				t.Fatalf("restored-source retry: %v", err)
			}
			snap, err := st.ReadSession("a_index")
			if err != nil || snap.Name != "Rebuilt" || snap.LastSessionID != "latest-native" || snap.Effort != "high" || !snap.Fast {
				t.Fatalf("repaired snapshot = %+v, %v", snap, err)
			}
			assertRepairRollups(t, st.DB())
		})
	}
}

// TS-02.R58: fail storage after replay, at retained-field restoration. This
// must roll back the entire replacement and stop before the next conversation.
func TestReindexStorageFailureBeforePublication(t *testing.T) {
	st, home := openTestDB(t)
	indexFixture(t, st.DB())
	retainRepairAuthority(t, st.DB())
	// A later healthy source must remain untouched after the storage error.
	m := meta()
	m.Name = "Not yet repaired"
	if err := New(st.DB()).UpsertSessionMeta("z_later", m); err != nil {
		t.Fatal(err)
	}
	m.Name = "Must not publish"
	w, err := transcript.Open(home, "z_later", &m)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	before := reindexSnapshot(t, st.DB())
	writeRepairSource(t, home)
	if _, err := st.DB().Exec(`CREATE TRIGGER fail_repair BEFORE UPDATE OF effort ON sessions
WHEN NEW.effort='high' BEGIN SELECT RAISE(ABORT, 'injected restore failure'); END`); err != nil {
		t.Fatal(err)
	}
	if err := Reindex(home, st.DB()); err == nil || !strings.Contains(err.Error(), "injected restore failure") {
		t.Fatalf("repair error = %v", err)
	}
	if got := reindexSnapshot(t, st.DB()); !reflect.DeepEqual(got, before) {
		t.Fatalf("storage failure published partial state\nbefore=%v\nafter=%v", before, got)
	}
}

// A source-only replay must not leave rows written before later bad metadata.
func TestReindexFailedNewSourceLeavesNoPartialRows(t *testing.T) {
	st, home := openTestDB(t)
	writeRepairSource(t, home)
	path := filepath.Join(home, "sessions", "a_index", "transcript.ndjson")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.WriteString(`{"seq":7,"type":"session_meta","data":"invalid"}` + "\n")
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	if err := Reindex(home, st.DB()); err == nil || !strings.Contains(err.Error(), "no prior snapshot") {
		t.Fatalf("repair error = %v", err)
	}
	if got := reindexSnapshot(t, st.DB()); len(got) != 0 {
		t.Fatalf("failed new source left partial rows: %v", got)
	}
}

// FS-05.A26 / TS-02.R58: a real killed process proves SQLite recovery, rather
// than deferred Rollback.
// The trigger pauses only after replay has completed and retained fields are
// being restored, while the transaction still holds every destructive write.
func TestReindexKilledBeforePublication(t *testing.T) {
	if home := os.Getenv("CHUCK_TEST_REINDEX_KILL_HOME"); home != "" {
		sql.Register("reindex-kill", &sqlite3.SQLiteDriver{ConnectHook: func(c *sqlite3.SQLiteConn) error {
			return c.RegisterFunc("pause_repair", func() int {
				if err := os.WriteFile(filepath.Join(home, "repair-paused"), []byte("ready"), 0o600); err != nil {
					panic(err)
				}
				select {} // Parent kills this process; no rollback defer can run.
			}, false)
		}})
		db, err := sql.Open("reindex-kill", filepath.Join(home, "state.db"))
		if err != nil {
			t.Fatal(err)
		}
		db.SetMaxOpenConns(1)
		if _, err := db.Exec(`CREATE TRIGGER pause_repair BEFORE UPDATE OF effort ON sessions
WHEN NEW.effort='high' BEGIN SELECT pause_repair(); END`); err != nil {
			t.Fatal(err)
		}
		if err := Reindex(home, db); err != nil {
			t.Fatal(err)
		}
		t.Fatal("repair returned without hitting the publication barrier")
	}
	st, home := openTestDB(t)
	indexFixture(t, st.DB())
	retainRepairAuthority(t, st.DB())
	before := reindexSnapshot(t, st.DB())
	writeRepairSource(t, home)
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe, "-test.run=^TestReindexKilledBeforePublication$", "-test.timeout=20s")
	cmd.Env = append(os.Environ(), "CHUCK_TEST_REINDEX_KILL_HOME="+home)
	var output bytes.Buffer
	cmd.Stdout, cmd.Stderr = &output, &output
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	t.Cleanup(func() { _ = cmd.Process.Kill() })
	deadline := time.NewTimer(15 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()
	waiting := true
	for waiting {
		select {
		case err := <-done:
			t.Fatalf("child exited before barrier: %v\n%s", err, output.String())
		case <-deadline.C:
			t.Fatal("child never reached publication barrier")
		case <-tick.C:
			_, err := os.Stat(filepath.Join(home, "repair-paused"))
			waiting = err != nil
		}
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	<-done
	reopened, err := state.Open(home)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if got := reindexSnapshot(t, reopened.DB()); !reflect.DeepEqual(got, before) {
		t.Fatalf("process kill published partial state\nbefore=%v\nafter=%v", before, got)
	}
	if _, err := reopened.DB().Exec(`DROP TRIGGER pause_repair`); err != nil {
		t.Fatal(err)
	}
	if err := Reindex(home, reopened.DB()); err != nil {
		t.Fatal(err)
	}
	snap, err := reopened.ReadSession("a_index")
	if err != nil || snap.Name != "Rebuilt" || snap.LastSessionID != "latest-native" || snap.Effort != "high" || !snap.Fast {
		t.Fatalf("post-kill retry snapshot = %+v, %v", snap, err)
	}
	assertRepairRollups(t, reopened.DB())
}

func assertRepairRollups(t *testing.T, db *sql.DB) {
	t.Helper()
	var turns, files, commands int
	if err := db.QueryRow(`SELECT turn_count, files_touched, commands_run FROM sessions WHERE agent_id='a_index'`).Scan(&turns, &files, &commands); err != nil {
		t.Fatal(err)
	}
	if turns != 1 || files != 1 || commands != 1 {
		t.Fatalf("rollups = %d/%d/%d, want 1/1/1", turns, files, commands)
	}
}
