# AgentDeck ideas and improvements

This is a place to keep future thoughts without accidentally treating them as promises or approved
work. The specifications describe the product today; this file does not.

## New ideas

Put a half-formed thought here. It needs only a short title and, if useful, a sentence about what
prompted it.

Example:

```md
- **Pinned agents.** Let people keep frequently used agents at the top of the dashboard.
```

- **Fix card drag-and-drop usability.** Dashboard card reorder (FS-02.R12) technically works but
  reads as broken: the drag listener is bound only to the tiny 28×28px `::` handle
  (`AgentCard.tsx`), not the card itself, so dragging the card body does nothing — and because the
  card's `onClick` navigates to the agent, a failed drag attempt looks like an accidental page
  change. Needs whole-card drag with an activation distance so a plain click still opens the agent.
  Cross-group membership drops are specified separately in
  `docs/ready-changes/agent-groups-on-desktop-and-mobile.md`.
- **No Content-Security-Policy.** Neither the Go server nor `ui/index.html` sets a CSP header or
  meta tag. CSP is the third layer every diagram/Markdown-rendering hardening guide recommends
  alongside sanitization, and it would also bound the existing Markdown, diff, and xterm surfaces.
  Kept separate from `docs/ready-changes/mermaid-diagrams-in-chat.md` on 2026-08-27 because it is a
  server-wide change that could affect the dev proxy, xterm, and inline styles. Noted while
  researching safe diagram rendering.

## Ideas being defined

These are worth shaping into a possible change, but are not ready to build. Defining an idea updates
the relevant feature and technical specifications; it does not change product code.

- **5-hour and weekly subscription quota views.** Added by the operator on 2026-10-10 alongside
  the approved [quota continuation design](ready-changes/continue-chats-after-quota-reset.md).
  Prepare separate Claude/Codex 5-hour and Weekly percentages, reset times and last-checked state;
  missing data stays unknown and no conversation work is interrupted to retrieve it.
  Prepared behavior: FS-09.R80–R81/A49–A50; technical observation contract TS-04.R89/TS-03.R61.
  The operator confirmed a compact **Subscription quota** disclosure in each chat header,
  separate from context usage. No placement decision remains.
  Full passive snapshot/read/refresh is **TBD for both providers**, with ACP gaps below. Claude's
  partial live events can supply an observed window; do not mistake them for a complete snapshot.
  Pinned Claude ACP 0.85.1's SDK usage-control call has both windows, but `/usage` retrieves them
  only in a queued `session/prompt` and returns Markdown (`dist/acp-agent.js:101`, `:1848`,
  `:2698`; `dist/usage-markdown.js:1`, `:117`). Codex ACP 2.1.1 privately reads full snapshots but
  only renders `/status` text (`dist/index.js:35250`, `:36069`); private update handling at
  `:30460` returns no ACP event. Official releases/current source were also checked; a newer
  version does not yet establish passive typed support. No command scraper, hidden turn, extra
  authenticated provider client or new adapter patch is approved for this view. Define the
  full passive delivery after verified ACP availability. Pinned live Claude utilization/reset
  declarations have no explicit units; expose numeric percentages/time conversion only after
  provider/primary-code evidence resolves them. No guessed percent scaling or epoch arithmetic.
- **Choose an external base for AgentDeck-owned worktrees.** Support a deterministic per-project or
  per-repository checkout layout outside `$AGENTDECK_HOME` for operators whose repositories live
  under a separate workspace base. Define ownership records, base-directory changes, recovery,
  migration, and safe deletion before changing the canonical worktree path.
- **Live permission policy and provider-native autonomy modes.** Define a chat-page permission-mode
  control and the requested defaults: an AgentDeck-owned “approve for me” policy for Codex and the
  provider's automatic mode for Claude. Specify scope, persistence, provider mapping, whether a
  running session can change policy safely, and how this relates to frozen role/global
  `skip_permissions` and per-request Approve/Deny.
