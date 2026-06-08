# cctrack tailnet sync — design

## Goal

Aggregate Claude Code usage from every machine on the user's Tailscale
network into one cctrack instance, so token costs and the projected-spend
figures reflect the whole tailnet, not just the local machine.

Three requirements, in the user's words:

a. discover all devices on the tailnet
b. filter to the ones that have SSH available
c. grab the `.claude` data from each and fold it into the existing
   costs / projections

## Topology

This machine is the single aggregator ("hub"). It pulls from every
SSH-reachable peer; remote machines need no cctrack install — only their
JSONL files and Tailscale SSH access. Data flows one way: peer → hub.

## Data flow (one sync cycle)

```
tailscale status --json
        │  (discover + filter)
        ▼
  [peer, peer, …]   ← Online && len(sshHostKeys) > 0
        │  (pull, per-peer, isolated + timed out)
        ▼
~/.cctrack/hosts/<host>/projects/…   ← local mirror of remote ~/.claude/projects
        │  (parse, host-stamped, incremental via file_offsets)
        ▼
   sessions table (host column)
        │  (existing SUM queries, unchanged)
        ▼
   totals / projected / breakdowns
```

### 1. Discover + filter

Run `tailscale status --json` and walk the `Peer` map. Keep a peer when:

- `Online == true`, and
- `sshHostKeys` is present and non-empty.

`sshHostKeys` is Tailscale's structured signal that a peer has Tailscale SSH
enabled — verified against the live tailnet (mac + pi included, Windows box
excluded). This single field *is* requirement (b); no probe connections
needed. Each kept peer yields its `HostName` (used both as the SSH target and
the stored host label).

Config `include_hosts` / `exclude_hosts` (hostname lists) override the
automatic set when non-empty: `include_hosts` restricts to a whitelist;
`exclude_hosts` drops names from whatever set remains.

### 2. Pull

For each peer, mirror its remote `~/.claude/projects` to
`~/.cctrack/hosts/<HostName>/projects`, using the user's established idiom
(from `dotfiles/shared/bin/tmux-switch`): `tailscale ssh <HostName> -- <cmd>`
with **no `user@`** — Tailscale ACLs resolve the login user.

Primary mechanism — rsync for cheap incremental transfer:

```
rsync -az --timeout=<n> -e 'tailscale ssh' <host>:<remote_claude_dir>/ <mirror>/
```

Fallback (when rsync is unavailable or the `-e 'tailscale ssh'` invocation
fails on a host) — tar stream, the direct analogue of the switcher's form:

```
tailscale ssh <host> -- sh -c 'tar -C "$HOME" -cf - .claude/projects' | tar -C <mirror_parent> -xf -
```

(The remote shell resolves `$HOME`, so the hub never needs to know the remote
home path. `remote_claude_dir` config still drives the rsync source.)

Mirrors are **not** pruned (`--delete` off): if a remote rotates or deletes an
old session log, the hub keeps the mirrored copy so historical cost stays
counted.

Isolation: each peer runs under a bounded timeout (`ssh_timeout_seconds`); a
peer that is offline, slow, or errors logs a warning and is skipped — it never
aborts the rest of the sync. This mirrors the switcher's per-peer `timeout` +
parallel pattern.

### 3. Parse (host-stamped, incremental)

Run the existing parser over each peer's mirror dir, stamping the peer's
hostname onto every session it writes. `file_offsets` are keyed by the mirror
file path, so re-parsing a mirror only reads new bytes — identical to local
incremental behavior. The dash-encoded project name (`-home-pi-code-foo`) is
derived from the directory basename, so it survives the mirror relocation
unchanged.

The local machine's own `~/.claude/projects` is parsed as today, stamped with
`os.Hostname()`.

### 4. Aggregate

No query changes. Totals, projected spend, and existing breakdowns are
`SUM`/`GROUP BY` over the sessions table and pick up remote rows
automatically. Session IDs are UUIDs, so cross-host sessions never collide.

## Code changes

### New: `internal/tailnet`

- `Peer{ Host string }` and a `Discoverer` interface
  (`Discover() ([]Peer, error)`) with a real impl that shells out to
  `tailscale status --json` and filters as above. Pure filter logic
  (`filterPeers(status) []Peer`) is separated for testing against a JSON
  fixture.
