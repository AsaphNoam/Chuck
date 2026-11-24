package config

import "testing"

func TestFirstChatMarkerRequiresAbsentConfigAndCatalog(t *testing.T) {
	for _, existing := range []string{"fresh", "config", "catalog"} {
		t.Run(existing, func(t *testing.T) {
			store := newTestStore(t)
			if existing == "config" {
				if err := store.WriteConfig(DefaultConfig()); err != nil {
					t.Fatal(err)
				}
			}
			if existing == "catalog" {
				if err := store.WriteBackends(DefaultBackends()); err != nil {
					t.Fatal(err)
				}
			}
			if err := store.SeedIfAbsent(); err != nil {
				t.Fatal(err)
			}
			if err := store.SeedIfAbsent(); err != nil {
				t.Fatal(err)
			}
			config, err := store.ReadConfig()
			if err != nil {
				t.Fatal(err)
			}
			if config.FirstChatEligible != (existing == "fresh") {
				t.Fatalf("eligibility = %v for %s", config.FirstChatEligible, existing)
			}
		})
	}
}