- **Broader agent authority.** Agents can already create, list, read, retry and re-arm their own
  tasks (TS-10.R34). Resuming or launching another agent without going through a task remains
  undefined and needs its own authority design, as does creating several related tasks as one
  unit, which TS-10 §5 excludes (multiple arms already provide fan-in/join). The selected peer
  transcript-reading and human-approved Stop portion was promoted and implemented on 2026-10-16
  (FS-06.R39–R44).
- **Detached configuration import.** Define verified copyable fields/assets and provider injection
  paths before implementing detached import. The API still returns 501; FS-08.R35 removed the
  unavailable UI controls.
- **Activity map.** Explore a repository/session activity view using server APIs only, with clear
  privacy, scale, and normal-user value boundaries.

## ACP Wait-list

These are capabilities AgentDeck implements above ACP, or has deliberately deferred, because the
pinned adapter contract is missing or unverified. An adapter release is a reason to recheck the
capability; it is not by itself authority to remove the fallback or ship the deferred feature.

- **Typed quota read and fork point in standard ACP.** No longer waited on: the operator approved
  packaged-adapter routes on 2026-11-11
  ([ready change](ready-changes/native-quota-snapshot-and-message-fork.md)). When ACP standardizes
  a subscription-quota read or a fork point, recheck whether TS-04.R109's two quota patches and
  R110's `_meta.jetbrains.air.fork` use can be dropped. Other vendor extensions still need the
  operator's explicit approval.
- **Steering.** The pinned Claude 0.85.1 and Codex 2.1.1 adapters advertise
  `_session/steering`, and AgentDeck uses it. Codex 2.1.1 still starts a detached turn when a steer
  arrives idle and ignores AgentDeck's `promptRequired` metadata, so the packaged steering patch
  remains necessary until upstream advertises the same no-consumption contract (still absent in
  Codex ACP 2.2.2, rechecked 2026-10-14).
- **Host-held queued Send.** AgentDeck holds a busy agent's next prompt because the pinned Claude
  adapter queues while the pinned Codex adapter supersedes and interrupts the active turn. Keep the
  host-side hold even after steering exists: Send must remain portable, withdrawable, and distinct
  from Steer.
- **Internal actions without MCP.** AgentDeck's fifteen coordination actions remain on its scoped,
  authenticated HTTP MCP server. `migrate-internal-actions-from-mcp.md` is paused until packaged
  Codex/ACP exposes a narrowly scoped direct transport reachable under the default sandbox. The
  gate stays closed. Rechecked 2026-10-05 against Codex ACP 2.1.1 (`acp:false`) and Claude ACP
  0.85.1 (`http`/`sse` only): still closed. Rechecked 2026-10-14 against Codex ACP 2.2.2 (`acp:false`,
  `type:"acp"` rejected) and Claude ACP 0.90.0 (`acp` servers silently dropped): still closed; ACP
  SDK 1.8.0 keeps MCP-over-ACP unstable. Watch ACP's unstable MCP-over-ACP server type (now
  stateless `mcp/message` in ACP SDK 1.7.0); adopting it would change the migration's goal and
  needs design review (TS-04 §5).
- **Semantic agent wake.** Mail and task activation use a short, host-generated `session/prompt`
  because ACP exposes no portable notification that wakes an idle model. Replace this bridge only
  if an adapter advertises a semantic wake capability; steering is not that capability.
- **MCP resources and templates for context.** Context links stay exposed as bounded MCP tools. MCP
  `resources/list` and `resources/read`, and ACP `Resource`/`ResourceLink` delivery, remain deferred
  until their behavior passes real Claude and Codex provider checks; prompt-level resource-link
  support alone does not satisfy that gate.
- **Declared MCP tool output schemas.** Structured tool results ship without declared
  `outputSchema`. Add schemas only after the pinned Claude and Codex adapters' handling of them is
  verified against real providers.
