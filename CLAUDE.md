# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build

The binary embeds the Vue SPA via `go:embed`, so the frontend must be built first:

```bash
cd web && npm install && npm run build && cd ..
go build -o cctrack .
```

For frontend-only changes during development:
```bash
cd web && npm run dev   # Vite dev server on :5173 (does not use the Go backend)
```

## Tests

```bash
go test ./...                        # all tests
go test ./internal/store/...         # store tests only
go test ./internal/tailnet/...       # tailnet tests only
go test -run TestHostColumnRoundTrip ./internal/store/...  # single test
```

Tests use `t.TempDir()` for scratch databases — no fixtures needed.

## Architecture

**Data flow:**
1. Claude Code writes JSONL to `~/.claude/projects/<project>/<session>.jsonl`
2. `internal/parser` scans those files, deduplicates by `requestId`, and upserts into SQLite via `internal/store`
3. `internal/calculator` maps model-name prefixes to per-token rates and computes cost
4. `cmd/serve` wires store + hub + watcher + API; file watcher pushes WebSocket events to connected clients
5. Embedded Vue SPA (`web/dist`) is served at `/`; REST API at `/api/v1/`

**Key packages:**

| Package | Responsibility |
|---|---|
| `internal/store` | SQLite via `modernc.org/sqlite` — schema in `migrate()`, queries split across `queries.go`/`sessions.go`/`offsets.go` |
| `internal/parser` | JSONL file discovery, incremental parsing with file offsets, host attribution |
| `internal/calculator` | Token-rate table and cost calculation; rates in `rates.go` |
| `internal/tailnet` | Tailscale peer discovery, rsync/tar pull, sync orchestration |
| `internal/hub` | In-process WebSocket broadcast hub |
| `internal/watcher` | `fsnotify`-based file watcher with debounce |
| `internal/api` | HTTP handlers; SPA fallback in `spa.go` |
| `cmd/` | Cobra sub-commands (`serve`, `parse`, `sync`, `status`, `config`, `version`) |

**Database schema** (`~/.cctrack/cctrack.db`):
- `sessions` — one row per Claude Code session; has `host` column for multi-machine attribution (added as an additive migration)
- `requests` — one row per API request (`requestId`); foreign key to `sessions`
- `file_offsets` — byte offsets so the parser only reads new data on subsequent runs

**Frontend** (`web/`): Vue 3 + Vite + Pinia + TanStack Query + Chart.js. Views are in `src/views/`, reusable chart components in `src/components/charts/`. WebSocket connection is managed in `src/composables/useRealtimeUpdates.ts`.

**Tailnet sync** (`cctrack sync`): discovers SSH-reachable Tailscale peers, mirrors their `~/.claude/projects` logs to `~/.cctrack/hosts/<host>/projects`, then calls `parser.ParseAllForHost` to ingest them. Sessions are stamped with the source hostname, enabling per-host breakdown in the dashboard.

## Configuration

Config lives at `~/.cctrack/config.json`. `internal/config/config.go` owns the struct and `Save()`/`Load()` helpers. Settings can also be updated via `POST /api/v1/settings`.

## Model rates

When Anthropic releases new models, add an entry to `internal/calculator/rates.go`. The `GetRates` function matches by model-name prefix (e.g. `"claude-sonnet-4"` matches `"claude-sonnet-4-5-20251001"`).
