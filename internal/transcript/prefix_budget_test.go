package transcript

import (
	"errors"
	"os"
	"strings"
	"testing"
)

type prefixReadBudget struct{ remaining int }

var errPrefixReadBudget = errors.New("prefix read budget")

func (b *prefixReadBudget) Write(p []byte) (int, error) {
	if len(p) > b.remaining {
		return 0, errPrefixReadBudget
	}
	b.remaining -= len(p)
	return len(p), nil
}

// A crash-torn tail has no complete-record size guarantee. Prefix capture
// must charge the backward scan before the ordinary canonical replay starts.
func TestCompletePrefixChargesTornTailToBudget(t *testing.T) {
	home := t.TempDir()
	w, err := Open(home, "a_budget", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = w.Close(); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(transcriptPath(home, "a_budget"), []byte("{}\n"+strings.Repeat("x", 128<<10)), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = CompletePrefixWithBudget(home, "a_budget", &prefixReadBudget{remaining: 64 << 10}); !errors.Is(err, errPrefixReadBudget) {
		t.Fatalf("tail bypassed budget: %v", err)
	}
	if n, err := CompletePrefix(home, "a_budget"); err != nil || n != 3 {
		t.Fatalf("ordinary prefix=%d %v", n, err)
	}
}
