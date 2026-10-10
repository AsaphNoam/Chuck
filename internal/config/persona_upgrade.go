package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

// defaultPersonaUpgradeMarker is deliberately private: the completion file is
// installation bookkeeping and is not part of the public configuration shape.
type defaultPersonaUpgradeMarker struct {
	Version int `json:"version"`
}

var writeDefaultPersonaUpgradeMarker = func(path string) error {
	return writeJSONAtomic(path, defaultPersonaUpgradeMarker{Version: 1})
}

// PrepareDefaultPersonaUpgrade applies the one-time legacy Implementer default
// migration. The caller must have seeded absent files first and must pass a
// valid, readable config. No user-owned role or preference other than the
// legacy default_role is changed.
func (s *Store) PrepareDefaultPersonaUpgrade(cfg Config) (Config, error) {
	complete, err := s.defaultPersonaUpgradeComplete()
	if err != nil {
		return cfg, err
	}
	if complete {
		return cfg, nil
	}

	if cfg.DefaultRole == "implementer" {
		role, err := s.ReadRole("default")
		if err != nil {
			return cfg, fmt.Errorf("config: read default persona for upgrade: %w", err)
		}
		if role.SystemPrompt != "" || role.SkipPermissions != nil || !role.IsChuckAware() {
			return cfg, fmt.Errorf("config: cannot upgrade default persona: role %q is customized; restore an empty prompt, inherited permissions, and Chuck awareness, or rename it before retrying", s.rolePath("default"))
		}
		cfg.DefaultRole = "default"
		if err := s.WriteConfig(cfg); err != nil {
			return cfg, fmt.Errorf("config: write upgraded default role: %w", err)
		}
	}
	if err := writeDefaultPersonaUpgradeMarker(s.defaultPersonaUpgradePath()); err != nil {
		return cfg, fmt.Errorf("config: write default persona upgrade marker: %w", err)
	}
	return cfg, nil
}

func (s *Store) defaultPersonaUpgradeComplete() (bool, error) {
	data, err := os.ReadFile(s.defaultPersonaUpgradePath())
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("config: read default persona upgrade marker: %w", err)
	}
	var marker defaultPersonaUpgradeMarker
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&marker); err != nil || marker.Version != 1 {
		if err == nil {
			err = fmt.Errorf("unsupported version %d", marker.Version)
		}
		return false, fmt.Errorf("config: invalid default persona upgrade marker: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("trailing data")
		}
		return false, fmt.Errorf("config: invalid default persona upgrade marker: %w", err)
	}
	return true, nil
}
