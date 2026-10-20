package state

import (
	"encoding/json"
	"fmt"
)

// ExecutionContractVersion is the only execution-contract shape this build
// writes and reads (TS-02.R53).
const ExecutionContractVersion = 1

// Model-choice kinds: native leaves the model to the provider's own setup;
// explicit sends ID verbatim.
const (
	ModelChoiceNative   = "native"
	ModelChoiceExplicit = "explicit"
)

// ExecutionContract is a session's frozen effective native selection
// (TS-02.R53, TS-01.R40). Ordinary continuation copies it; only an explicit
// model switch or configuration refresh derives a new one. It carries no
// credential, raw native configuration or environment value.
type ExecutionContract struct {
	Version     int         `json:"version"`
	Backend     string      `json:"backend"`
	Model       string      `json:"model"`
	BackendType string      `json:"backend_type"`
	ModelChoice ModelChoice `json:"model_choice"`
	Effort      string      `json:"effort,omitempty"`
	Fast        bool        `json:"fast,omitempty"`
	SkipPerms   bool        `json:"skip_permissions,omitempty"`
	Setup       SetupChoice `json:"setup"`
	// AuthOrigin names the personal authentication selector; never a secret.
	AuthOrigin string `json:"auth_origin"`
	// NativeHome names who owns the native session storage.
	NativeHome string `json:"native_home"`
}

// ModelChoice is the provider model decision, separate from the catalogue key.
type ModelChoice struct {
	Kind string `json:"kind"`
	ID   string `json:"id,omitempty"`
}

// SetupChoice is the supported native setup selector and its provenance.
// Source is "native" for the personal native setup, "bound" for a source
// binding resolved through that same supported path.
type SetupChoice struct {
	Source   string `json:"source"`
	Provider string `json:"provider,omitempty"`
	Profile  string `json:"profile,omitempty"`
	Mode     string `json:"mode,omitempty"`
}

// ProviderModelID is the model a provider start sends; empty means native.
func (c ExecutionContract) ProviderModelID() string {
	if c.ModelChoice.Kind == ModelChoiceExplicit {
		return c.ModelChoice.ID
	}
	return ""
}

func (c ExecutionContract) validate() error {
	if c.Version != ExecutionContractVersion {
		return fmt.Errorf("unsupported execution contract version %d", c.Version)
	}
	switch c.ModelChoice.Kind {
	case ModelChoiceNative:
	case ModelChoiceExplicit:
		if c.ModelChoice.ID == "" {
			return fmt.Errorf("explicit model choice has no id")
		}
	default:
		return fmt.Errorf("unknown model choice %q", c.ModelChoice.Kind)
	}
	return nil
}

// EncodeExecutionContract returns the stored column value; nil stores NULL.
func EncodeExecutionContract(c *ExecutionContract) (any, error) {
	if c == nil {
		return nil, nil
	}
	if err := c.validate(); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	return string(raw), nil
}

// decodeExecutionContract reads a stored column. NULL/empty is a legacy row
// (nil); an unreadable or unknown-version value is also nil, so continuation
// treats it as unknown rather than guessing.
func decodeExecutionContract(raw *string) *ExecutionContract {
	if raw == nil || *raw == "" {
		return nil
	}
	var c ExecutionContract
	if json.Unmarshal([]byte(*raw), &c) != nil || c.validate() != nil {
		return nil
	}
	return &c
}
