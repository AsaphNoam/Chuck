package runtime

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"

	"github.com/AsaphNoam/Chuck/internal/state"
)

// ExecutionManifestVersion is the only manifest shape this build writes.
const ExecutionManifestVersion = 1

const maxManifestText = 128

// ExecutionManifest records what one process start requested (TS-11.R24,
// FS-18.R21): its purpose, frozen selection, the hash and size of each Chuck
// instruction layer sent, and the operating-package and tool-inventory
// digests offered. It holds no raw prompt, configuration, environment or
// credential, and never claims the provider adopted or read anything.
type ExecutionManifest struct {
	Version   int                `json:"version"`
	Purpose   string             `json:"purpose"`
	Backend   string             `json:"backend"`
	Model     string             `json:"model"`
	Choice    *state.ModelChoice `json:"model_choice,omitempty"` // nil: legacy/unknown
	Effort    string             `json:"effort,omitempty"`
	SkipPerms bool               `json:"skip_permissions,omitempty"`
	Setup     *state.SetupChoice `json:"setup,omitempty"`
	Provider  ManifestProvider   `json:"provider"`
	Layers    []ManifestLayer    `json:"layers"`
	Knowledge string             `json:"knowledge_digest,omitempty"`
	Tools     string             `json:"tool_inventory_digest,omitempty"`
	// Delivery is always "requested": the assembled request, not adoption.
	Delivery  string `json:"delivery"`
	Adoption  string `json:"adoption"`
	Truncated bool   `json:"truncated,omitempty"`
}

// ManifestProvider is the provider selection and the adapter's own
// initialize self-description (observed, unverified).
type ManifestProvider struct {
	Source         string `json:"source,omitempty"`
	AdapterName    string `json:"adapter_name,omitempty"`
	AdapterVersion string `json:"adapter_version,omitempty"`
}

// ManifestLayer identifies one requested instruction layer.
type ManifestLayer struct {
	Kind   string `json:"kind"`
	SHA256 string `json:"sha256"`
	Bytes  int    `json:"bytes"`
}

// adapterIdentity is what an adapter's initialize response says it is.
type adapterIdentity struct{ Name, Version string }

func decodeAdapterIdentity(initRes json.RawMessage) adapterIdentity {
	var body struct {
		AgentInfo struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"agentInfo"`
	}
	_ = json.Unmarshal(initRes, &body)
	return adapterIdentity{Name: body.AgentInfo.Name, Version: body.AgentInfo.Version}
}

func executionManifest(spec LaunchSpec, adapter adapterIdentity) *ExecutionManifest {
	m := &ExecutionManifest{
		Version: ExecutionManifestVersion, Purpose: spec.Purpose(),
		Backend: spec.Agent.Backend, Model: spec.Agent.Model, Effort: spec.Effort, SkipPerms: spec.SkipPerms,
		Layers: []ManifestLayer{}, Knowledge: spec.KnowledgeDigest, Delivery: "requested", Adoption: "unknown",
	}
	if c := spec.ExecutionContract; c != nil {
		choice, setup := c.ModelChoice, c.Setup
		m.Choice, m.Setup = &choice, &setup
	}
	clipped := func(s string) string {
		if len(s) > maxManifestText {
			m.Truncated = true
			return s[:maxManifestText]
		}
		return s
	}
	m.Provider = ManifestProvider{Source: clipped(spec.ProviderSource), AdapterName: clipped(adapter.Name), AdapterVersion: clipped(adapter.Version)}
	// A Codex fork sends no Chuck layer: the forked thread keeps its own.
	if !(spec.BackendType == "codex-acp" && spec.Purpose() == PurposeFork) {
		persona, kind := spec.SystemPrompt, "persona"
		if spec.RuntimeSystemPrompt != "" {
			persona, kind = spec.RuntimeSystemPrompt, "persona_override"
		}
		for _, layer := range []struct{ kind, text string }{
			{kind, persona}, {"bridge", spec.HistoryPrimer}, {"runtime_guidance", spec.RuntimeSystemPromptSuffix},
		} {
			if layer.text != "" {
				m.Layers = append(m.Layers, ManifestLayer{Kind: layer.kind, SHA256: hashText(layer.text), Bytes: len(layer.text)})
			}
		}
	}
	if len(spec.AutoApproveTools) > 0 {
		names := make([]string, 0, len(spec.AutoApproveTools))
		for name := range spec.AutoApproveTools {
			names = append(names, name)
		}
		sort.Strings(names)
		raw, _ := json.Marshal(names)
		m.Tools = hashText(string(raw))
	}
	return m
}

func hashText(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
