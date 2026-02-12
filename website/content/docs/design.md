---
title: "Design"
description: "Architecture and design decisions behind healthsync."
date: 2026-02-12T00:00:00+05:30
draft: false
weight: 600
toc: true
---

## Architecture

```
healthsync/
├── cmd/                   # Cobra CLI commands
│   ├── root.go            # Root command, --db flag
│   ├── parse.go           # Parse command with verbose logging
│   ├── query.go           # Query with format options
│   └── server.go          # HTTP server command
├── internal/
│   ├── parser/            # Streaming XML parser
│   │   ├── types.go       # Record/Workout structs
│   │   └── xml.go         # DTD stripping, XML decode, zip support
│   ├── storage/           # SQLite layer
│   │   ├── sqlite.go      # DB init, schema, batch insert
│   │   └── queries.go     # Query helpers, table name mapping
│   └── server/            # HTTP server
│       ├── server.go      # Chi router, graceful shutdown
│       └── handlers.go    # Upload, status, query endpoints
└── main.go
```

## Streaming XML parser

Apple Health exports can be 950MB+. The parser uses constant memory (~10MB) by:

1. **DTD stripping** — Apple's XML includes a DTD section that Go's `xml.Decoder` can't handle. We strip it using an `io.Pipe` goroutine that filters lines before the decoder sees them.

2. **Token-based parsing** — `xml.NewDecoder` + `Token()` loop. Only calls `DecodeElement()` for `<Record>` and `<Workout>` start elements.

3. **Type filtering** — Checks the `type` attribute before inserting. Skips irrelevant record types (DietaryWater, BodyMass, etc.) without allocating.

4. **Zip streaming** — Opens the zip, finds `export.xml`, and streams directly from the zip reader. No extraction to disk needed.

## Batch inserts

Records are buffered in memory (1000 per batch) and inserted in a single transaction:

```sql
INSERT OR IGNORE INTO heart_rate (source_name, start_date, end_date, value, unit)
VALUES (?,?,?,?,?), (?,?,?,?,?), ...
```

The `OR IGNORE` clause combined with `UNIQUE` constraints makes re-imports idempotent — running `healthsync parse` on the same file twice inserts 0 new rows.

## Async uploads

The HTTP server returns `202 Accepted` immediately and parses in a background goroutine. This prevents request timeouts on large files (~30s parse time).

- Progress is tracked via `sync/atomic` counters
- Status is polled via `GET /api/upload/status`
- Only one parse job can run at a time (returns `409 Conflict` if busy)

## SQLite configuration

- **WAL mode** — Allows concurrent reads during server mode
- **`synchronous=NORMAL`** — Good performance with WAL
- **Pure Go driver** — `modernc.org/sqlite` requires no CGO, simplifying cross-compilation

## Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/spf13/cobra` | CLI framework |
| `github.com/go-chi/chi/v5` | HTTP router |
| `modernc.org/sqlite` | Pure Go SQLite driver |
