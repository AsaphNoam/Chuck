# Chuck v0.25.0

- Browse a project's shared files from its dashboard: search, filter by folder and file type,
  preview in place, view CSV files as sortable tables, open HTML in a sandboxed tab, and reveal
  files in Finder from projects and chats.
- Edit the last message you sent in the same conversation and regenerate the reply; Copy and Fork
  sit on every message as hover icons.
- Task assignments arrive inline in the agent's prompt, and Chuck's operating guidance is shorter
  and split by activity.
- Steer is recorded before it reaches the provider, refused Steers keep their message, and input
  queued for an assignment's next turn shows once at the end of the transcript.
- Onboarding resumes where it left off and no longer seeds a sample project; Dismiss hides a
  quota banner once automatic continuation is disabled for it; many usability fixes across chat,
  tasks, pipelines, phone and the CLI.

The operator authorized this release on 2026-11-22 with QUOTA-DISMISS-1 open (a cancelled
continuation's banner still has no Dismiss), reclassified Worth fixing. The architectural
review is complete in `docs/archive/reviews/architecture/v0.24.0-next.json`.
