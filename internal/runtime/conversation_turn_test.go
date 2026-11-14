package runtime

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/AsaphNoam/Chuck/internal/state"
)

func TestConversationPromptCallSites(t *testing.T) {
	got := make(map[string]int)
	paths, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
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
			if !ok {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok || len(call.Args) < 2 {
					return true
				}
				method, ok := call.Fun.(*ast.SelectorExpr)
				arg, literal := call.Args[1].(*ast.BasicLit)
				if ok && method.Sel.Name == "Call" && literal && arg.Value == `"session/prompt"` {
					got[path+":"+fn.Name.Name]++
				}
				return true
			})
		}
	}
	want := map[string]int{"chat.go:beginTurn": 1, "chat.go:nameCodexSession": 1, "quota_check.go:driveQuotaCheck": 2}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("provider prompt call sites = %v, want %v", got, want)
	}
}

func TestConversationTurnEndsWithProviderOutcomeOrStop(t *testing.T) {
	for _, stop := range []bool{false, true} {
		name := "provider"
		if stop {
			name = "stop"
		}
		t.Run(name, func(t *testing.T) {
			c, spec := newChatTest(t, "hold_turn")
			hold := filepath.Join(t.TempDir(), "hold")
			spec.Env = append(spec.Env, "FAKEACP_HOLD_FILE="+hold)
			spec.Generation = "g"
			history := recordHistory(c)
			ctx := context.Background()
			h, err := c.Start(ctx, spec)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if _, err := c.lookup(h.AgentID); err == nil {
					_ = c.Stop(ctx, h.AgentID)
				}
			})
			ch, unsubscribe, err := c.Subscribe(h.AgentID)
			if err != nil {
				t.Fatal(err)
			}
			defer unsubscribe()
			if err := c.SendPrompt(ctx, h.AgentID, "work"); err != nil {
				t.Fatal(err)
			}
			waitForChunk(t, ch, "working")
			running, err := c.store.ReadConversationTurn(h.AgentID, "g")
			if err != nil || running.State != "running" || running.Kind != state.TurnOriginPerson || running.Outcome != "" {
				t.Fatalf("working turn = %+v, err %v", running, err)
			}
			as, err := c.lookup(h.AgentID)
			if err != nil || !as.canStartTurn(running.TurnID) {
				t.Fatalf("working turn missing from runtime: %v", err)
			}
			want := "completed"
			if stop {
				if err := c.Stop(ctx, h.AgentID); err != nil {
					t.Fatal(err)
				}
				want = "process_ended"
				if slices.ContainsFunc(history(), func(ev Event) bool { return ev.Type == EvTurnEnd }) {
					t.Fatal("Stop produced a turn_end")
				}
			} else {
				if err := os.WriteFile(hold, []byte("go"), 0o600); err != nil {
					t.Fatal(err)
				}
				ev := waitForEvent(t, ch, EvTurnEnd)
				outcome, _, err := DecodeTurnEnd(ev)
				if err != nil || outcome != TurnCompleted || ev.TurnID != running.TurnID {
					t.Fatalf("turn_end = %+v, outcome %v, err %v", ev, outcome, err)
				}
				waitTurnSettled(t, c, h.AgentID)
			}
			row, err := c.store.ReadConversationTurn(h.AgentID, "g")
			if err != nil || row.TurnID != running.TurnID || row.State != "ended" || row.Outcome != want || row.EndedAt == "" {
				t.Fatalf("ended turn = %+v, err %v", row, err)
			}
		})
	}
}
