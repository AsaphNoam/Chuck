package attachment

import (
	"strings"
	"testing"
)

func TestStartupPrunesOnlyProvisionalUnreferencedCopies(t *testing.T) {
	s := New(t.TempDir())
	provisional, err := s.Put("a_one", "unaccepted.txt", strings.NewReader("unaccepted"))
	if err != nil {
		t.Fatal(err)
	}
	accepted, err := s.Put("a_one", "accepted.txt", strings.NewReader("accepted"))
	if err != nil {
		t.Fatal(err)
	}
	durable, err := s.Put("a_one", "durable.txt", strings.NewReader("durable"))
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Retain("a_one", []string{accepted.ID}); err != nil {
		t.Fatal(err)
	}
	if err = s.PruneUnowned("a_one", map[string]bool{durable.ID: true}); err != nil {
		t.Fatal(err)
	}
	if f, _, err := s.Open("a_one", provisional.ID); err == nil {
		f.Close()
		t.Fatal("unaccepted copy survived startup")
	}
	for _, id := range []string{accepted.ID, durable.ID} {
		f, _, err := s.Open("a_one", id)
		if err != nil {
			t.Fatal(err)
		}
		f.Close()
	}
	// A retained copy remains safe even when an older corrupt transcript cannot
	// enumerate its references on a future run.
	if err = s.PruneUnowned("a_one", map[string]bool{}); err != nil {
		t.Fatal(err)
	}
	f, _, err := s.Open("a_one", durable.ID)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
}
