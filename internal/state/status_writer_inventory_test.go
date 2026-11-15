package state

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// statusTableWriters pins how many UPDATE statements write each lifecycle
// table in each non-test source file under internal/. These three tables carry
// status columns whose legal moves no single function owns (CR-A10), so every
// new writer is another place a transition rule can diverge (INV §2, §5).
//
// When this test fails because a count rose: route the change through an
// existing writer if one fits, otherwise raise the number here in the same
// commit and say why the new writer is needed. When a count fell, lower it —
// the inventory only ratchets down for free.
var statusTableWriters = map[string]map[string]int{
	"quota_intents": {
		"internal/index/indexer.go":          1,
		"internal/state/quota.go":            17, // Includes the due-owner hold, fenced before a claim exists; MarkQuotaNoticeConfirmed was uncalled and removed (TS-01.R50).
		"internal/state/quota_checks.go":     4,
		"internal/state/quota_controls.go":   1,
		"internal/state/quota_success.go":    1,
		"internal/state/think_tank_turns.go": 1,
	},
	"tasks": {
		"internal/state/attachment_receipts.go": 1,
		"internal/state/pipeline_tasks.go":      4,
		"internal/state/quota.go":               1,
		"internal/state/quota_admission.go":     2,
		"internal/state/task_time.go":           2,
		"internal/state/task_waits.go":          4,
		"internal/state/tasks.go":               17,
		"internal/state/schema.go":              1, // Migration clears historic terminal task markers.
	},
	"pipeline_runs": {
		"internal/state/pipeline_decisions.go": 1,
		"internal/state/pipeline_tasks.go":     13,
		"internal/state/pipelines.go":          3,
		"internal/state/schema.go":             1,
		"internal/state/task_waits.go":         1,
		"internal/state/tasks.go":              1,
	},
}

func TestStatusTableWriterInventory(t *testing.T) {
	root := filepath.Join("..", "..")
	got := map[string]map[string]int{}
	patterns := map[string]*regexp.Regexp{}
	for table := range statusTableWriters {
		got[table] = map[string]int{}
		patterns[table] = regexp.MustCompile(`UPDATE\s+` + table + `\b`)
	}

	err := filepath.WalkDir(filepath.Join(root, "internal"), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		for table, re := range patterns {
			if n := len(re.FindAll(src, -1)); n > 0 {
				got[table][rel] = n
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk source: %v", err)
	}

	var diffs []string
	for table, want := range statusTableWriters {
		files := map[string]bool{}
		for f := range want {
			files[f] = true
		}
		for f := range got[table] {
			files[f] = true
		}
		for f := range files {
			if got[table][f] != want[f] {
				diffs = append(diffs, fmt.Sprintf("%s in %s: inventory says %d, source has %d", table, f, want[f], got[table][f]))
			}
		}
	}
	if len(diffs) > 0 {
		sort.Strings(diffs)
		t.Fatalf("status-table writers changed:\n  %s\nA new writer should reuse an existing one where it fits; otherwise update statusTableWriters in this commit and say why (INV §2).", strings.Join(diffs, "\n  "))
	}
}
