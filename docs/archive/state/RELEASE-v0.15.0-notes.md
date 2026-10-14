# Chuck v0.15.0

`v0.14.0` was tagged but never published, so these notes cover everything since `v0.13.0`.

- **Images and files in chat.** Attach, paste or drop images and files into a chat message or
  Steer. Images reach image-capable agents as images; other files arrive as read-only copies the
  agent opens with its normal tools. Sent attachments stay viewable after reload, restart and
  archive, and clones keep their own copies. Assistant and local images open in a viewer.
- **Permission modes from the chat header.** When a running chat agent's provider offers modes
  (for Claude: Manual, Accept Edits, Plan Mode, Don't Ask, Bypass Permissions), switch between them
  without restarting. The mode resets to the provider default on resume or relaunch.
- **Full operator CLI.** Everything the dashboard does — agents, conversations, projects, groups,
  worktrees, archive, pipelines, tasks, Think Tanks, configuration and remote administration — is
  now available as `chuck <family> <operation>` against the running service, with `--json`,
  file/stdin input and `--help` examples. See `docs/operator-cli.md`.
- **Projects.** A redesigned projects home with larger cards, and a create menu on the project
  page's background. File changes no longer clutter agent chats; the Files tab still shows diffs.
- **New agents default to No persona.** The seeded `default` role launches with no persona text,
  and each persona can turn **Chuck aware agent** off.
- **Quota interruptions.** Chuck can continue work after a provider quota resets (**Auto continue
  after quota reset**, on by default). Providers do not yet report a verified reset time, so
  resuming by hand is the normal path for now.
- **Phone.** Redesigned agent header with back button and context meter, 44px touch targets, a
  model switch above the composer, drafts that survive navigation and reload, groups that stay
  collapsed, long-press agent actions, and a Theme choice (Chuck Core, Sky & Grove, Studio).
- **Fixes.** Attachment recovery isolates a corrupt file to that item, conversation deletion
  is retryable, provider-reported modes survive empty replies, and project header buttons keep
  their own context menus.

Apple-silicon macOS only. The installer verifies the archive against its SHA-256 manifest.
Artifacts are unsigned and not notarized; macOS may require developer approval.
Provider sign-in is required. Use `chuck update` to update explicitly and `chuck update --rollback`
to restore the previous Chuck runtime. Updating does not restart a running dashboard.

Both Go test variants, UI tests/style checks, static analysis and the distributable build are
release gates. Credentialed Claude/Codex login/chat, live permission-mode receipts and the other
manual gates in the live handoff remain owed.
