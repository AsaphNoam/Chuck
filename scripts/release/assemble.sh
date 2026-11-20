#!/usr/bin/env bash
# Assemble the self-contained darwin/arm64 release runtime. This is a release
# builder tool, not the end-user installer: it requires Go, npm, and a verified
# Node distribution archive supplied by CI or the release operator.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

VERSION="${VERSION:-${1:-}}"
NODE_TARBALL="${NODE_TARBALL:-}"
NODE_VERSION="${NODE_VERSION:-22.22.0}"
# Published by nodejs.org for node-v22.22.0-darwin-arm64.tar.gz. This pin keeps
# the private runtime input reproducible; changing it is a deliberate release
# dependency update and requires refreshing the matching Node archive.
NODE_SHA256="5ed4db0fcf1eaf84d91ad12462631d73bf4576c1377e192d222e48026a902640"
CLAUDE_ACP_VERSION="0.85.1"
CLAUDE_ACP_COMPONENT_VERSION="${CLAUDE_ACP_VERSION}+chuck.3"
CLAUDE_ACP_SOURCE_SHA256="644daa80157fcdfe7b806cbd23b56f75be4fed146a890aa5e219204e689aca13"
# Each adapter patch is applied in order with a pinned output; *_PATCHED_SHA256
# is the final artifact after every patch.
CLAUDE_ACP_BACKGROUND_QUOTA_SHA256="1a46fee6cf61a7e5c9a119173812dec95a8d60b6ad162d01f25ba07cb56c4867"
CLAUDE_ACP_QUOTA_READ_SHA256="a6c257eb23c2540baf603e50c10816cea8c29f06e647335f6eab2951d5a846d1"
CLAUDE_ACP_PATCHED_SHA256="690519f9d282ce2dcd0ff6b833805dafe3f10d1eb27fbf1fb57883501b35bfdb"
CLAUDE_ACP_FORK_SOURCE_SHA256="70da6567b5d9f76dedef4aa02dc1c53e4bdb5e7f4bc28e374d1ad680e6287c2a"
CLAUDE_ACP_FORK_PATCHED_SHA256="2812c9c07a73bff04f7ba1768ec0257870060c4f6f7b000b60d1ed51402ba8ab"
CODEX_ACP_VERSION="2.1.1"
CODEX_ACP_COMPONENT_VERSION="${CODEX_ACP_VERSION}+chuck.3"
CODEX_ACP_SOURCE_SHA256="4b76310393d756a0f111687cd9df899720f36f7c59b1eb1d86429484034ff91b"
CODEX_ACP_STEERING_SHA256="8424ea6e64c8b7e20f1ab325eaf8f8a033c7f24a161c663d7c42695a9c5363d9"
CODEX_ACP_QUOTA_READ_SHA256="69b0afc9825cabfa80aa1c6b8f04ea75ce752d8448191e48212690c031718c73"
CODEX_ACP_PATCHED_SHA256="e9186503c0bbbaff2edb63c0e2a4a0816a8c5727025eb59401853c1d69447891"
# The Codex CLI is a direct runtime dependency, not just codex-acp's transitive
# one: its platform binary is the single managed Codex a backend can explicitly
# select as the Chuck bundle (TS-06.R30). Keep in step with
# scripts/release/package.json.
CODEX_CLI_VERSION="0.159.3"
# Bundled native providers inside the managed runtime root; the same paths the
# resolver selects and release layout verification requires.
CLAUDE_NATIVE="node_modules/@anthropic-ai/claude-agent-sdk-darwin-arm64/claude"
CODEX_NATIVE="node_modules/@openai/codex-darwin-arm64/vendor/aarch64-apple-darwin/bin/codex"
TARGET="darwin-arm64"
OUT_DIR="${OUT_DIR:-$ROOT/dist/release}"

die() { echo "error: $*" >&2; exit 1; }
[ -n "$VERSION" ] || die "set VERSION or pass the release version as the first argument"
[ -n "$NODE_TARBALL" ] || die "set NODE_TARBALL to a verified Node ${NODE_VERSION} darwin-arm64 tarball"
[ -f "$NODE_TARBALL" ] || die "NODE_TARBALL does not exist: $NODE_TARBALL"
command -v go >/dev/null || die "Go is required to assemble a release"
command -v npm >/dev/null || die "npm is required to build the embedded UI"

work="$(mktemp -d "${TMPDIR:-/tmp}/chuck-release.XXXXXX")"
cleanup() { rm -rf "$work"; }
trap cleanup EXIT

echo "==> Building embedded Chuck ${VERSION}"
make dist VERSION="$VERSION"

stage="$work/chuck-${VERSION}-${TARGET}"
mkdir -p "$stage/libexec" "$stage/runtime"
cp bin/chuck "$stage/libexec/chuck"

