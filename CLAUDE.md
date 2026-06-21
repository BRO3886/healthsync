# healthsync

## Overview
CLI + HTTP server for syncing Apple Health export data into a local SQLite database. Parses `.zip` or `.xml` exports, stores in `~/.healthsync/healthsync.db`.

## Architecture
```
cmd/           — Cobra CLI commands (parse, query, server, skills, version)
  skills/      — embedded skill files (go:embed source)
    healthsync/  — skill prompt + version file for agents
  skills_embed.go — package cmd; //go:embed skills/healthsync
internal/
  parser/      — Streaming XML parser with DTD stripping, zip support
  storage/     — SQLite schema, batch inserts, query helpers
  server/      — Chi HTTP server with async upload
  skills/      — generic Install/Uninstall/DetectAgents logic
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

## Test Coverage (2026-02-25)
- `internal/parser` — ~90% (~30 tests; expanded for 40+ metric types, blood pressure, category types)
- `internal/storage` — ~88% (~55 tests; dedup, daily totals, all new tables)
- `internal/server` — 73.7% (17 tests)
- `internal/skills` — 8 tests (Install/Uninstall round-trip, idempotent reinstall, DetectAgents, etc.)
- `cmd/` — 13 cases (formatCommas, QueryTotalSupportedTables)
- Total: ~86 tests; ~88-90% on core packages

## Key Technical Details

### XML Parsing
- DTD must be stripped via `io.Pipe` goroutine (not `bufio.Scanner` + `MultiReader` — scanner consumes too many bytes)
- Must NOT call `decoder.Skip()` on `<HealthData>` root element — it skips all children
- Category types (sleep, mindful_sessions, stand_hours) are `HKCategoryType` (no unit attribute) — `RecordColumns()` returns 4 columns, not 5; value stored as TEXT
- Parser uses per-table `map[string][][]any` batch buffers — each table flushes at 1000 rows; all flush at EOF
- Blood pressure staging: systolic + diastolic keyed by `{sourceName, startDate}`; row emitted only when both are staged; unpaired records silently dropped
- Zip parsing is filename-agnostic: `findHealthExport` sniffs the first 32 KiB of each `.xml` entry for `<HealthData ` (trailing space guards against `<HealthDataArchive`). Handles localized filenames (e.g. `导出.xml` on Chinese-locale devices) without hardcoded locale lists. CDA sibling (`export_cda.xml`) is rejected naturally because its root is `<ClinicalDocument>`.
- Timestamps are normalized at parse time: `normalizeTimestamp` strips the trailing ` ±HHMM` offset so stored values are plain `YYYY-MM-DD HH:MM:SS` local time. Applied to all date fields (Record start/end, Workout start/end, blood pressure staging key). SQLite date funcs (`julianday`, `date`) cannot parse the space-separated offset format, so this is required for `--total` aggregations to work.

### SQLite
- WAL mode enabled for concurrent reads during server mode
- `INSERT OR IGNORE` with UNIQUE constraints for dedup
- Batch size: 1000 rows per transaction
- DB path: `~/.healthsync/healthsync.db` (override with `--db`)
- Schema variants: 5-col standard, 4-col no-unit (category types), 6-col blood_pressure, 10-col workouts

### Query
- `TableNameMap` maps both hyphen and underscore CLI names to DB table names (60+ entries)
- `--format table|json|csv` — CSV/JSON use `sortedKeys()` for deterministic column order
- `--total` routes to dedicated daily-total methods (NOT `QueryRows`) with overlap dedup. Supported: `steps`, `active-energy`, `basal-energy`, `sleep`
- `sleep --total` uses `date(start_date, '-6 hours')` to group by sleep night (pre-midnight sessions attributed to the correct night); filters `value LIKE '%Asleep%'` to exclude InBed and Awake; `--to` filter also gates on the shifted night date, not raw `start_date`
- Source-priority deduplication: Watch=2 > iPhone=1 > other=0 (uses `strings.Contains` on sourceName)

### Server
- `POST /api/upload` returns `202 Accepted`, parses async in goroutine
- `GET /api/upload/status` for polling progress (uses `sync/atomic` counters)
- Returns `409 Conflict` if a parse is already running

## Dependencies
- `github.com/spf13/cobra` — CLI
- `github.com/go-chi/chi/v5` — HTTP router
- `modernc.org/sqlite` — pure Go SQLite (no CGO)
- `github.com/jedib0t/go-pretty/v6` — table output (query command)
- `github.com/charmbracelet/huh` — interactive prompts (skills install agent picker)
- `github.com/fatih/color` — terminal color output

## Install Script
- `scripts/install.sh` — curl installer, supports macOS and Linux (arm64 + amd64)
- Copied verbatim to `website/static/install` (Hugo serves it at `/install`, no extension)
- Install: `curl -fsSL https://healthsync.sidv.dev/install | bash`
- Pattern: pre-flight OS/arch check → resolve latest GitHub release → download tar.gz → extract → install to `/usr/local/bin` (sudo if needed)
- No agent skill prompt in install script itself — skills are installed via `healthsync skills install`

## Skills Command (v0.3.0+)
- `healthsync skills install/uninstall/status` — installs agent skill prompt into `~/.claude/skills/healthsync/` (claude) or `~/.agents/skills/healthsync/` (codex)
- `--agent claude|codex|all` flag; interactive `huh.MultiSelect` when TTY; auto-detect when non-TTY
- Skills files embedded via `//go:embed skills/healthsync` in `cmd/skills_embed.go`
- **go:embed path constraint**: paths are relative to source file, no `..` — skills must live under `cmd/` (not root) because root is `package main`
- Version tracking via `.healthsync-version` file inside each installed skill dir
- Tests use `testing/fstest.MapFS` as fake embedded FS (no real files needed)

## Latest Release
- v0.5.1 (unreleased, on main) — strip tz offsets from stored timestamps, `sleep --total` with 6h night shift, localized zip support (Chinese etc. via content sniffing)
- v0.5.0 — `db info` subcommand, background update checker, OpenClaw skills target
- v0.4.0 — 40+ Apple Health metrics, multi-format query output (table/json/csv), --total flag
- v0.3.0 — skills install/uninstall/status command; 6 platform archives

## Release Process
Steps in order — do not skip or reorder:
1. `git push` — push all commits to main **first**. Never tag unpushed commits.
2. `git tag vX.Y.Z` — tag after push so the tag points to a commit already on remote main
3. `git push origin vX.Y.Z` — push the tag explicitly
4. `make release` — builds all 6 archives (`bin/*.tar.gz bin/*.zip`)
5. `gh release create vX.Y.Z bin/*.tar.gz bin/*.zip`

- **CRITICAL: Push before tag.** Tagging an unpushed commit then running `gh release create` pushes the tag + that commit but leaves `main` behind on remote — release binary is built from code not reachable from main.

## IndexNow

`make indexnow` submits the live sitemap URLs to IndexNow (Bing, Yandex, Naver, Seznam, Yep). Run it after any deploy that changes page content — not needed for binary-only releases.

The key lives in two places that must stay in sync:
- `website/static/09d76431580e356eafd8d91aeecc0906.txt` — the key file served at `https://healthsync.sidv.dev/<key>.txt` (filename stem == file contents)
- `KEY` variable in `scripts/indexnow.sh`

Live submission only works once the key file is deployed to Cloudflare Pages. Running the script before the key file is live will return HTTP 403 from the IndexNow endpoint — expected, not a bug.

## Conventions
- Conventional commits
- No mocks — tests use real temp SQLite databases
- Be proactive, not reactive — when given a task, just do it; don't ask for approval before starting
