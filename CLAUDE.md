# Chuck — Claude Code

Follow the shared repository instructions in @AGENTS.md and the selected skill. The SessionStart hook injects the handoff header; count that as the startup read.

Edit and Write hooks check generated paths, formatting, invariants, skill twins and spec mechanics. Apply the shared completion checks even when hooks pass. Keep `.gocache/` unstaged. When tracing old commits through the historical rehash, deduplicate by subject as well as SHA.
