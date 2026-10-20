package runtime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AsaphNoam/Chuck/internal/state"
)

func sha(s string) string { sum := sha256.Sum256([]byte(s)); return hex.EncodeToString(sum[:]) }

// FS-18.A17 / TS-11.R24: the manifest names each requested layer by kind,
// hash and size, separates runtime additions from the frozen persona, omits a
// Codex fork's layers, keeps unknown fields unknown and carries no raw text.
func TestExecutionManifestIdentifiesRequestedLayers(t *testing.T) {
	spec := LaunchSpec{
		Agent:       state.Agent{Backend: "claude", Model: "sonnet"},
		BackendType: "claude-acp", Effort: "high", ProviderSource: "bundled",
		SystemPrompt: "PERSONA_SENTINEL", HistoryPrimer: "BRIDGE_SENTINEL", RuntimeSystemPromptSuffix: "RUNTIME_SENTINEL",
		KnowledgeDigest:  "pkgdigest",
		AutoApproveTools: map[string]struct{}{"mcp__chuck__b": {}, "mcp__chuck__a": {}},
		ExecutionContract: &state.ExecutionContract{Version: 1, ModelChoice: state.ModelChoice{Kind: "native"},
			Setup: state.SetupChoice{Source: "native"}},
	}
	m := executionManifest(spec, adapterIdentity{Name: "claude-agent-acp", Version: "0.85.1"})
	want := []ManifestLayer{
		{Kind: "persona", SHA256: sha("PERSONA_SENTINEL"), Bytes: 16},
		{Kind: "bridge", SHA256: sha("BRIDGE_SENTINEL"), Bytes: 15},
		{Kind: "runtime_guidance", SHA256: sha("RUNTIME_SENTINEL"), Bytes: 16},
	}
	if len(m.Layers) != len(want) {
		t.Fatalf("layers = %+v", m.Layers)
	}
	for i := range want {
		if m.Layers[i] != want[i] {
			t.Fatalf("layer %d = %+v, want %+v", i, m.Layers[i], want[i])
		}
	}
	if m.Purpose != "fresh" || m.Choice.Kind != "native" || m.Delivery != "requested" || m.Adoption != "unknown" {
		t.Fatalf("manifest = %+v", m)
	}
	if m.Tools != sha(`["mcp__chuck__a","mcp__chuck__b"]`) || m.Knowledge != "pkgdigest" {
		t.Fatalf("digests = %q %q", m.Tools, m.Knowledge)
	}
	raw, _ := json.Marshal(m)
	for _, secret := range []string{"PERSONA_SENTINEL", "BRIDGE_SENTINEL", "RUNTIME_SENTINEL"} {
		if strings.Contains(string(raw), secret) {
			t.Fatalf("manifest carries raw text %q: %s", secret, raw)
		}
	}
	if len(raw) > 16<<10 {
		t.Fatalf("manifest is %d bytes", len(raw))
	}

	fork := spec
	fork.BackendType, fork.Fork = "codex-acp", &ForkPlan{SourceSessionID: "src"}
	if m := executionManifest(fork, adapterIdentity{}); m.Purpose != "fork" || len(m.Layers) != 0 {
		t.Fatalf("codex fork manifest = %+v", m)
	}

	legacy := spec
	legacy.ExecutionContract, legacy.LastSessionID = nil, "native-1"
	m = executionManifest(legacy, adapterIdentity{Name: strings.Repeat("x", 500)})
	if m.Choice != nil || m.Setup != nil || m.Purpose != "resume" || len(m.Provider.AdapterName) != maxManifestText || !m.Truncated {
		t.Fatalf("legacy manifest = %+v", m)
	}
}

// A real start writes the manifest into the process's session_meta record,
// with the adapter's observed initialize identity, while the frozen persona
// field is unchanged.
func TestStartRecordsExecutionManifest(t *testing.T) {
	c, spec := newChatTest(t, "stream_text")
	home := t.TempDir()
	f, err := os.Create(filepath.Join(home, "t.ndjson"))
	if err != nil {
		t.Fatal(err)
	}
	var meta *SessionMetaData
	c.SetPersistence(home, func(_ string, _ string, m *SessionMetaData) (TranscriptWriter, error) {
		meta = m
		return &messageTestWriter{file: f}, nil
	}, &removalIndexer{})
	spec.SystemPrompt = "PERSONA_SENTINEL"
	h, err := c.Start(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Stop(context.Background(), h.AgentID) })
	if meta == nil || meta.ExecutionManifest == nil {
		t.Fatal("session_meta has no execution manifest")
	}
	if meta.SystemPrompt != "PERSONA_SENTINEL" || meta.ExecutionManifest.Layers[0].SHA256 != sha("PERSONA_SENTINEL") {
		t.Fatalf("meta persona %q manifest %+v", meta.SystemPrompt, meta.ExecutionManifest)
	}
}
