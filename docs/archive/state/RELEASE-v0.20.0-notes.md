# Chuck v0.20.0

## Changes

- Follow files through saved tabs in desktop conversations, archived conversations and Think
  Tank rooms. Return to a file, close or reorder tabs, and recover the reading trail after
  navigation or reload. Phone Files gains tabs within its existing tracked-file access.
- Group suggestions and collapse choices now follow the selected project. Typed group names
  are accepted directly. Annotation recipients and room participants start with the current
  project; Include other projects expands the choices.
- Fresh installations skip permission prompts by default. Settings → Roles → Default permissions
  and persona overrides control new launches; upgrades preserve saved choices and existing
  sessions keep their frozen policy.
- Capturing an annotation reveals a collapsed tray and focuses the new instruction. Annotation
  input uses the regular text font.
- File-tab fixes preserve Back/Forward behavior, cited-line navigation and keyboard focus.
  Tab-limit refusals and damaged-storage recovery remain visible even before a first tab opens.

## Verification and limits

Release verification and the independent architectural review are recorded in the live handoff
and `docs/archive/reviews/architecture/v0.19.0-next.json`.

Saved tabs belong to browser site data and do not synchronize across devices. File contents are
read again when a workspace is restored; tabs are not retained file copies. Native phone touch,
background/reconnect and credentialed provider checks remain owed as listed in the handoff.

## Installation

This release supports Apple-silicon macOS. The archive includes Chuck's private Node and adapter
runtime. Provider selection defaults to Installed; Bundle is an explicit Settings choice.
The release is unsigned and not notarized, so macOS may require unidentified-developer approval.
Updates are explicit and preserve Chuck user data; restart the dashboard to use the new runtime.
