package server

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// TestDerivedEffectsAndBoundedPasses enumerates each control effect that follows
// a commit with the periodic pass that recomputes it from durable rows, and each
// bounded pass with its progress rule (TS-01.R49). A post-commit call may wake
// a pass but carries no meaning of its own, so removing a pass from its loop, or
// a cursor from its pass, is the regression this guards.
func TestDerivedEffectsAndBoundedPasses(t *testing.T) {
	funcs := serverFuncFacts(t)
	for _, tc := range []struct {
		effect, loop, pass, reads string
		// cursor is the field the pass advances for every visited row; empty
		// when each visited row instead changes or gets a later due time.
		cursor string
	}{
		{"park the dependents of a parked task", "runTaskDispatcher", "reconcileParkedTaskSources", "ParkedTaskSourcesAfter", "parkedTaskAfter"},
		{"project a room phase onto its run", "startThinkTanks", "syncPipelineRoomPhases", "SyncRoomPhase", "roomPhaseAfter"},
		{"release a runtime after a reported or yielded turn", "runTaskDispatcher", "reconcileTaskCleanup", "DueTaskCleanup", ""},
		{"continue a due quota intent", "runTaskDispatcher", "dispatchQuotaContinuations", "DueQuotaIntentsAfter", "quotaDueAfter"},
		{"start a ready task", "runTaskDispatcher", "dispatchReadyTasks", "", "taskSweepAfter"},
		{"recover a missed stage stall", "runTaskDispatcher", "recoverMissedStageStalls", "", "pipelineRecoveryAfter"},
	} {
		t.Run(tc.effect, func(t *testing.T) {
			loop, pass := funcs[tc.loop], funcs[tc.pass]
			if loop == nil || pass == nil {
				t.Fatalf("loop %s or pass %s not found", tc.loop, tc.pass)
			}
			if !loop.calls[tc.pass] {
				t.Errorf("%s does not run %s", tc.loop, tc.pass)
			}
			if tc.reads != "" && !pass.calls[tc.reads] {
				t.Errorf("%s does not recompute from %s", tc.pass, tc.reads)
			}
			if tc.cursor != "" && !pass.assigns[tc.cursor] {
				t.Errorf("%s does not advance its cursor %s", tc.pass, tc.cursor)
			}
		})
	}
}

type funcFacts struct{ calls, assigns map[string]bool }

func serverFuncFacts(t *testing.T) map[string]*funcFacts {
	t.Helper()
	paths, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]*funcFacts{}
	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || fn.Body == nil {
				continue
			}
			facts := &funcFacts{calls: map[string]bool{}, assigns: map[string]bool{}}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				switch n := n.(type) {
				case *ast.CallExpr:
					if sel, ok := n.Fun.(*ast.SelectorExpr); ok {
						facts.calls[sel.Sel.Name] = true
					}
				case *ast.AssignStmt:
					for _, lhs := range n.Lhs {
						if sel, ok := lhs.(*ast.SelectorExpr); ok {
							facts.assigns[sel.Sel.Name] = true
						}
					}
				}
				return true
			})
			out[fn.Name.Name] = facts
		}
	}
	return out
}