- A `Puller` interface (`Pull(host, remoteDir, mirrorDir) error`) with a real
  impl (rsync, tar fallback) behind it.
- `Syncer` orchestrates discover → pull → parse for all peers, returns a
  per-host summary (`[]HostResult{ Host, FilesParsed, SessionsAffected, Err }`).
  Takes `Discoverer`, `Puller`, and `*parser.Parser` so tests inject fakes and
  never touch the network.

### `internal/parser`

- Add `ParseAllForHost(dir, host string) (files, sessions int, err error)`.
  Thread `host` down to where `SessionDelta` is built.
- `ParseAll(dir)` becomes `ParseAllForHost(dir, localHostname)`.
- `ParseFiles` (watcher path) keeps stamping the local hostname.

### `internal/store`

- Migration: idempotent `ALTER TABLE sessions ADD COLUMN host TEXT NOT NULL
  DEFAULT ''` (guarded — ignore "duplicate column" error).
- `SessionDelta` gains `Host`; `UpsertSession` writes it (set on insert; on
  conflict, overwrite when non-empty, same idiom as `slug`/`model`).
- `Session` struct gains `Host` (`json:"host"`); all `SELECT`/`Scan` sites
  that build a `Session` include the column.
- `GetHostBreakdown() ([]HostSummary, error)` — `GROUP BY host`, mirroring
  `GetModelBreakdown` (session_count, total_cost, total_tokens, last_activity).

### `internal/config`

New nested block:

```json
"tailnet": {
  "enabled": false,
  "sync_interval_minutes": 5,
  "remote_claude_dir": ".claude/projects",
  "include_hosts": [],
  "exclude_hosts": [],
  "ssh_timeout_seconds": 20
}
```

`enabled` defaults false so existing single-machine users are unaffected;
`cctrack sync` and the serve ticker no-op when disabled. `remote_claude_dir`
is relative to the remote home.

### Commands / serve

- New `cmd/sync.go` — `cctrack sync`: load config, open store, build a real
  `Syncer`, run it once, print a per-host summary table. If `tailscale` is
  missing or `tailnet.enabled` is false, print a clear message and exit 0
  (local tracking is independent).
- `cmd/serve.go` — when `tailnet.enabled`, start a goroutine ticking every
  `sync_interval_minutes` that runs the same `Syncer` and, on changes,
  broadcasts the existing `summary.updated` (and per-session) WebSocket events
  so the dashboard stays live. Runs one sync shortly after startup too.

### API + dashboard ("By host" card)

- `GET /api/v1/hosts` → `GetHostBreakdown()`, mirroring `/api/v1/models`.
- Frontend: a "By host" breakdown card on the Overview view, modeled on the
  existing `ModelBreakdown` component (host label, cost, tokens, share). Added
  last; purely additive.

## Error handling

- `tailscale` binary absent → discovery returns a typed "tailscale
  unavailable" result; sync is a no-op with a clear log line. Local tracking
  unaffected.
- Per-peer failures (offline, SSH denied, timeout, tar/rsync error) are
  collected into that peer's `HostResult.Err` and reported; other peers
  proceed.
- Malformed JSONL is already skipped by the parser.
- Mirror directory creation failure for one host is that host's error only.

## Testing

- `tailnet.filterPeers` — unit test against a captured `tailscale status
  --json` fixture: online+sshHostKeys included; offline excluded; Windows (no
  sshHostKeys) excluded; include/exclude overrides honored. Pure, no network.
- `Syncer` — fake `Discoverer` (returns fixed peers) + fake `Puller`
  (populates a temp mirror dir from testdata, or returns an error for a chosen
  host) + real parser/store on a temp DB. Asserts: per-peer isolation (one
  puller error doesn't sink the others), correct mirror paths, host stamping
  in the DB, incremental re-parse (second run adds no duplicate cost).
- `parser.ParseAllForHost` — parses a testdata dir and asserts the `host`
  column is set on the resulting sessions.
- `store` migration — opening an old DB (no host column) then a new one is
  idempotent; `GetHostBreakdown` groups correctly.

## Out of scope (YAGNI)

- Pushing local data to a remote hub (one-way pull only).
- Per-host login users / `user@` (Tailscale ACLs handle it).
- Windows peers (no Tailscale SSH; filtered out automatically).
- `host` on the per-request table (sessions carry host; requests join if ever
  needed).
- Pruning mirrors / deleting historical remote data.
