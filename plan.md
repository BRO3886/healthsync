# Fix: Sleep queries — julianday() NULL on +0530 timestamps & post-midnight grouping

## Problem Summary (Issue #6)

Two bugs in sleep data handling:

1. **`julianday()` returns NULL for timezone-offset timestamps** — Apple Health exports timestamps like `2026-03-29 00:05:10 +0530`. SQLite's `julianday()` cannot parse the space-separated offset, so duration calculations silently return NULL/zero.

2. **`date(start_date)` misgroups post-midnight sleep sessions** — A session starting at 23:45 on March 29 gets grouped under March 29 instead of being attributed to the March 30 "sleep night".

## Root Cause

Timestamps are stored verbatim from Apple Health XML (e.g. `2024-01-01 22:00:00 +0530`). This format is not compatible with SQLite's built-in date functions, which expect either no timezone or `±HH:MM` format (not `±HHMM`).

## Plan

### Step 1: Strip timezone offsets during parsing (`internal/parser/xml.go`)

**What:** Add a `normalizeTimestamp(s string)` function that strips the trailing ` +NNNN` / ` -NNNN` timezone offset from Apple Health timestamps, converting to local time representation.

**Where:** `internal/parser/xml.go` — apply to `rec.StartDate` and `rec.EndDate` before building the row (line ~217). Also apply to `rec.StartDate` in the blood pressure key (line ~186) and workout dates (line ~263).

**Why local time, not UTC conversion:** Apple Health records times in the device's local timezone. The offset varies (travel, DST). Converting to UTC would make dates confusing (a 10 PM sleep session in +0530 becomes 4:30 PM UTC). Stripping the offset preserves the wall-clock time the user experienced, which is what matters for "what night did I sleep?" grouping. This matches the issue author's recommendation.

**Format:** `"2024-01-01 22:00:00 +0530"` → `"2024-01-01 22:00:00"` (plain ISO 8601 without offset)

This is safe because:
- All data comes from a single user's device
- Existing `--from`/`--to` string-comparison filtering already assumes no offset
- `julianday()` and `date()` will work correctly on the cleaned format
- The UNIQUE constraint still deduplicates correctly (same logical precision)

### Step 2: Add `QuerySleepDailyTotal` method (`internal/storage/queries.go`)

**What:** New method that computes nightly sleep duration totals, with proper night-boundary attribution.

```go
func (db *DB) QuerySleepDailyTotal(params QueryParams) ([]map[string]interface{}, error)
```

**SQL approach:**
```sql
SELECT date(start_date, '-6 hours') AS night,
       ROUND(SUM((julianday(end_date) - julianday(start_date)) * 24), 1) AS hours
FROM sleep
WHERE value LIKE '%Asleep%'
  [AND start_date >= ? AND start_date <= ?]
GROUP BY night
ORDER BY night DESC
```

**Key details:**
- `date(start_date, '-6 hours')` shifts the grouping boundary so that sessions starting between 6 PM and 5:59 AM are attributed to the same "night" (the calendar date of the evening). This is a standard sleep-tracking convention.
- Filters to only `Asleep` variants (Core, Deep, REM, Unspecified) — excludes InBed and Awake stages to report actual sleep time.
- Source-priority dedup is less critical for sleep (sessions don't overlap the same way steps do), but we can apply it if needed in a follow-up.

### Step 3: Wire up `--total` for sleep in `cmd/query.go`

**What:** Add `"sleep"` to the `totalSupportedTables` map and add a `case` in the switch to call `db.QuerySleepDailyTotal(params)`.

**Changes:**
- `cmd/query.go:52` — add `"sleep": true` to `totalSupportedTables`
- `cmd/query.go:74-83` — add case for `queryTotal && table == "sleep"`
- Update the `--total` flag description to include sleep

### Step 4: Update documentation (`website/content/docs/schema.md`)

**What:** Update the example sleep query in the docs to reflect that timestamps no longer have offsets (the existing example query will now "just work"). Note the `--total` support for sleep.

### Step 5: Add tests

**Files:**
- `internal/parser/xml_test.go` — test `normalizeTimestamp` with various offset formats (+0530, -0800, +0000, no offset)
- `internal/storage/storage_test.go` — test `QuerySleepDailyTotal`:
  - Basic duration calculation
  - Post-midnight session grouped to correct night
  - Filters out InBed/Awake values
  - Empty result set
- `cmd/parse_test.go` — update the unsupported-tables list to remove "sleep"

### Step 6: Update skill prompt if needed

Check `cmd/skills/healthsync/` for any sleep-related documentation that references the old timestamp format or lack of `--total` support.

## Files Changed

| File | Change |
|------|--------|
| `internal/parser/xml.go` | Add `normalizeTimestamp()`, apply to all date fields |
| `internal/parser/xml_test.go` | Tests for timestamp normalization |
| `internal/storage/queries.go` | Add `QuerySleepDailyTotal()` |
| `internal/storage/storage_test.go` | Tests for sleep daily totals |
| `cmd/query.go` | Wire `--total` for sleep |
| `cmd/parse_test.go` | Update unsupported table list |
| `website/content/docs/schema.md` | Update sleep query docs |

## Migration Note

Existing databases will still have old-format timestamps with offsets. The fix only affects **newly parsed** data. Users will need to re-parse their export (`healthsync parse`) to get clean timestamps. This is acceptable because:
- Parse is idempotent (INSERT OR IGNORE with UNIQUE constraints)
- Actually, existing rows won't be replaced due to INSERT OR IGNORE — so users need to delete and re-parse, or we add a `--force` / re-create option

**Recommended approach:** Document that a fresh parse is needed. A `healthsync parse --reparse` flag (drops + recreates tables) could be a follow-up enhancement, but is out of scope for this fix.
