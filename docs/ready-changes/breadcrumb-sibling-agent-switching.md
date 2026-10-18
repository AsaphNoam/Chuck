# Switch between a project's running agents from the agent page

**State:** Waiting to start
**Why:** Direct `/design-feature` request 2026-10-18: navigate between agent chats without going
through the project page. The operator chose same-project scope, single-click entries, running
agents only, a `+n` overflow, and a compact card-like look that keeps the page uncluttered.
**Relevant requirements:** FS-03.R98/A76, FS-12.R70/A40, TS-08.R137–R138; existing FS-03.R27,
FS-02.R3/R45, FS-12.R39/R61; INV §1, §2, §13.

## Outcome

From any live agent's chat, one click opens another running agent in the same project, and a
waiting or failed sibling is visible without leaving the current chat.

## Included work

Mini agent cards (monogram tile with state dot plus name) replace the breadcrumb's
`project / agent` text: running siblings plus the current agent, dashboard order, five visible
and `+n` for the rest, live join/leave. Shared helpers for ordering, the five-cap and the overflow
disclosure with the header project links. Excluded: stopped siblings, cross-project switching,
archived pages, Think Tank rooms, phone, API/server, persistence and keyboard shortcuts.

## How we will know it works

FS-03.A76 (membership, order stability, current-stopped, overflow, unchanged single-agent and
archived breadcrumbs) through helper and component tests; FS-12.A40 through a real-browser pass with
`fakeacp` agents at 1024px and 1440px in Core, Sky & Grove and Studio. TS-06 UI closure,
style/presentation checks and `make embed`.

## Design direction

Options considered 2026-10-18: ⌘K quick switcher, header attention tray, chat-page agent rail and
breadcrumb sibling menu. The operator wanted same-project single-click access without crowding, so
the existing breadcrumb row carries the entries: no new row, sidebar or measured layout. Entries
look like shrunken agent cards and stay quiet until hover/focus, except the current agent and
attention states. Verified seams: `ChatPanel.tsx` `AgentBreadcrumb` (also used by
`ArchiveAgentPage`), `orderProjectAgents` in `projectDashboardData.ts`, `deriveActiveProjects`/
`ProjectNav` in `ActiveProjectNav.tsx`, `agentConversationPath`, and `.agent-monogram` styles in
`features/agent.css`. No server or provider capability is involved.

## Waiting on

Nothing.
