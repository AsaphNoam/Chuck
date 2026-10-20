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

	"github.com/AsaphNoam/Chuck/internal/backend"
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

// TS-11.R24: every retained field is clipped and the serialized record stays
// within 16 KiB, signalling truncation instead of failing the start.
func TestExecutionManifestBoundsEveryField(t *testing.T) {
	long := strings.Repeat("m", 20<<10)
	spec := LaunchSpec{Agent: state.Agent{Backend: long, Model: long}, Effort: long, KnowledgeDigest: long,
		ExecutionContract: &state.ExecutionContract{ModelChoice: state.ModelChoice{Kind: state.ModelChoiceExplicit, ID: long},
			Setup: state.SetupChoice{Source: long, Provider: long, Profile: long, Mode: long}}}
	m := executionManifest(spec, adapterIdentity{Artifact: long})
	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) > maxManifestBytes || !m.Truncated || len(m.Choice.ID) != maxManifestText {
		t.Fatalf("manifest bytes=%d truncated=%v id=%d", len(raw), m.Truncated, len(m.Choice.ID))
	}
}

// FS-18.R21 / TS-11.R24: the native Codex instructions actually merged at
// process start, and the managed adapter artifact, are identified by hash.
func TestExecutionManifestIdentifiesNativeInstructionsAndArtifact(t *testing.T) {
	ad, ok := backend.For("codex-acp")
	if !ok {
		t.Fatal("codex adapter missing")
	}
	c := NewChatRuntime(nil)
	c.command = "/usr/bin/true"
	var records []*ExecutionManifest
	for _, native := range []string{"native-one", "native-two"} {
		spec := LaunchSpec{Cwd: t.TempDir(), BackendType: "codex-acp", SystemPrompt: "same Chuck persona",
			Env: []string{`CODEX_CONFIG={"developer_instructions":"` + native + `"}`}}
		cmd, err := c.spawnCmd(ad, spec)
		if err != nil {
			t.Fatal(err)
		}
		if err := c.startCmd(cmd, &spec); err != nil {
			t.Fatal(err)
		}
		_ = cmd.Wait()
		m := executionManifest(spec, adapterIdentity{Artifact: "artifact-digest"})
		if len(m.Layers) == 0 || m.Layers[0] != (ManifestLayer{Kind: "native_instructions", SHA256: sha(native), Bytes: len(native)}) {
			t.Fatalf("layers = %+v", m.Layers)
		}
		if m.Provider.ArtifactSHA256 != "artifact-digest" {
			t.Fatalf("provider = %+v", m.Provider)
		}
		records = append(records, m)
	}
	if records[0].Layers[0] == records[1].Layers[0] {
		t.Fatal("different native instruction requests yield identical layers")
	}

	root := t.TempDir()
	_, args := backend.LaunchCommand(ad, root)
	if err := os.MkdirAll(filepath.Dir(args[0]), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(args[0], []byte("entrypoint"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := backend.ManagedArtifactDigest(ad, root); got != sha("entrypoint") {
		t.Fatalf("artifact digest = %q", got)
	}
	if got := backend.ManagedArtifactDigest(ad, ""); got != "" {
		t.Fatalf("PATH adapter digest = %q, want unknown", got)
	}
}

// FS-18.R21: terminal process starts record a manifest through the shared
// metadata builder, with adapter identity unknown.
func TestTerminalSessionMetaCarriesManifest(t *testing.T) {
	spec := LaunchSpec{Agent: state.Agent{Interface: "terminal"}, BackendType: "claude-acp", SystemPrompt: "terminal persona"}
	meta := NewSessionMeta(spec, "native-1")
	if meta.ExecutionManifest == nil || len(meta.ExecutionManifest.Layers) != 1 || meta.ExecutionManifest.Provider.AdapterName != "" {
		t.Fatalf("terminal manifest = %+v", meta.ExecutionManifest)
	}
}
