# Configuration and files

Chuck keeps human-editable configuration as JSON files and machine state in one SQLite database,
all under `~/.chuck/`. You can edit everything in **Settings**, or edit the files directly.

## Layout (`~/.chuck/`)

```
roles/{role}.json     personas (seeded: default, chucky, implementer, reviewer, researcher)
projects/{p}.json     workspaces (none seeded; created in onboarding or Settings)
backends.json         providers + models (version 2)
config-sources.json   optional Claude/Codex native-config bindings
layout.json           card order + density
config.json           port, defaults (version 1)
state.db              agent identity, running registry, status, messages
sessions/{id}/        normalized transcript + session artifacts
cache/config-sources/ redacted, regenerable federation mirror data
cache/agent-skills/   product-managed operating skill, republished and verified at each start
```

## Environment variables

- `CHUCK_HOME` overrides `~/.chuck/` (used by tests and CI).
- `CHUCK_LOG_LEVEL` sets the log level: `debug`, `info` (default), `warn` or `error`.

## Logs

Every `chuck dashboard start` appends structured application logs to
`$CHUCK_HOME/dashboard.log`, including foreground starts; foreground logs are also shown in the
terminal. To collect the latest records for a bug report:

```sh
tail -n 500 "${CHUCK_HOME:-$HOME/.chuck}/dashboard.log" > chuck-dashboard.log
```

Review `chuck-dashboard.log` before attaching or sending it.

## Roles

Seeding is per-file and if-absent: new roles appear on the next `dashboard start`, and your edits
to existing ones are never overwritten. Edit them in Settings or directly in `roles/{role}.json`.

- **`default`**: no specialized persona (shown as No persona). New agents use it by default.
- **`chucky`**: the built-in Chuck expert and coordinator. Ask it how anything works (launch
  syntax, config files, switching runtimes, archive, messaging), or give it a goal: it can launch
  other agents through the `chuck` CLI and coordinate them over MCP messaging when the selected
  provider CLI passes the HTTP-MCP compatibility gate recorded in the specifications.
- **`implementer`**: completes a requested change within existing conventions and reports the
  verification it actually ran.
- **`reviewer`**: assesses work against its requirements and reports evidence-backed findings.
- **`researcher`**: investigates code, specifications and history, or external documentation,
  and reports sourced findings.

Every role, including one you write yourself, also receives a short Chuck operating context at
launch unless its persona turns off **Chuck aware agent**.

## Permissions

New installs skip permission prompts by default, and the No persona option and every seeded persona
inherit that setting. Change it in **Settings → Roles → Default permissions**, or override it per
persona. Saved settings are kept when you upgrade.
