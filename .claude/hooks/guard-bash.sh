#!/bin/sh
# PreToolUse[Bash]: workflow guards for shell commands.
# Normal pushes follow the workflow's publication authorization; passing this
# hook grants no authority. History rewrites additionally prompt for confirmation.
command -v jq >/dev/null 2>&1 || exit 0
cmd=$(jq -r '.tool_input.command // empty' 2>/dev/null) || exit 0
case "$cmd" in
*git*push*--force*|*git*push*\ -f\ *|*git*push*\ -f)
  printf '%s\n' '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"ask","permissionDecisionReason":"Force-pushing rewrites remote history — a destructive action (AGENT-WORKFLOW §3). Only proceed if the user asked for this force-push in this session."}}'
  ;;
esac
exit 0
