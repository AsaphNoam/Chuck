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

const (
	maxManifestText  = 128
	maxManifestBytes = 16 << 10
)

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

// ManifestProvider is the provider selection, the adapter's own initialize
// self-description (observed, unverified) and the SHA-256 of the managed
// adapter entrypoint actually run (empty: a PATH adapter, unknown).
type ManifestProvider struct {
	Source         string `json:"source,omitempty"`
	AdapterName    string `json:"adapter_name,omitempty"`
	AdapterVersion string `json:"adapter_version,omitempty"`
	ArtifactSHA256 string `json:"artifact_sha256,omitempty"`
}

// ManifestLayer identifies one requested instruction layer.
type ManifestLayer struct {
	Kind   string `json:"kind"`
	SHA256 string `json:"sha256"`
	Bytes  int    `json:"bytes"`
}

// adapterIdentity is what an adapter's initialize response says it is, plus
// the managed artifact digest Chuck computed for it.
type adapterIdentity struct{ Name, Version, Artifact string }

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
		Version: ExecutionManifestVersion, Purpose: spec.Purpose(), SkipPerms: spec.SkipPerms,
		Layers: []ManifestLayer{}, Delivery: "requested", Adoption: "unknown",
	}
	clipped := func(s string) string {
		if len(s) > maxManifestText {
			m.Truncated = true
			return s[:maxManifestText]
		}
		return s
	}
	m.Backend, m.Model, m.Effort = clipped(spec.Agent.Backend), clipped(spec.Agent.Model), clipped(spec.Effort)
	m.Knowledge = clipped(spec.KnowledgeDigest)
	if c := spec.ExecutionContract; c != nil {
		choice := state.ModelChoice{Kind: clipped(c.ModelChoice.Kind), ID: clipped(c.ModelChoice.ID)}
		setup := state.SetupChoice{Source: clipped(c.Setup.Source), Provider: clipped(c.Setup.Provider),
			Profile: clipped(c.Setup.Profile), Mode: clipped(c.Setup.Mode)}
		m.Choice, m.Setup = &choice, &setup
	}
	m.Provider = ManifestProvider{Source: clipped(spec.ProviderSource), AdapterName: clipped(adapter.Name),
		AdapterVersion: clipped(adapter.Version), ArtifactSHA256: clipped(adapter.Artifact)}
	// Native Codex instructions are merged ahead of the Chuck layers.
	if n := spec.nativeInstructions; n != nil {
		m.Layers = append(m.Layers, *n)
	}
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
	// Every field above is clipped, so this backstop only guards future growth.
	if raw, err := json.Marshal(m); err != nil || len(raw) > maxManifestBytes {
		m.Layers, m.Truncated = []ManifestLayer{}, true
	}
	return m
}

func hashText(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
