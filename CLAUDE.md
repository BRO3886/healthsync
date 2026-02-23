# healthsync

## Overview
CLI + HTTP server for syncing Apple Health export data into a local SQLite database. Parses `.zip` or `.xml` exports, stores in `~/.healthsync/healthsync.db`.

## Architecture
```
cmd/           — Cobra CLI commands (parse, query, server)
internal/
  parser/      — Streaming XML parser with DTD stripping, zip support
  storage/     — SQLite schema, batch inserts, query helpers
  server/      — Chi HTTP server with async upload
scripts/       — install.sh (served at healthsync.sidv.dev/install)
website/       — Hugo site (custom theme, no npm)
  static/install — copy of scripts/install.sh, served at /install
```

## Build & Test
```bash
make build          # outputs bin/healthsync
make test           # run all tests
make release        # darwin/linux tar.gz + windows zip into bin/
go test ./... -coverprofile=coverage.out && go tool cover -func=coverage.out
```

## Test Coverage (2026-02-24)
- `internal/parser` — 86.9% (17 tests)
- `internal/storage` — 85.5% (29 tests)
- `internal/server` — 73.7% (17 tests)
- `cmd/` — first tests added (formatCommas, 11 cases)
- Total: 54 tests, 59.5% overall (82-87% on core packages)

## Key Technical Details

### XML Parsing
- DTD must be stripped via `io.Pipe` goroutine (not `bufio.Scanner` + `MultiReader` — scanner consumes too many bytes)
- Must NOT call `decoder.Skip()` on `<HealthData>` root element — it skips all children
- Sleep records are `HKCategoryType` (no unit attribute) — `RecordColumns("sleep")` returns 4 columns, not 5

### SQLite
- WAL mode enabled for concurrent reads during server mode
- `INSERT OR IGNORE` with UNIQUE constraints for dedup
- Batch size: 1000 rows per transaction
- DB path: `~/.healthsync/healthsync.db` (override with `--db`)

### Server
- `POST /api/upload` returns `202 Accepted`, parses async in goroutine
- `GET /api/upload/status` for polling progress (uses `sync/atomic` counters)
- Returns `409 Conflict` if a parse is already running

## Dependencies
- `github.com/spf13/cobra` — CLI
- `github.com/go-chi/chi/v5` — HTTP router
- `modernc.org/sqlite` — pure Go SQLite (no CGO)
- `github.com/jedib0t/go-pretty/v6` — table output (query command)
- `github.com/charmbracelet/huh` — interactive prompts (not yet used)

## Install Script
- `scripts/install.sh` — curl installer, supports macOS and Linux (arm64 + amd64)
- Copied verbatim to `website/static/install` (Hugo serves it at `/install`, no extension)
- No agent skill prompt (healthsync has no `skills` subcommand)
- Pattern: pre-flight OS/arch check → resolve latest GitHub release → download tar.gz → extract → install to `/usr/local/bin` (sudo if needed)

## Conventions
- Conventional commits
- No mocks — tests use real temp SQLite databases
