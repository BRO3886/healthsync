---
title: "Agent Skills"
description: "Let your AI coding agent query your Apple Health data directly."
date: 2026-02-25T00:00:00+05:30
draft: false
weight: 250
toc: true
---

healthsync ships with an agent skill — a set of documentation files that teach AI coding agents how to query your Apple Health data. Once installed, your agent can answer health questions directly in conversation by running CLI commands or SQL queries against your local database.

## Why this exists

The primary motivation for building healthsync was agent-queryable health data. Apple Health has a great export, but the raw XML is unwieldy. healthsync bridges the gap: parse once, query forever — from the CLI, from SQL, or from a conversation with your AI agent.

## Installing the skill

```bash
healthsync skills install
```

Running this command shows an interactive picker to select which agents to install for. Use `--agent` to skip the picker:

```bash
# Claude Code
healthsync skills install --agent claude

# Codex CLI
healthsync skills install --agent codex

# All supported agents
healthsync skills install --agent all
```

**Install destinations:**

| Agent | Skill directory |
|-------|----------------|
| `claude` | `~/.claude/skills/healthsync/` |
| `codex` | `~/.agents/skills/healthsync/` |

The skill is automatically detected and loaded by the agent on the next session start — no configuration needed.

## What the skill teaches

The installed skill includes:

- **Database schema** — table definitions for all 6 metrics (heart rate, steps, SpO2, VO2 Max, sleep, workouts)
- **CLI reference** — all `healthsync query` flags and available tables
- **SQL query examples** — daily step totals, heart rate averages, sleep duration, workout summaries, VO2 Max trends
- **Date format** — how Apple Health timestamps are stored (for correct `--from/--to` filtering)

## Example agent interactions

Once the skill is installed, your agent can run commands like:

```bash
# Agent runs this to answer "what was my average heart rate last week?"
healthsync query heart-rate --from 2026-02-18 --to 2026-02-25 --format json

# Agent runs this to answer "how many steps did I take each day this month?"
healthsync query steps --total --from 2026-02-01 --format json

# Agent runs this to answer "show me my recent workouts"
healthsync query workouts --limit 10 --format json
```

Or query the SQLite database directly for more complex analysis:

```bash
sqlite3 ~/.healthsync/healthsync.db "
  SELECT date(start_date) as day, ROUND(AVG(value), 1) as avg_bpm
  FROM heart_rate
  WHERE start_date >= '2026-01-01'
  GROUP BY day
  ORDER BY day DESC
  LIMIT 30
"
```

## Checking status

```bash
healthsync skills status
```

```
healthsync skill (binary v0.3.0):
  ✓ claude       ~/.claude/skills/healthsync/ (installed v0.3.0)
  ✗ codex        ~/.agents/skills/healthsync/ (not installed)
```

The status command shows whether the installed version matches the current binary. If outdated, re-run `healthsync skills install` to update.

## Uninstalling

```bash
healthsync skills uninstall --agent claude
```

This removes the skill directory. The agent will no longer have access to healthsync documentation on next session start.

## HTTP server for remote agents

If your agent runs on a different machine, you can expose health data over HTTP using `healthsync server`:

```bash
healthsync server --port 8080
```

The agent can then query via the HTTP API:

```bash
curl "http://localhost:8080/api/health/heart-rate?limit=10"
curl "http://localhost:8080/api/health/steps?from=2026-01-01&format=json"
```

See the [Server API](/docs/server-api/) page for full endpoint documentation.
