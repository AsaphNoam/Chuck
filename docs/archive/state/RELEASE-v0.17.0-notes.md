# Chuck v0.17.0

- **Agents can read each other's conversations.** A chat agent can read another Chuck agent's
  recorded conversation by its id, in any project and including stopped or archived agents. A
  read never wakes or messages the other agent.
- **Agents can ask you to stop another agent.** The request names the agent and a reason and
  waits for you to approve it once, on desktop or phone. No permission mode or earlier approval
  stands in for that decision; a denied request is final.
- **Quota recovery without a known reset time.** When a provider's quota failure carries a reset
  time Chuck can verify, the conversation continues shortly after it. Otherwise, with Auto
  continue on, Chuck checks access hourly (up to six times) in a separate, throwaway session that
  never touches your conversation, and continues the work only after a check succeeds. The
  conversation shows the next check apart from any real reset.
- **Context survives provider handoffs.** Switching an agent between providers or models carries
  its instructions, recent turns and tool evidence, and that bridge survives Stop/wake, restart
  and native compaction. A resume the provider cannot load fails visibly instead of quietly
  starting fresh. Switches that would need the bridge on OpenCode or OpenHands are refused before
  the agent is stopped.
- **Roomier file reader.** The file reader beside a conversation can be resized, and **Focus
  file** opens a file in a large reading view.
- **Phone annotations.** On the phone you can annotate events and tracked files, review the
  annotation tray, and launch a New task with annotations, as on desktop.
- **Phone: New agent from a group.** A group header starts a new agent with that group
  preselected (disabled while offline).
- **Think Tank setup.** Attach files to a room's goal, give the judge synthesis instructions, see
  each participant's model and effort during setup.
- **Withdraw gives your message back.** Withdrawing a held message puts its text back in that
  agent's draft on desktop and phone.
- **Default persona in Settings.** Settings → Roles can choose the default persona.
- **Tasks report reliably in skill workflows.** A launched task's agent gets a reporting reminder
  after your instruction, and Codex skills are invoked as `$name` so the assignee can report.
- **Fixes.** A send the server definitely refused no longer offers an unconfirmed Retry. Tool
  results stay in their run, and tool input that arrives late is recorded. Transcripts stay pinned
  while rows settle; stopped and errored agents show Resume. Card order is stable. Mail-triggered
  turns name the mail that started them. Deleted tasks disappear from every open view, and a
  pipeline whose stage was cut off by a restart pauses for Retry. Duplicate live user messages
  after a completed reply are gone. Invalid pipeline ids and blank backend names are refused with
  clear errors.

Live Claude/Codex provider checks and physical-device checks for these changes remain owed.
