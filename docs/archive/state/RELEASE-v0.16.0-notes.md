# Chuck v0.16.0

- **Think Tanks: participants can pass.** Any participant can pass a discussion turn when it has
  nothing useful to add, instead of writing filler or stalling the room. A pass uses that turn,
  shows a "passed" marker and keeps the participant in the discussion. When everyone still in the
  room has passed since the last real entry, discussion ends as "nothing further to add" and the
  judge, if configured, runs.
- **Agent page layout.** The agent card takes about 80% of the window with a centered
  conversation; the file viewer and annotations sit beside it. On narrower windows the header and
  composer toolbar wrap and side panels shrink so the conversation and composer stay usable.
- **Project rooms.** A project's Think Tank rooms now sit below its agents in a collapsible
  section; the collapse choice is shared across projects and Archive. Load errors stay visible
  while collapsed.
- **Archive search** matches conversation text only, not tool calls or file edits. Older
  archived conversations keep their tool text in search until `chuck reindex`.
- **Phone.** The connection recovers on its own after the phone sleeps or the network drops
  (heartbeat detection, one recovery at a time, capped retry backoff), and actions wait until it
  is healthy again. Chats and pages remember their scroll position, including after a live update
  reloads the page. Model and effort are separate controls (tap effort to cycle). Appearance
  wallpapers show on mobile again.
- **Performance.** Large agent updates, such as big images, are processed faster.
