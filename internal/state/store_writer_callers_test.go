package state

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// uncalledWriterAllowlist names exported *Store writers whose only production
// callers are reached through an interface or function value the selector scan
// cannot see. Each entry needs a justification. Empty is the goal (TS-01.R50).
var uncalledWriterAllowlist = map[string]string{}

// fixtureOnlyWriters is the open remainder of EXEC-6(c): writers that only
// other packages' tests call, so they cannot move into a state _test.go file
// without retargeting those tests. The list may only shrink: a new uncalled
// writer fails, and so does an entry that gained a production caller.
var fixtureOnlyWriters = map[string]bool{
	"AddThinkTankRecord": true, "AttachTaskContext": true, "BindAssignmentInput": true,
	"BindPipelineStageTaskStandingAgent": true, "ConsumeTurnBudget": true,
	"FinalizeThinkTankAttempt": true, "InsertPipelineAttempt": true, "InterruptTask": true,
	"MarkQuotaProviderStarted": true, "ObserveQuota": true, "StartQuotaCheck": true,
}

var sqlWriteStatement = regexp.MustCompile(`(?i)\b(INSERT\s+(OR\s+\w+\s+)?INTO|UPDATE\s+(OR\s+\w+\s+)?\w+\s+SET|DELETE\s+FROM|REPLACE\s+INTO)\b`)

// TestExportedStoreWritersHaveProductionCallers enforces TS-01.R50: an
// exported *Store writer with no production caller is deleted.
//
// Classification: a method is a writer when its body, or the body of any
// function/method it calls by name inside internal/state (transitively, name
// based), contains a string literal (or a package-level string const/var
// referenced by identifier) holding an INSERT INTO / UPDATE .. SET /
// DELETE FROM / REPLACE INTO statement. Readers are never named; the SQL text
// decides. Name-based resolution may over-approximate writers (a reader that
// shares a callee name with a writer) but only ever errs toward checking more.
//
// A caller is any `.Name(` selector call or method-value reference in a
// non-test .go file under cmd/ or internal/ (including internal/state
// itself). Test-only use does not count. Same-named methods on other types
// can mask an uncalled writer; that is accepted to keep the scan simple.
func TestExportedStoreWritersHaveProductionCallers(t *testing.T) {
	fset := token.NewFileSet()
	files := parseNonTestFiles(t, fset, ".")

	pkgStrings := map[string]string{}
	for _, f := range files {
		for _, d := range f.Decls {
			gd, ok := d.(*ast.GenDecl)
			if !ok || (gd.Tok != token.CONST && gd.Tok != token.VAR) {
				continue
			}
			for _, spec := range gd.Specs {
				vs := spec.(*ast.ValueSpec)
				for i, name := range vs.Names {
					if i < len(vs.Values) {
						pkgStrings[name.Name] = literalText(vs.Values[i])
					}
				}
			}
		}
	}

	type fn struct {
		writes bool
		calls  map[string]bool
	}
	funcs := map[string]*fn{}
	exportedStore := map[string]bool{}
	for _, f := range files {
		for _, d := range f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Body == nil {
				continue
			}
			info := funcs[fd.Name.Name]
			if info == nil {
				info = &fn{calls: map[string]bool{}}
				funcs[fd.Name.Name] = info
			}
			if fd.Recv != nil && fd.Name.IsExported() && isStoreReceiver(fd.Recv.List[0].Type) {
				exportedStore[fd.Name.Name] = true
			}
			ast.Inspect(fd.Body, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.BasicLit:
					if x.Kind == token.STRING {
						if s, err := strconv.Unquote(x.Value); err == nil && sqlWriteStatement.MatchString(s) {
							info.writes = true
						}
					}
				case *ast.Ident:
					if s, ok := pkgStrings[x.Name]; ok && sqlWriteStatement.MatchString(s) {
						info.writes = true
					}
					info.calls[x.Name] = true
				case *ast.SelectorExpr:
					info.calls[x.Sel.Name] = true
				}
				return true
			})
		}
	}

	memo := map[string]bool{}
	var isWriter func(name string, seen map[string]bool) bool
	isWriter = func(name string, seen map[string]bool) bool {
		if v, ok := memo[name]; ok {
			return v
		}
		info := funcs[name]
		if info == nil || seen[name] {
			return false
		}
		seen[name] = true
		res := info.writes
		for c := range info.calls {
			if res {
				break
			}
			if c != name && isWriter(c, seen) {
				res = true
			}
		}
		delete(seen, name)
		memo[name] = res
		return res
	}

	writers := map[string]bool{}
	for name := range exportedStore {
		if isWriter(name, map[string]bool{}) {
			writers[name] = true
		}
	}
	if len(writers) == 0 {
		t.Fatal("classified no exported Store writers; the scan is broken")
	}

	referenced := map[string]bool{}
	root := filepath.Join("..", "..")
	for _, dir := range []string{"cmd", "internal"} {
		for _, f := range parseNonTestFiles(t, fset, filepath.Join(root, dir)) {
			ast.Inspect(f, func(n ast.Node) bool {
				if sel, ok := n.(*ast.SelectorExpr); ok && writers[sel.Sel.Name] {
					referenced[sel.Sel.Name] = true
				}
				return true
			})
		}
	}

	var uncalled []string
	for name := range writers {
		if referenced[name] {
			if fixtureOnlyWriters[name] {
				t.Errorf("%s now has a production caller; remove it from fixtureOnlyWriters", name)
			}
			continue
		}
		if _, ok := uncalledWriterAllowlist[name]; ok || fixtureOnlyWriters[name] {
			continue
		}
		uncalled = append(uncalled, name)
	}
	sort.Strings(uncalled)
	if len(uncalled) > 0 {
		t.Fatalf("exported Store writers with no production caller (delete, or move to a test helper): %s", strings.Join(uncalled, ", "))
	}
}

func parseNonTestFiles(t *testing.T, fset *token.FileSet, dir string) []*ast.File {
	t.Helper()
	var out []*ast.File
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if dir == "." && path != "." {
				return fs.SkipDir
			}
			switch d.Name() {
			case "node_modules", "dist", ".gocache", "testdata":
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		f, err := parser.ParseFile(fset, path, src, 0)
		if err != nil {
			return err
		}
		out = append(out, f)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func isStoreReceiver(e ast.Expr) bool {
	if star, ok := e.(*ast.StarExpr); ok {
		e = star.X
	}
	id, ok := e.(*ast.Ident)
	return ok && id.Name == "Store"
}

func literalText(e ast.Expr) string {
	var sb strings.Builder
	ast.Inspect(e, func(n ast.Node) bool {
		if bl, ok := n.(*ast.BasicLit); ok && bl.Kind == token.STRING {
			if s, err := strconv.Unquote(bl.Value); err == nil {
				sb.WriteString(s)
			}
		}
		return true
	})
	return sb.String()
}
