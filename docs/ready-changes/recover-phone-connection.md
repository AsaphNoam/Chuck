# Recover the phone connection without losing the person's place

**State:** Waiting to start
**Why:** Operator report and design request, 2026-10-14: remote mobile sometimes looks disconnected
after background/return and needs manual refresh. The operator subsequently asked for technical
evaluation/selection, including foreground heartbeats, healthy-stream reuse and low-cost retries.
**Relevant requirements:** FS-20.R17/R23/R49/R51–R53/A22–A25,
TS-13.R14/R18/R27–R30, TS-03.R7, TS-08.R103, INV §1/§4/§5/§16.

## Outcome

Returning to Chuck or encountering a silent foreground stall recovers current Mac state without
refreshing the page, losing navigation/reading position/forms/text/files or replaying an action.
Healthy connections stay in use and outages do not cause an aggressive background retry loop.

## Included work

Extend the existing phone connection owner with heartbeat liveness and serialized foreground
recovery/backoff, retaining its snapshot/transcript catch-up and shared browser-local drafts.
Keep screens and entered forms mounted across refresh; preserve the rendered message anchor.
Use existing connection/unreachable/unpaired feedback and bounded history fallback. No product code
has changed in design. Server wire/cadence, pairing/security, persistence/retention, desktop SSE,
agent runtime, cold-start scroll restoration and mutation replay are outside this change.

Evidence and choice:

- `internal/server/sse.go:12` sends a flushed, dispatched `ping` every 10 seconds; `:27` advertises
  `retry: 2000`. `internal/bus/bus.go:120` constructs a real `ping`, not a comment that JS cannot see.
- `ui/src/remote/connection.ts:77` has no ping listener, watchdog or lifecycle recovery; its error
  path leaves native retries running. `:56` can mark a successful health read connected before
  snapshot completion. `PhoneApp.tsx:75` classifies/connects only on mount. Exact device failure
  is still a hypothesis; missing protection is established by source inspection.
- Desktop `ui/src/api/sse.ts:101/:285` already uses pings and a 25-second stale threshold. Reuse
  that liveness margin with one rearmed foreground deadline, keeping a recently healthy stream
  rather than replacing it on every return. Close failed sources and own capped foreground retry
  to stop competing two-second native retries. A successful HTTP probe alone cannot verify SSE.
- Retain an existing healthy hidden stream to avoid a full snapshot/transcript refresh on every
  short app switch. Pause recovery timers/checks while hidden and close failed/incomplete streams.
  Closing all healthy streams on hide would guarantee less background traffic but impose a fresh
  connection and snapshot on every return; no zero-background-traffic requirement was requested.
- A periodic watchdog or HTTP probe adds recurring work; the selected deadline adds no healthy
  network requests. [EventSource.close](https://html.spec.whatwg.org/dev/server-sent-events.html)
  aborts the stream, enabling app-owned retry. Browser suspension/timer delays remain platform
  constraints, addressed on return and verified on actual devices.

## How we will know it works

FS-20.A22–A25: fake-clock/EventSource failure and resource-bound tests; current-state catch-up,
draft/form/attachment preservation and rendered reading-position journey at phone width; healthy
and outage request-count checks; real paired iPhone Home Screen and Android background/return gates.
Run TS-13.R30's applicable TS-06 closure matrix after implementation. Source inspection, simulated
events and documentation checks are not device reproduction or a measured battery claim.

## Waiting on

None for implementation. Device receipts are implementation acceptance gates, not pre-design blockers.