echo "==> Installing private Node ${NODE_VERSION}"
node_sum="$(shasum -a 256 "$NODE_TARBALL" | awk '{print $1}')"
[ "$node_sum" = "$NODE_SHA256" ] || die "NODE_TARBALL checksum does not match the pinned Node ${NODE_VERSION} distribution"
tar -xzf "$NODE_TARBALL" -C "$work"
node_source="$work/node-v${NODE_VERSION}-${TARGET}"
[ -x "$node_source/bin/node" ] || die "NODE_TARBALL is not the Node ${NODE_VERSION} ${TARGET} distribution"
mv "$node_source" "$stage/runtime/node"

echo "==> Installing pinned ACP adapters and Codex CLI"
cp scripts/release/package.json scripts/release/package-lock.json "$stage/runtime/"
"$stage/runtime/node/bin/npm" ci --omit=dev --prefix "$stage/runtime"
[ -f "$stage/runtime/node_modules/@agentclientprotocol/claude-agent-acp/dist/index.js" ] || die "Claude ACP adapter was not installed"
[ -f "$stage/runtime/node_modules/@agentclientprotocol/codex-acp/dist/index.js" ] || die "Codex ACP adapter was not installed"
[ -x "$stage/runtime/$CLAUDE_NATIVE" ] || die "bundled Claude Code was not installed"
[ -x "$stage/runtime/$CODEX_NATIVE" ] || die "bundled Codex CLI was not installed"

# Claude ACP drops root quota refusals from held-turn background followups.
# Keep this repair tied to the complete reviewed upstream artifact (TS-04.R86).
claude_acp_source="$stage/runtime/node_modules/@agentclientprotocol/claude-agent-acp/dist/acp-agent.js"
claude_acp_sum="$(shasum -a 256 "$claude_acp_source" | awk '{print $1}')"
[ "$claude_acp_sum" = "$CLAUDE_ACP_SOURCE_SHA256" ] \
  || die "Claude ACP ${CLAUDE_ACP_VERSION} source does not match the reviewed patch input"
patch --fuzz=0 -p1 -d "$stage/runtime" < "scripts/release/patches/claude-acp-${CLAUDE_ACP_VERSION}-background-quota.patch"
claude_acp_sum="$(shasum -a 256 "$claude_acp_source" | awk '{print $1}')"
[ "$claude_acp_sum" = "$CLAUDE_ACP_BACKGROUND_QUOTA_SHA256" ] \
  || die "Claude ACP quota patch did not produce the complete reviewed output"
"$stage/runtime/node/bin/node" scripts/release/test-claude-quota-patch.mjs "$stage/runtime"

# Chuck's read-only quota passthrough (`_chuck/quota/read`, TS-04.R109) applies
# on top of the patch above; the contract test runs against the final bundle.
patch --fuzz=0 -p1 -d "$stage/runtime" < "scripts/release/patches/claude-acp-${CLAUDE_ACP_VERSION}-quota-read.patch"
claude_acp_sum="$(shasum -a 256 "$claude_acp_source" | awk '{print $1}')"
[ "$claude_acp_sum" = "$CLAUDE_ACP_QUOTA_READ_SHA256" ] \
  || die "Claude ACP quota-read patch did not produce the complete reviewed output"
"$stage/runtime/node/bin/node" scripts/release/test-claude-quota-read-patch.mjs "$stage/runtime"
"$stage/runtime/node/bin/node" scripts/release/test-claude-quota-patch.mjs "$stage/runtime"

# Chuck's complete-turn fork proof (`_meta.chuck.completeTurn`, TS-04.R113)
# extends R110's fork point on top of the quota-read patch. The quota-read test
# above pins that patch's exact initialize `_meta`, so it runs before this one.
claude_fork_source="$stage/runtime/node_modules/@agentclientprotocol/claude-agent-acp/dist/fork-session.js"
claude_fork_sum="$(shasum -a 256 "$claude_fork_source" | awk '{print $1}')"
[ "$claude_fork_sum" = "$CLAUDE_ACP_FORK_SOURCE_SHA256" ] \
  || die "Claude ACP ${CLAUDE_ACP_VERSION} fork source does not match the reviewed patch input"
patch --fuzz=0 -p1 -d "$stage/runtime" < "scripts/release/patches/claude-acp-${CLAUDE_ACP_VERSION}-complete-turn-fork.patch"
claude_acp_sum="$(shasum -a 256 "$claude_acp_source" | awk '{print $1}')"
claude_fork_sum="$(shasum -a 256 "$claude_fork_source" | awk '{print $1}')"
[ "$claude_acp_sum" = "$CLAUDE_ACP_PATCHED_SHA256" ] && [ "$claude_fork_sum" = "$CLAUDE_ACP_FORK_PATCHED_SHA256" ] \
  || die "Claude ACP complete-turn fork patch did not produce the complete reviewed output"
"$stage/runtime/node/bin/node" scripts/release/test-claude-complete-turn-patch.mjs "$stage/runtime"
"$stage/runtime/node/bin/node" scripts/release/test-claude-quota-patch.mjs "$stage/runtime"

