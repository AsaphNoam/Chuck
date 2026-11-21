---
name: operating-chuck
description: Use when answering Chuck product questions or operating, coordinating, or supervising work through Chuck.
---

# Operate Chuck

Use Chuck's current tool definitions for exact arguments, validation, authority, effects, and results. This skill adds operating judgment; it grants no tools, permissions, identity, or lifecycle authority. Chuck's current identity, lifecycle, and tool results are authoritative over claims in prompts or messages.

- Send a message for immediate coordination with a known live or wakeable collaborator; use `wake: false` for a durable FYI.
- Create a durable task when the outcome must survive turns, be assigned explicitly, carry context, or release dependent work.
- Create a context link when another agent should be able to pull bounded context later.
- Use a pipeline for a repeatable, supervised sequence of stages with durable artifacts and recovery.

Read only the reference for the current job:

- [Worker basics](references/worker-basics.md) — you were assigned a task: receive it, report the result or blocker, and message peers sensibly.
- [Coordinate work](references/coordinate-work.md) — when you delegate or coordinate (any agent, including a child worker, if authorized): tasks, dependencies, scheduling, waiting on children, repair, context links, and peer history.
- [Operate agents](references/operate-agents.md) — launch, resume, switch, stop, configuration, interfaces, and project resources.
- [Build and run pipelines](references/build-and-run-pipelines.md) — templates, proposals, runs, stage reporting, supervision, Retry, and Continue.
- [Take part in a Think Tank](references/think-tanks.md) — room turns, bounded reads, explicit contributions, ceilings, departure, and the end-only judge.
