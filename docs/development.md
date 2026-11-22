# Building and developing Chuck

Start from the [repository map](../MAP.md) and the
[feature and technical specifications](specs/README.md); they, not archived phase plans, define the
product. New ideas go in [ideas and improvements](ideas.md); specified changes approved to start
live in [ready changes](ready-changes/README.md). Live work state is in the
[handoff](features/HANDOFF.md).

## Prerequisites

- **Go 1.26.6** for the server and single binary (authoritative version: `go.mod`)
- **Node 20+ and npm** to build the UI. Node 22+ is required at runtime when source-installing the
  optional official Claude ACP adapter with `INSTALL_ACP=1`.
- macOS or Linux. The default terminal runtime is an embedded xterm.js/PTY bridge; tmux is
  optional and the iTerm2 driver is macOS-only.
- At least one authenticated agent backend. `install.sh` installs the pinned official Claude ACP
  adapter only when requested (`INSTALL_ACP=1`); chat launch needs the selected adapter on `PATH`
  and an installed Claude Code or Codex. Source builds have no Chuck bundle.
- `curl` and `jq` for the shell hooks used by terminal agents.

## Install from source

```sh
./install.sh            # build the UI + binary and install `chuck` on PATH
chuck dashboard start   # seeds ~/.chuck on first run, binds 127.0.0.1:4317
chuck dashboard open    # in another terminal
chuck dashboard stop
```

### Run without installing

```sh
make dist               # build UI, embed it, build ./bin/chuck
./bin/chuck --version
./bin/chuck dashboard start
```

### Live UI development

```sh
# Terminal 1: Go API with on-disk UI fallback
go run -tags dev ./cmd/chuck dashboard start

# Terminal 2: Vite dev server (proxies /api to :4317)
cd ui && npm ci && npm run dev   # http://localhost:5173
```

## Development tasks

```sh
make check-specs # validate the authoritative spec set
make test        # spec lint + both Go variants
make vet         # go vet ./...
make ui          # build the UI only
make dist        # full release build
make clean       # remove build artifacts
```

## Local HTTP API (`127.0.0.1:{port}`)

All routes are loopback-only. Browser API routes rely on the loopback/Host/Origin boundary; hook
and MCP producer routes use per-launch tokens. The full surface is in
[internal/server/routes.go](../internal/server/routes.go); the [operator CLI](operator-cli.md)
wraps it for scripting.

- **Health/state:** `GET /api/health` · `GET /api/sessions` · `GET /api/archive`
  · `GET /api/capabilities`
- **Session lifecycle:** `POST /api/sessions` (launch) · `GET /api/sessions/{id}`
  · `.../transcript` · `.../files` · `.../commands` · `.../messages` ·
  `POST .../prompt` · `.../cancel` · `.../stop` · `.../rename` · `.../identity`
  · `.../permission` · `.../resume` · `.../switch-runtime`
- **Config CRUD:** `GET/POST /api/roles`, `PUT/DELETE /api/roles/{role}` (same shape for
  `/api/projects`) · `GET/PUT /api/backends` · `GET/PUT /api/config` · `GET/PUT /api/layout`
- **Groups:** `POST /api/projects/{project}/groups/stop|archive` with `{group}`; actions affect
  only that project's members and return per-agent results.
- **Config federation:** `GET /api/config-sources` · `POST .../preview` ·
  `PUT .../{backend_id}` · `POST .../{backend_id}/refresh` · `DELETE .../{backend_id}`
- **Producers / live channels:** `POST /api/hook` (agent lifecycle, token-authed)
  · `GET /api/events` (SSE: `state_update`, `new_message`, `notification`,
  `config_source_update`, `ping`) · `GET /api/sessions/{id}/terminal/ws` (PTY↔WebSocket bridge)
  · `/mcp` (in-process MCP messaging server)
