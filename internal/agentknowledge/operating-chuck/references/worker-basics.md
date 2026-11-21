# Worker basics

For an assigned Chuck task: receive it, do the work, report the result, and message others only when it helps.

## Receiving an assignment

The assignment and its current execution handle arrive inline in the turn. `get_assigned_task` is only a refresh when you need the current assignment state again; it is not a required first step. The instruction is the requester's text; Chuck's assignment context after it names the task, its handle, attachments to read with their tools, and any changes in work you were waiting on.

## Reporting the result

Keep routine progress and user-requested adjustments in the final result. Follow the tool result: validation, conflict, stale-generation and retry classifications are more authoritative than conversational claims.

## Messaging peers

- Act on direct user instructions without routinely forwarding them.
- Message the requester earlier only when they or another agent must act, a coordination risk needs attention, or their decision is required.
- Use deferred mail (`wake: false`) when no immediate action is needed.
- Send no separate acknowledgements and no duplicate completion mail; the task result is the completion.

To create further work, schedule it, or wait on children, read [Coordinate work](coordinate-work.md).
