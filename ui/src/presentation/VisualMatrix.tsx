import { useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { Badge, Button, PageHeader, ProjectColorPicker, Surface, VisuallyHidden } from "../components/ui";
import { AgentIdentity } from "../components/chat/ChatPanel";
import { ContextBar } from "../components/grid/ContextBar";
import { StateBadge } from "../components/grid/StateBadge";
import { EmptyState } from "../components/grid/EmptyState";
import { AssistantText } from "../components/chat/renderers/AssistantText";
import { DiffBlock } from "../components/chat/renderers/DiffBlock";
import { ToolCall } from "../components/chat/renderers/ToolCall";
import { ToolResult } from "../components/chat/renderers/ToolResult";
import { Composer } from "../components/chat/Composer";
import { FileViewerView, type FileReader } from "../components/chat/FileViewer";
import { QuotaBanner } from "../components/agent/QuotaBanner";
import { NoticeRow } from "../components/chat/renderers/NoticeRow";
import type { AgentState, AgentStatus } from "../api/types";
import { DndContext } from "@dnd-kit/core";
import { SortableContext } from "@dnd-kit/sortable";
import { AgentCard } from "../components/grid/AgentCard";
import { applyAppearance, effectiveAppearance, type EffectiveAppearance } from "../features/appearance/appearance";
import { PROJECT_COLOR_PRESETS } from "../lib/projectColors";
import { ProjectNav } from "../components/shell/ActiveProjectNav";
import "./contract-fixture.css";

const agentStates: AgentStatus[] = ["busy", "idle", "waiting_input", "done", "error", "unknown"];

// The dashboard fixture renders the real AgentCard (TS-08.R116), so this fills only
// the fields a card reads.
function matrixAgent(overrides: Partial<AgentState> & Pick<AgentState, "agent_id" | "name">): AgentState {
  return {
    role: "builder", project: "chuck-demo", backend: "codex", model: "gpt-fixture", interface: "chat",
    fast: false, fast_available: false, steering_available: false, created_at: "", running: true,
    state: "idle", detail: "Long operational detail remains bounded inside the card surface.",
    context_pct: 0, updated_at: 0, archived: false,
    ...overrides,
  };
}

export function VisualMatrix() {
  const [highVariance, setHighVariance] = useState(false);
  const [appearance, setAppearance] = useState<EffectiveAppearance>("core");
  const [liveState, setLiveState] = useState<AgentStatus | "stopped">("busy");
  const initialSkin = useRef(document.documentElement.getAttribute("data-skin"));

  useEffect(() => {
    applyAppearance(appearance === "core" ? "" : appearance);
  }, [appearance]);

  useEffect(() => () => {
    const previous = initialSkin.current;
    if (previous) document.documentElement.dataset.skin = previous;
    else document.documentElement.removeAttribute("data-skin");
  }, []);

  return (
    <div className={`visual-matrix ${highVariance ? "visual-matrix-high-variance" : ""}`}>
      <PageHeader
        eyebrow="Development fixture"
        title="Presentation matrix"
        description="Deterministic Chuck core surfaces for visual contract review."
        actions={(
          <div className="visual-matrix-controls">
            <label className="visual-matrix-toggle">
              Appearance
              <select
                aria-label="Fixture appearance"
                value={appearance}
                onChange={(event) => setAppearance(effectiveAppearance(event.target.value))}
              >
                <option value="core">Chuck Core</option>
                <option value="sky-grove">Sky & Grove</option>
                <option value="studio">Studio</option>
              </select>
            </label>
            <label className="visual-matrix-toggle">
              <input
                type="checkbox"
                checked={highVariance}
                onChange={(event) => setHighVariance(event.target.checked)}
              />
              High-variance contract
            </label>
          </div>
        )}
      />

      <nav className="visual-matrix-routes" aria-label="Fixture route links">
        <Link to="/">Dashboard</Link>
        <Link to="/archive">Archive</Link>
        <Link to="/settings">Settings</Link>
      </nav>

      <section className="visual-matrix-section">
        <h2>Active-project shell navigation</h2>
        <div className="visual-matrix-shell-row">
          <ShellNavFixture visible={[]} overflow={[]} />
          <ShellNavFixture visible={fixtureProjects.slice(0, 1)} overflow={[]} currentProjectID="alpha" />
          <ShellNavFixture visible={fixtureProjects.slice(0, 5)} overflow={[]} currentProjectID="alpha" />
          <ShellNavFixture visible={[...fixtureProjects.slice(0, 4), fixtureProjects[5]]} overflow={[fixtureProjects[4]]} currentProjectID="foxtrot" />
        </div>
      </section>

      <section className="visual-matrix-section">
        <h2>Shared construction</h2>
        <div className="visual-matrix-row">
          <Button variant="primary">Primary action</Button>
          <Button variant="secondary">Secondary action</Button>
          <Button variant="ghost">Quiet action</Button>
          <Button variant="danger">Destructive action</Button>
          <Button disabled>Disabled action</Button>
          <Button variant="icon" aria-label="Icon action">×</Button>
          <Button variant="icon-primary" aria-label="Icon primary action">→</Button>
        </div>
        <div className="visual-matrix-row">
          <Badge variant="neutral">Neutral</Badge>
          <Badge variant="info">Information</Badge>
          <Badge variant="success">Success</Badge>
          <Badge variant="warning">Warning</Badge>
          <Badge variant="danger">Error</Badge>
          <Badge variant="technical">Technical</Badge>
        </div>
        <Surface className="visual-matrix-surface">
          <label className="form-field">
            <span>Field label</span>
            <input defaultValue="Deterministic value" />
          </label>
          <p className="form-warning">A bounded warning stays attached to its field.</p>
          <p className="form-error">A mutation error remains visible.</p>
        </Surface>
      </section>

      <section className="visual-matrix-section">
        <h2>Chat session configuration</h2>
        <div className="chat-card">
          <header className="chat-header" data-slot="header">
            <AgentIdentity name="Nova orchestration reviewer" state="busy" status={<StateBadge state="busy" />} details={["reviewer", "Chuck", "Fast mode"]} />
            <div data-slot="context"><ContextBar detailed value={0.34} used={68000} size={200000} /></div>
          </header>
          <QuotaBanner value={{ episode_id: "fixture-episode", revision: 3, observed_at: "2026-10-10T10:00:00Z", reset_at: "2026-10-10T12:00:00Z", due_at: "2026-10-10T12:01:00Z", continuation_state: "scheduled" }} onCancel={() => undefined} />
          <QuotaBanner value={{ episode_id: "fixture-check", revision: 2, observed_at: "2026-10-10T10:00:00Z", reset_at: null, due_at: "2026-10-10T11:00:00Z", continuation_state: "scheduled", next_action: "check", checks_remaining: 5 }} onCancel={() => undefined} />
          <div className="chat-runtime-picker">
            <fieldset className="chat-runtime-staged">
              <legend>Runtime</legend>
              <div className="form-field"><label htmlFor="matrix-backend">Backend</label><select id="matrix-backend" defaultValue="codex"><option value="codex">Codex (codex)</option></select></div>
              <div className="form-field"><label htmlFor="matrix-model">Model</label><select id="matrix-model" defaultValue="gpt-5"><option value="gpt-5">GPT-5 (gpt-5)</option></select></div>
            </fieldset>
            <fieldset className="chat-session-settings">
              <legend><VisuallyHidden>Session settings</VisuallyHidden></legend>
              <div className="form-field"><label htmlFor="matrix-effort">Effort</label><select id="matrix-effort" defaultValue="high"><option value="low">low</option><option value="high">high</option></select></div>
              <label className="chat-fast-toggle"><input type="checkbox" defaultChecked /> Fast mode — higher provider usage</label>
            </fieldset>
            <fieldset className="chat-permission-modes">
              <legend>Permissions</legend>
              <div className="chat-permission-mode-options">
                {["Manual", "Accept Edits", "Plan Mode", "Don't Ask"].map((name, index) => (
                  <label key={name} className="chat-permission-mode"><input type="radio" name="matrix-permission-mode" defaultChecked={index === 1} />{name}</label>
                ))}
              </div>
            </fieldset>
            <div className="chat-runtime-pending">
              <span>Unapplied changes</span>
              <button className="chat-runtime-switch ad-button-primary" type="button">Switch</button>
              <button className="ad-button-secondary" type="button">Discard</button>
            </div>
            <p className="form-error" role="alert">Runtime switch failed; current settings were restored.</p>
          </div>
        </div>
      </section>

      <section className="visual-matrix-section" data-ui="dashboard">
        <h2>Dashboard states</h2>
        <div className="visual-matrix-agent-grid" data-slot="groups">
          <DndContext>
          <SortableContext items={[]}>
          {agentStates.map((state, index) => (
            <AgentCard
              key={state}
              showProject={false}
              onToggle={() => undefined}
              agent={matrixAgent({
                agent_id: `matrix-${state}`,
                name: state === "waiting_input" ? "Needs a decision from the orchestration and delivery reviewer" : state === "error" ? "unbroken-agent-name-that-must-not-escape-the-card-boundary" : `${state} agent`,
                state,
                model: `gpt-fixture-${index + 1}`,
                context_pct: index * 0.17,
                unread_messages: index === 2 ? 3 : undefined,
                last_sent_at: index === 3 ? "2026-10-10T00:00:00Z" : undefined,
                quota: index === 0 ? { episode_id: "fixture-unknown-reset", revision: 1, observed_at: "2026-10-10T10:00:00Z", reset_at: null, due_at: null, continuation_state: "unknown_reset" } : undefined,
              })}
            />
          ))}
          <AgentCard showProject={false} agent={matrixAgent({ agent_id: "matrix-stopped", name: "Stopped agent", role: "reviewer", interface: "terminal", driver: "xterm", running: false, detail: "" })} />
          <AgentCard
            expanded
            showProject={false}
            onToggle={() => undefined}
            agent={matrixAgent({
              agent_id: "expanded-fixture",
              name: "Expanded long-name agent fixture",
              backend: "anthropic-extended-thinking-partner",
              model: "claude-opus-4-fixture-long-runtime-identity-string",
              effort: "maximum-reasoning-depth",
              state: liveState === "stopped" ? "busy" : liveState,
              running: liveState !== "stopped",
              context_pct: 0.74,
              context_used: 153482,
              context_size: 200000,
            })}
          >
            <MatrixChatPane liveState={liveState} />
          </AgentCard>
          </SortableContext>
          </DndContext>
        </div>
        <label className="visual-matrix-toggle">Expanded card state
          <select aria-label="Expanded card state" value={liveState} onChange={(event) => setLiveState(event.target.value as AgentStatus | "stopped")}>
            {[...agentStates, "stopped"].map((state) => <option key={state} value={state}>{state}</option>)}
          </select>
        </label>
        <EmptyState onNewAgent={() => undefined} />
        <div className="context-menu" data-ui="context-menu" role="menu">
          <div className="context-menu-color" role="menuitem">
            <span>Project color picker</span>
            <ProjectColorPicker value={PROJECT_COLOR_PRESETS[0].color} onChange={() => undefined} />
          </div>
        </div>
      </section>

      <section className="visual-matrix-section">
        <h2>Transcript and technical surfaces</h2>
        <div className="visual-matrix-transcript" data-ui="transcript">
          <article className="message user-message" data-slot="event" data-variant="user">
            Preserve product behavior while changing presentation.
          </article>
          <article className="message assistant-message" data-slot="event" data-variant="assistant">
            <h3>Assistant response</h3>
            <p>Readable Markdown uses a deliberate measure and technical detail stays distinct.</p>
            <code>inline_code()</code>
          </article>
          <AssistantText event={{ seq: 17, kind: "assistant_text", text: "```ts\nconst appearance = 'sky-grove';\n```" }} />
          <AssistantText event={{ seq: 19, kind: "assistant_text", text: "```mermaid\ngraph TD;\n  Deck-->Agent;\n  Agent-->Task;\n```" }} />
          <AssistantText event={{ seq: 20, kind: "assistant_text", text: "```mermaid\ngraph LR;\n  Intake-->Plan-->Implement-->Verify-->Release;\n```" }} />
          <NoticeRow event={{ seq: 21, kind: "notice", origin: "chuck", continuation_attempt: "fixture-attempt", severity: "info", title: "Chuck automatically continued this conversation after a quota interruption" }} />
          <DiffBlock
            event={{ seq: 18, kind: "diff", path: "ui/src/styles/skins/sky-grove.css", old_text: "--surface: core;", new_text: "--surface: sky;" }}
            onAnnotate={() => undefined}
          />
          <article className="tool-run" data-ui="tool-run" data-state="collapsed">
            <button className="tool-toggle" data-slot="trigger" type="button">▸ Ran 2 tools</button>
          </article>
          <article className="tool-run" data-ui="tool-run" data-state="expanded">
            <button className="tool-toggle" data-slot="trigger" type="button">▾ Ran 2 tools</button>
            <div className="tool-run-content" data-slot="content">
              <ToolCall event={{ kind: "tool_call", name: "inspect_workspace", args: { path: "ui/src" } }} />
              <ToolCall event={{ kind: "tool_call", name: "search_files" }} />
              <ToolResult event={{ kind: "tool_result", status: "completed", content: "Representative tool output" }} />
              <ToolResult event={{ kind: "tool_result", status: "failed", error: "Representative tool error output" }} />
            </div>
          </article>
          <div data-ui="turn-activity" data-state="collapsed">
            <div className="turn-activity-toggle"><button className="tool-toggle" data-slot="trigger" type="button" aria-expanded="false">▸ Show activity</button></div>
          </div>
          <div data-ui="turn-activity" data-state="expanded">
            <div className="turn-activity-toggle"><button className="tool-toggle" data-slot="trigger" type="button" aria-expanded="true">▾ Hide activity</button></div>
            <div className="turn-activity-content" data-slot="content">
              <ToolCall event={{ kind: "tool_call", name: "inspect_workspace" }} />
            </div>
            <p className="turn-outcome">Cancelled — response is partial</p>
          </div>
          <article className="permission-prompt" data-ui="permission-prompt" data-state="pending">
            <strong data-slot="title">Permission required</strong>
            <p data-slot="reason">Run a bounded local verification command.</p>
            <div data-slot="actions"><button type="button" className="ad-button-secondary">Deny</button><button type="button" className="ad-button-primary">Approve</button></div>
          </article>
          <div className="terminal-panel" data-ui="terminal">
            <pre className="visual-matrix-terminal" data-slot="viewport">$ make test{"\n"}all checks passed</pre>
          </div>
        </div>
      </section>

      <ReaderFixtures />

      <section className="visual-matrix-section visual-matrix-columns">
        <article className="pipeline-run-page" data-ui="pipeline-run">
          <section className="pipeline-run-hero" data-slot="live">
            <div className="pipeline-run-kicker"><code>run_release</code><Badge variant="warning">needs attention</Badge></div>
            <div className="pipeline-run-title"><div><h2>Release verification</h2><p>Compare Core and Sky & Grove against the same deterministic fixture.</p></div><div className="pipeline-live-stage"><small>Current stage</small><strong>Visual review</strong><span>Visit 2 · attempt 4</span></div></div>
          </section>
          <section className="pipeline-timeline" data-slot="timeline">
            <ol><li className="pipeline-timeline-item pipeline-timeline-current" data-slot="attempt"><span className="pipeline-timeline-line" aria-hidden="true" /><details open><summary><span className="pipeline-stage-number">04</span><span className="pipeline-attempt-identity"><strong>Visual review</strong><small>Claude · Sonnet</small></span><span className="pipeline-state pipeline-state-paused">paused</span><span className="pipeline-disclosure-chevron">⌄</span></summary><div className="pipeline-attempt-body"><p className="pipeline-result-summary">One focused attempt owns the main reading column.</p><div className="pipeline-agent-grid" data-slot="agents"><div className="pipeline-agent-card"><span className="pipeline-agent-dot pipeline-agent-dot-idle" /><span><strong>Reviewer</strong><small>Stage agent · waiting</small><em>Review ready for approval</em></span><span>↗</span></div></div></div></details></li></ol>
          </section>
        </article>
        <div className="dialog-content onboarding-wizard" data-ui="dialog" data-slot="content" data-variant="onboarding">
          <div className="onboarding-flow" data-ui="onboarding" data-state="current">
            <div className="wizard-progress" data-slot="progress">Step 2 of 4</div>
            <div className="onboarding-step" data-slot="content">
              <h2>Choose a project</h2>
              <p>Onboarding keeps its existing structure and behavior in both appearances.</p>
            </div>
            <div className="onboarding-actions" data-slot="actions"><Button>Back</Button><Button variant="primary">Continue</Button></div>
          </div>
        </div>
      </section>

      <section className="visual-matrix-section visual-matrix-columns">
        <div data-ui="archive">
          <h2 data-slot="header">Archive result</h2>
          <div className="archive-row" data-slot="result" data-state="inactive">
            <div className="archive-row-top" data-slot="metadata"><span className="archive-name">Interface redesign</span><Badge variant="success">inactive</Badge></div>
            <p className="archive-snippet" data-slot="snippet">…semantic tokens and stable hooks…</p>
            <div className="archive-row-meta" data-slot="metadata"><span>12 turns</span><span>8 files</span></div>
          </div>
        </div>
        <div data-ui="settings">
          <h2 data-slot="header">Settings editor</h2>
          <div className="backend-card" data-ui="config-editor" data-variant="backends" data-slot="item">
            <div className="backend-card-header"><strong>Codex</strong><Badge variant="success">ready</Badge></div>
            <div className="model-row"><code className="config-slug">gpt-fixture</code><span className="config-badge">default</span></div>
            <div className="source-panel" data-ui="config-source" data-state="ok"><span className="source-health source-health-ok" data-slot="status">ok</span><code className="source-root" data-slot="root">~/.codex</code></div>
          </div>
        </div>
      </section>

      <section className="visual-matrix-section">
        <h2>Overlay and feedback</h2>
        <div className="visual-matrix-overlay-sample">
          <div className="visual-matrix-dialog" data-ui="dialog" data-slot="content" data-variant="default">
            <h3 data-slot="title">New agent</h3>
            <p data-slot="body">Dialog geometry, fields, and actions share the core construction.</p>
            <div className="form-actions" data-slot="actions"><Button>Cancel</Button><Button variant="primary">Launch</Button></div>
          </div>
          <button className="toast done" data-ui="toast" data-state="done" type="button">
            <strong data-slot="title">Agent finished</strong><span data-slot="body">The fixture task completed.</span>
          </button>
        </div>
      </section>
    </div>
  );
}

const readerMarkdown = [
  "# Reader fixture",
  "",
  "Prose wraps `internal/server/deeply/nested/directory/with/a/long/path/handler_with_a_long_name.go` inside the document.",
  "",
  "| Column | A deliberately wide value |",
  "|---|---|",
  `| alpha | ${"wide-value-".repeat(12)} |`,
  "",
  "```go",
  `func Handler() { log.Printf("${"a long log line ".repeat(8)}") }`,
  "```",
  "",
  "> A quoted note keeps the document rhythm.",
].join("\n");

// One static reader per presentation the workspace can take (TS-08.R132): the
// fixture shares the real view; widths come from the fixture stylesheet.
function matrixReader(): FileReader {
  return {
    link: { path: "docs/reader-fixture-with-a-deliberately-long-file-name.md" },
    view: { status: "loaded", readAt: new Date("2026-10-17T09:00:00Z"), file: { agent_id: "fixture", path: "docs/reader-fixture-with-a-deliberately-long-file-name.md", size: readerMarkdown.length, mod_time: "", line_count: 13, content: readerMarkdown, truncated: false, language: "markdown" } },
    rendered: true,
    setRendered: () => undefined,
    reload: () => undefined,
    positions: { current: { rendered: null, formatted: null, source: null } },
    pendingLine: { current: null },
  };
}

function MatrixTray({ collapsed = false }: { collapsed?: boolean }) {
  return (
    <aside className="annotation-tray" data-ui="annotation-tray" data-state={collapsed ? "collapsed" : "expanded"} aria-label="Pending annotations">
      <header className="annotation-tray-header">
        <div><strong className="annotation-tray-title">Pending annotations</strong><span>2/20</span></div>
        <div><button type="button" className="annotation-tray-collapse ad-button-icon" aria-label={collapsed ? "Expand pending annotations" : "Collapse pending annotations"}>{collapsed ? "‹" : "›"}</button><button type="button" className="annotation-tray-discard ad-button-danger">Discard all</button></div>
      </header>
      <div className="annotation-tray-body">
        <ol className="annotation-drafts"><li className="annotation-draft"><div className="annotation-draft-head"><h3 className="annotation-draft-anchor">File docs/reader-fixture.md</h3></div><blockquote>Prose wraps inside the document.</blockquote></li></ol>
      </div>
      <footer className="annotation-tray-footer"><button type="button" className="annotation-send ad-button-primary">Send annotations</button></footer>
    </aside>
  );
}

function MatrixDivider({ variant }: { variant: "file" | "annotations" }) {
  return <div className="workspace-divider" data-ui="workspace-divider" data-variant={variant} role="separator" aria-orientation="vertical" aria-label={variant === "file" ? "Resize file and conversation" : "Resize annotations and conversation"} tabIndex={0} />;
}

function MatrixConversation({ overlay = false }: { overlay?: boolean }) {
  return (
    <div className="chat-card">
      <div className="transcript-wrap">
        <div className="transcript-view" data-slot="list">
          <AssistantText event={{ kind: "assistant_text", text: "The conversation keeps its own reading column beside the file." }} />
        </div>
        {overlay && <MatrixTray />}
      </div>
    </div>
  );
}

function ReaderFixtures() {
  return (
    <section className="visual-matrix-section visual-matrix-reader">
      <h2>File reader workspace</h2>
      <div className="chat-stage" data-file="open" data-tray="docked">
        <div className="chat-side chat-side-start"><FileViewerView reader={matrixReader()} variant="docked" onClose={() => undefined} actions={<button type="button" className="ad-button-secondary">Focus file</button>} /></div>
        <MatrixDivider variant="file" />
        <MatrixConversation />
        <MatrixDivider variant="annotations" />
        <div className="chat-side chat-side-end"><MatrixTray /></div>
      </div>
      <div className="chat-stage" data-file="open" data-tray="collapsed">
        <div className="chat-side chat-side-start"><FileViewerView reader={matrixReader()} variant="docked" onClose={() => undefined} /></div>
        <MatrixDivider variant="file" />
        <MatrixConversation />
        <div className="chat-side chat-side-end"><MatrixTray collapsed /></div>
      </div>
      <div className="chat-stage visual-matrix-reader-narrow" data-file="open" data-tray="overlay">
        <div className="chat-side chat-side-start"><FileViewerView reader={matrixReader()} variant="docked" onClose={() => undefined} /></div>
        <MatrixDivider variant="file" />
        <MatrixConversation overlay />
      </div>
      <div className="dialog-content file-focus visual-matrix-focus">
        <div className="file-focus-body" data-tray="closed">
          <FileViewerView reader={matrixReader()} variant="focused" onClose={() => undefined} closeLabel="Close file" actions={<button type="button" className="ad-button-secondary">Return to chat</button>} />
          <MatrixTray />
        </div>
      </div>
    </section>
  );
}

// The expanded card's pane, outside the dashboard section so its slot belongs to the
// agent card it renders in.
function MatrixChatPane({ liveState }: { liveState: AgentStatus | "stopped" }) {
  return (
    <div className="dashboard-chat-pane" data-slot="chat-pane">
      <div className="transcript-wrap">
        <div className="transcript-view" data-ui="transcript" data-slot="list">
          <AssistantText event={{ kind: "assistant_text", text: "A populated conversation keeps the remaining card height.\n\n```ts\nconst result = await verify();\n```" }} />
        </div>
      </div>
      <div className="dashboard-chat-composer">
        <Composer agentId="expanded-fixture" busy={liveState === "busy"} running={liveState !== "stopped"} variant="dashboard" />
      </div>
    </div>
  );
}

const fixtureProjects = ["Alpha", "Bravo workspace with a long title", "Charlie", "Delta", "Echo", "Foxtrot"].map((title, index) => ({
  id: title.split(" ")[0].toLowerCase(),
  title,
  color: PROJECT_COLOR_PRESETS[index].color,
}));

function ShellNavFixture({ visible, overflow, currentProjectID }: Parameters<typeof ProjectNav>[0]) {
  return (
    <div className="app-header visual-matrix-shell">
      <div className="app-logo">Chuck</div>
      <nav className="app-nav" aria-label="Fixture primary navigation"><a href="#dashboard">Dashboard</a><a href="#tasks">Tasks</a><a href="#pipelines">Pipelines</a><a href="#archive">Archive</a><a href="#settings">Settings</a></nav>
      <ProjectNav visible={visible} overflow={overflow} currentProjectID={currentProjectID} />
      <div className="app-connection"><span className="connection-status">Connected</span></div>
    </div>
  );
}