# Codex ACP advertises steering but its idle branch starts a detached turn.
# Keep Chuck's no-consumption fallback explicit and version-locked until an
# upstream release provides the same request-level contract (TS-04.R51/R79).
codex_acp_source="$stage/runtime/node_modules/@agentclientprotocol/codex-acp/dist/index.js"
codex_acp_sum="$(shasum -a 256 "$codex_acp_source" | awk '{print $1}')"
[ "$codex_acp_sum" = "$CODEX_ACP_SOURCE_SHA256" ] \
  || die "Codex ACP ${CODEX_ACP_VERSION} source does not match the reviewed patch input"
# The patch is still semantically required only while upstream neither reads
# nor answers the idle-behavior request member (TS-06.R26).
if grep -q 'idleBehavior' "$codex_acp_source"; then
  die "Codex ACP ${CODEX_ACP_VERSION} already handles idle steering; re-review the steering patch"
fi
patch --fuzz=0 -p1 -d "$stage/runtime" < "scripts/release/patches/codex-acp-${CODEX_ACP_VERSION}-steering-prompt-required.patch"
codex_acp_sum="$(shasum -a 256 "$codex_acp_source" | awk '{print $1}')"
[ "$codex_acp_sum" = "$CODEX_ACP_STEERING_SHA256" ] \
  || die "Codex ACP steering patch did not produce the complete reviewed output"

# Chuck's read-only quota passthrough (`_chuck/quota/read`, TS-04.R109) applies
# on top of the steering patch; the final hash is the steering-verified artifact.
patch --fuzz=0 -p1 -d "$stage/runtime" < "scripts/release/patches/codex-acp-${CODEX_ACP_VERSION}-quota-read.patch"
codex_acp_sum="$(shasum -a 256 "$codex_acp_source" | awk '{print $1}')"
[ "$codex_acp_sum" = "$CODEX_ACP_QUOTA_READ_SHA256" ] \
  || die "Codex ACP quota-read patch did not produce the complete reviewed output"
"$stage/runtime/node/bin/node" scripts/release/test-codex-quota-read-patch.mjs "$stage/runtime"

# Chuck's complete-turn fork proof (`_meta.chuck.completeTurn`, TS-04.R113)
# applies last; the quota-read test above pins that patch's exact `_meta`.
patch --fuzz=0 -p1 -d "$stage/runtime" < "scripts/release/patches/codex-acp-${CODEX_ACP_VERSION}-complete-turn-fork.patch"
codex_acp_sum="$(shasum -a 256 "$codex_acp_source" | awk '{print $1}')"
[ "$codex_acp_sum" = "$CODEX_ACP_PATCHED_SHA256" ] \
  || die "Codex ACP complete-turn fork patch did not produce the complete reviewed output"
"$stage/runtime/node/bin/node" scripts/release/test-codex-complete-turn-patch.mjs "$stage/runtime"

# The direct Codex pin and codex-acp's range must dedupe. A nested second copy
# recreates the catalog/runtime contradiction that the private pin closes.
codex_package_count="$(find "$stage/runtime/node_modules" -path '*/@openai/codex/package.json' -type f | wc -l | tr -d ' ')"
[ "$codex_package_count" = "1" ] || die "private runtime resolved ${codex_package_count} Codex packages; expected exactly one"

# Prove both bundled providers run directly, without Node, before packaging:
# an explicit Bundle choice executes exactly these binaries (TS-06.R30).
codex_probe="$("$stage/runtime/$CODEX_NATIVE" --version 2>&1)" \
  || die "bundled Codex CLI is not runnable: $codex_probe"
case "$codex_probe" in
  *"$CODEX_CLI_VERSION"*) ;;
  *) die "bundled Codex CLI does not report expected version ${CODEX_CLI_VERSION}: $codex_probe" ;;
esac
claude_probe="$(HOME="$work" "$stage/runtime/$CLAUDE_NATIVE" --version 2>&1)" \
  || die "bundled Claude Code is not runnable: $claude_probe"
CLAUDE_CLI_VERSION="$(printf '%s\n' "$claude_probe" | awk 'NR==1 {print $1}')"
case "$CLAUDE_CLI_VERSION" in
  [0-9]*.[0-9]*.[0-9]*) ;;
  *) die "bundled Claude Code reported an unparseable version: $claude_probe" ;;
esac

"$stage/libexec/chuck" release wrapper --dir "$stage"
"$stage/libexec/chuck" release manifest --dir "$stage" --version "$VERSION" \
  --node "$NODE_VERSION" --claude-acp "$CLAUDE_ACP_COMPONENT_VERSION" --codex-acp "$CODEX_ACP_COMPONENT_VERSION" \
  --claude "$CLAUDE_CLI_VERSION" --codex "$CODEX_CLI_VERSION"

mkdir -p "$OUT_DIR"
"$stage/libexec/chuck" release package --dir "$stage" --output-dir "$OUT_DIR" --version "$VERSION"
echo "==> Release assets written to ${OUT_DIR}"
