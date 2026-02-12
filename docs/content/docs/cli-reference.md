---
title: "CLI Reference"
description: "Complete reference for all healthsync commands and flags."
summary: ""
date: 2026-02-12T00:00:00+05:30
lastmod: 2026-02-12T00:00:00+05:30
draft: false
weight: 200
toc: true
seo:
  title: "CLI Reference — healthsync"
  description: "Complete reference for healthsync parse, query, and server commands."
  canonical: ""
  noindex: false
---

## `healthsync parse`

Parse an Apple Health export file into the SQLite database.

```bash
healthsync parse <file.zip|export.xml> [flags]
```

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--verbose` | `-v` | Enable verbose logging with progress rate | `false` |
| `--db` | | Path to SQLite database | `~/.healthsync/healthsync.db` |

**Examples:**

```bash
# Parse a zip export
healthsync parse export.zip

# Parse raw XML
healthsync parse export.xml

# Verbose mode with custom DB
healthsync parse export.zip -v --db ./test.db
```

**Behavior:**
- Accepts `.zip` (auto-extracts `export.xml`) or raw `.xml`
- Streams XML with constant memory (~10MB for 950MB files)
- Uses `INSERT OR IGNORE` for deduplication — safe to re-run
- Inserts in batches of 1000 rows per transaction

---

## `healthsync query`

Query health data from the local database.

```bash
healthsync query <table> [flags]
```

| Flag | Description | Default |
|------|-------------|---------|
| `--from` | Filter records from this date (inclusive) | — |
| `--to` | Filter records to this date (inclusive) | — |
| `--limit` | Maximum records to return | `50` |
| `--format` | Output format: `table`, `json`, `csv` | `table` |
| `--db` | Path to SQLite database | `~/.healthsync/healthsync.db` |

**Available tables:**

| CLI Name | DB Table | Data |
|----------|----------|------|
| `heart-rate` | `heart_rate` | BPM readings |
| `steps` | `steps` | Step counts |
| `spo2` | `spo2` | Blood oxygen (0-1 fraction) |
| `vo2max` | `vo2_max` | VO2 Max (mL/min·kg) |
| `sleep` | `sleep` | Sleep stages |
| `workouts` | `workouts` | All workout types |

**Examples:**

```bash
# Table output (default)
healthsync query heart-rate --limit 10

# JSON output
healthsync query workouts --format json --limit 5

# CSV with date range
healthsync query steps --from 2024-01-01 --to 2024-12-31 --format csv

# No limit (return all)
healthsync query spo2 --limit 0
```

---

## `healthsync server`

Start an HTTP server for receiving health data uploads and serving queries.

```bash
healthsync server [flags]
```

| Flag | Description | Default |
|------|-------------|---------|
| `--port` | Port to listen on | `8080` |
| `--host` | Host to bind to | `0.0.0.0` |
| `--db` | Path to SQLite database | `~/.healthsync/healthsync.db` |

**Examples:**

```bash
# Start with defaults
healthsync server

# Custom port and host
healthsync server --port 9090 --host 127.0.0.1
```

See the [Server API](/docs/server-api/) page for endpoint documentation.
