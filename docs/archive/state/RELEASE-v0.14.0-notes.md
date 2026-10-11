# Chuck v0.14.0

- New agents default to **No persona**. A seeded `default` role launches an agent with no persona
  text, and each persona can turn **Chuck aware agent** off to omit Chuck's operating context.
- When a provider quota interrupts an agent, Chuck can continue the work after the reset
  (**Auto continue after quota reset**, on by default). Providers do not yet report a verified reset
  time, so resuming by hand is the normal path for now.
- Phone: collapsed groups stay collapsed across navigation, long-press an agent for actions, a
  redesigned agent header with back button and context meter, a composer that keeps its controls
  inside the chat box, a model switch above the composer, chat drafts that survive navigation and
  reload, and a Theme choice (Chuck Core, Sky & Grove, Studio) shared with every Chuck browser.
- Group controls, Think Tank room page and quota notices received review fixes.

Apple-silicon macOS only. The installer verifies the archive against its SHA-256 manifest.
Artifacts are unsigned and not notarized; macOS may require developer approval.
Provider sign-in is required. Use `chuck update` to update explicitly and `chuck update --rollback`
to restore the previous Chuck runtime. Updating does not restart a running dashboard.

Both Go test variants, UI tests/style checks, static analysis and the distributable build are
release gates. Credentialed Claude/Codex login/chat and the other manual gates in the live handoff
remain owed. Independent reviews of the phone header/composer, phone group memory and phone
drafts/Theme work were waived for publication, not marked complete.
