# Chuck v0.23.0

## Changes

- Search chats from a project's heading, across running, stopped and archived conversations.
  Results prioritize active work and open the matching source passage in the shared chat reader.
  If an exact passage is unavailable, the conversation can still be opened normally.
- Keyboard selection stays visible in long search results. Leaving search restores normal
  transcript following while active passage reading keeps its position during new replies.
- Managed Claude sessions preserve root quota refusals after background subagent work, allowing
  Chuck's existing quota recovery to recognize them. This repair updates the managed adapter;
  globally installed PATH adapters are unchanged.
- Accepted cancellation wins over late quota evidence. Isolated access checks use the continued
  process's model and provider route and accept recovered successful replies correctly.
- Temporary policy-read failures leave quota work retryable, and stalled probe cleanup no longer
  blocks the shared work dispatcher.

## Verification and limits

The combined architectural source audit and verified quota repairs are reused in
`docs/archive/reviews/architecture/v0.22.0-next.json`; no second architecture review was run.
Recorded full Go and UI closure is reused for unchanged inputs. Release preparation builds
the embedded UI and FTS5 binary and verifies operating-package/release contracts.
Hosted archive assembly, transaction/bootstrap checks and fresh installation passed on publication.
The archive, checksum manifest and installer are attached to the GitHub Release. Credentialed provider,
physical-device and native journeys remain owed as listed in the live handoff.

The separate main CI retry passed Go checks but failed one dashboard waiting-input pane UI test;
the focused local CardGrid suite passes. This remains an open finding in the handoff. The first
main CI attempt also exposed an annotation replay fixture race, recorded for follow-up.

The release range requires no operating-chuck content change: quota guidance already describes
the recovery policy and search adds a human UI capability without changing agent-tool authority.
README/install instructions remain accurate. Managed Claude ACP is reported as
`0.85.1+chuck.1`; its version-locked patch and complete input/output hashes are verified in assembly.

## Installation

Apple-silicon macOS only. The archive includes Chuck's private Node and adapter runtime.
Provider selection defaults to Installed; Bundle is an explicit Settings choice. The release
is unsigned and not notarized. Updates preserve user data; restart the dashboard to use the
new application runtime. Existing running dashboards and global adapters are not patched in place.
