# cctrack

A cost tracker for [Claude Code](https://docs.anthropic.com/en/docs/claude-code). Parses your local JSONL logs, calculates spend per session/project/model, and serves a real-time dashboard — all from a single binary.

## Features

- **Cost tracking** — today, this week, this month, and projected monthly spend
- **Session explorer** — browse every Claude Code session with token and cost breakdowns
- **Project breakdown** — see spend grouped by project, with monthly trends
- **Model breakdown** — usage and cost per model (Opus, Sonnet, Haiku)
- **Activity heatmap** — visualize when you're using Claude Code most
- **Request timeline** — per-request token usage within each session
- **Real-time updates** — file watcher + WebSocket push when new activity is detected
- **Budget tracking** — set a monthly budget and see progress against it
- **Tailnet sync** — aggregate usage from every machine on your Tailscale network into one dashboard
- **Single binary** — Go CLI with an embedded Vue 3 SPA, no separate frontend server needed

## Installation

### From source

Requires Go 1.22+ and Node.js (to build the embedded dashboard).

```bash
git clone https://github.com/nedlane/cctrack.git
cd cctrack
cd web && npm install && npm run build && cd ..
go build -o cctrack .
```

The dashboard is embedded into the binary via `go:embed`, so `web/dist` must be
built (the `npm run build` step above) before `go build`.

## Usage

### Start the dashboard

```bash
cctrack serve
```

Opens a web dashboard on `http://localhost:7432` with real-time cost tracking. Parses logs on startup and watches for new activity.

### Parse logs manually

```bash
cctrack parse
```

Scans `~/.claude/projects/` for JSONL log files and updates the SQLite database.

### Quick status

```bash
cctrack status
```

Prints today/week/month spend and your most expensive session to stdout.

### View configuration

```bash
cctrack config
```

### Sync usage across your Tailscale network

```bash
cctrack sync
```

Discovers every SSH-reachable machine on your tailnet, mirrors each one's
`~/.claude/projects` logs locally, and folds their token usage into the same
totals and projections. One machine acts as the aggregator; the others need no
cctrack install — only Tailscale SSH access and their log files.

How it finds and reaches peers:

- **Discovery** — reads `tailscale status --json` and keeps peers that are
  online and advertise Tailscale SSH (`sshHostKeys`). Windows boxes and any
  peer without Tailscale SSH are skipped automatically.
- **Transport** — pulls over `tailscale ssh <host>` (no usernames; Tailscale
  ACLs resolve the login user), using `rsync` for incremental transfer with a
  `tar`-stream fallback.
- **Attribution** — each session is stamped with its host, so the dashboard
  shows a **Spend by Host** breakdown once more than one machine reports.

Enable it in config (or the dashboard settings) under the `tailnet` block.
When `tailnet.enabled` is true, `cctrack serve` also re-syncs automatically on
an interval. Mirrored logs are cached under `~/.cctrack/hosts/<host>/projects`.

## How it works

1. Claude Code writes JSONL logs to `~/.claude/projects/<project>/<session>.jsonl`
2. cctrack scans these files, extracts token usage from `assistant` messages, and deduplicates by `requestId`
3. Costs are calculated using Anthropic's published per-token rates for each model
4. Data is stored in a local SQLite database (`~/.cctrack/cctrack.db`)
5. The `serve` command starts an HTTP server with a REST API and embedded Vue SPA
6. A file watcher detects new log activity and pushes updates via WebSocket

## Configuration

Config is stored at `~/.cctrack/config.json`:

```json
{
  "log_dir": "~/.claude/projects",
  "db_path": "~/.cctrack/cctrack.db",
  "port": 7432,
  "monthly_budget_usd": 0,
  "open_browser_on_serve": true,
  "tailnet": {
    "enabled": false,
    "sync_interval_minutes": 5,
    "remote_claude_dir": ".claude/projects",
    "include_hosts": [],
    "exclude_hosts": [],
    "ssh_timeout_seconds": 20
  }
}
```

The `tailnet` block controls cross-machine sync (see *Sync usage across your
Tailscale network* above). It's disabled by default. `include_hosts` /
`exclude_hosts` take Tailscale short hostnames; leave `include_hosts` empty to
sync every SSH-reachable peer.

All settings can also be changed from the dashboard's settings page.

## License

[MIT](LICENSE)
