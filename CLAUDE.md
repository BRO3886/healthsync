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
```

## Build & Test
```bash
go build -o healthsync .
go test ./... -v
go test ./... -coverprofile=coverage.out && go tool cover -func=coverage.out
```

## Test Coverage (2026-02-12)
- `internal/parser` — 86.9% (17 tests)
- `internal/storage` — 85.5% (29 tests)
- `internal/server` — 73.7% (17 tests)
- `cmd/` — 0% (thin cobra wiring, no unit tests)
- Total: 53 tests, 59.5% overall (82-87% on core packages)

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
- `github.com/charmbracelet/huh` — interactive prompts (not yet used)

## Conventions
- Conventional commits
- No mocks — tests use real temp SQLite databases