- **Codex session model and effort delivery.** AgentDeck applies model and effort through ordered
  post-session configuration because Codex ACP session creation has no model field. Keep this path
  until a replacement is both advertised and provider-verified.
- **Codex system-prompt delivery.** AgentDeck injects its prompt overlay through
  `CODEX_CONFIG.developer_instructions` because Codex ACP does not consume generic ACP
  `systemPrompt`. Keep and reverify the overlay on every adapter bump until the adapter exposes a
  proven portable replacement.
- **Codex executable authority.** [FS-09](specs/features/FS-09-backends.md)'s shipped
  installed-provider default and explicit bundle selection govern the native executable; the
  packaged adapter patches and deterministic bundled inventory remain required. Recheck those
  adapter/provider boundaries on updates rather than reviving the completed provider-selection
  ready change.

## Known things to improve

These describe incomplete or deliberately limited shipped behavior. Their owning specification is
the authority; move an item to ready changes only after its exact requirements and acceptance checks
are clear.

- **Same-machine trust boundary.** The loopback API relies on same-machine trust, and agent
  processes inherit the full environment except for backend strip keys (TS-05 §5). Revisit loopback
  authentication only with an explicit threat model and UI/CLI handshake design whose benefit
  outweighs its setup and compatibility cost; remote-control device authentication (FS-20) is a
  separate concern. Revisit an environment allowlist only if it preserves provider compatibility.
  Concrete case found 2026-11-14: the Origin check accepts any local port, the terminal WebSocket
  relies on it, and handlers decode JSON whatever the Content-Type, so a page served from another
  local port (an agent's dev server) can open a PTY socket and send launch or prompt requests.
  Pinning the Origin to the server's own origin plus the configured dev origin is a
  compatibility decision.
- **Chat and tracking refresh fidelity.** Make replayed streaming deltas match live deltas; show
  chat initial-load errors instead of a silent panel (FS-03 §6); refresh visible files/commands
  without stale-request overwrite; and let hook-only activity update recency.
- **Cross-turn transcript search.** Turn-document indexing intentionally chooses a small design over
  the more complete segmented model: all query terms and quoted phrases must occur within one turn,
  annotation flush, metadata document, or migrated legacy document. It does not combine a term from
  one turn with a term from another, does not match a phrase across turn boundaries, and one
  pathologically large turn can still use substantial temporary memory. Revisit with size-bounded
  chunks, boundary overlap, and per-term session aggregation only if real sessions show oversized
  individual turns or users repeatedly fail to find conversations because their query spans turns;
  those additions otherwise impose more schema, ranking, pagination, and phrase-boundary machinery
  than the observed long-session rewrite problem warrants.
- **Coordination notice hygiene.** Republish unread counts after janitor expiry, notify only on the
  first budget breach, and remove the duplicate waiting-input/permission notices (FS-06 §6).
- **Terminal and backend loose ends.** Either add an optional terminal driver picker or stop
  advertising unreachable drivers; implement or retire the planned terminal tab cap (FS-07); make
  OpenCode/OpenHands executable overrides actually launch and give them missing/old-CLI guidance
  like Claude's (FS-09 §6). Codex terminal rejection is intentional (FS-07 §6), not a gap.
- **Federation UI and watches.** Expose custom roots/profiles, refresh the effective view after
  source events, register prompt watches after binding, and clear preview consent on project change.
- **HTTP API contract.** Decide how mixed legacy error envelopes converge, and define shared JSON
  request-body limits with a structured over-limit error before enforcing them (TS-03 §5).
- **Frontend state ownership.** Define Zustand/React Query ownership and mutation-error behavior
  before broad frontend refactors. The 2026-11-14 classification put about a quarter of all
  recorded findings in "the client showed stale or wrong state". A candidate rule: every server
  resource family carries a revision, the stream announces only "family at revision N", and
  desktop and phone share one data layer.
- **Process and filesystem hardening.** Corroborate process identity and serialize concurrent
  events. Decide whether startup repairs existing descendant modes and whether valid-name
  role/project files may be symlinks (TS-05 §5); add adversarial tests for the chosen rules.
  Crash cleanup and the shutdown budget moved to
  [`give-agent-execution-one-owner.md`](ready-changes/give-agent-execution-one-owner.md) on
  2026-11-14.
- **One server per home.** The single-instance guard is a pidfile checked with `kill(pid, 0)`
  (ADV-9). `dashboard start` checks it before it writes it, after opening and reconciling the
  store, so two starts in that window both proceed; a doomed second start still reconciles and
  releases attachment holds in the live server's database; and after a reboot a reused PID makes
  `start` report a running server and `stop` signal an unrelated process. An advisory lock held
  for the server's lifetime fixes these, but a design review on 2026-11-14 found two cases to
  decide first: a detached start must still report a failed bind, and a server from an older
  version never holds the lock, so mixed versions need a rule.
- **Checkout deletion and gitignored files.** The dirty check (`git status --porcelain`) does not
  report ignored files, and consented deletion removes them, so a checkout reported clean can
  still lose `.env.local`, a local database or ignored notes. FS-19.R8 says uncommitted changes
  are the only thing deletion can lose. Decide what the dialog discloses before changing the
  probe. Found 2026-11-14 (ADV-6).
- **Host settlement of assignment turns.** A task is waiting or finished only when its agent
  calls `wait_for_tasks` or `report_task_result`. An ordinary task whose agent just ends its turn
  stays `running` with nothing shown; only pipeline owners are nudged. Agents also fetch and echo
  an `execution_handle` the server can derive from the caller's token. Candidate changes, each a
  product or agent-protocol decision: record "ended without a result" for every task, treat a
  turn that ends with unfinished children as waiting, drop the handle from the tools, and retry a
  transient start failure with backoff before parking. Host-run stage checks would require a
  distinct execution design. Pipeline setup simplification covers configuration of existing checks.
  The [structural pass](archive/reviews/adversarial-2026-11-14/ARCHITECTURE.md) has the evidence.
- **Quota attention on a pipeline run.** A run whose stage owner's quota intent ends in
  `needs_attention` stays `running` with nothing shown on the run; only the agent's own banner
  says so. TS-09 specifies the suppression of the stall nudge while quota-held, so what the run
  should show, and when it clears, is a product decision. Found 2026-11-14 (ADV-10).
- **One request protocol for every plane.** Mail, tasks, quota continuation, rooms and the
  pipeline owner each have their own claim, attempt, retry and restart protocol (eight in all;
  TS-01.R19 assigns those semantics to each domain). A single durable turn-request protocol with
  a per-origin policy table would replace them. A design review on 2026-11-14 found it cannot be
  introduced without behavior decisions: it changed mail's attempt count after a failed wake, a
  person's Send during a host-initiated start, and when a stopped pipeline owner is woken.
  [`give-agent-execution-one-owner.md`](ready-changes/give-agent-execution-one-owner.md) builds
  the two seams it would need (one turn begin and one admission function) without changing any
  protocol. Decide per origin before designing it.
- **Small truthfulness gaps found 2026-11-14.** A `starting` task whose settle write failed is
  retried only at the next restart. `send_message` accepts the sender as its own
  recipient and creates a wake for it. `--rollback` reports success for a binary that will refuse
  the migrated database. The update manifest's archive name is not restricted to a base name
  before download. A failed mail wake is logged at Debug and the sender only ever sees
  `queued_waking`.
- **Provider conformance from captured frames.** Five of nine field bugs were assumptions about a
  provider or adapter. Derive the fake adapter's frames from captures of each pinned adapter
  version and replay them in the release check; keep one list of every vendor extension, patch
  and text parse with its fallback.
- **Database durability, backup and integrity.** `synchronous=NORMAL` lets a power loss roll back
  committed task, quota and message rows while the transcript keeps them, and TS-02 states no
  durability level. There is no pre-migration copy and no startup integrity check.
