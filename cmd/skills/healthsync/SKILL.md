---
name: healthsync
description: Queries Apple Health data stored in a local SQLite database. Use this skill to read heart rate, steps, SpO2, VO2 Max, sleep, and workout data. Can query via the healthsync CLI or directly via SQLite. Read-only — never write to the database.
metadata:
  author: sidv
  version: "1.0"
compatibility: Requires healthsync binary built from source. Database at ~/.healthsync/healthsync.db must be populated via `healthsync parse`.
---

# healthsync — Apple Health Data Query Skill

Query Apple Health export data stored in a local SQLite database. This skill is **read-only** — never INSERT, UPDATE, DELETE, or DROP anything.

## Important Constraints

- **READ ONLY** — You must NEVER write to the database. No INSERT, UPDATE, DELETE, DROP, ALTER, or any write operations.
- **Two query methods**: CLI (`healthsync query`) or direct SQLite (`sqlite3 ~/.healthsync/healthsync.db`)
- **Prefer CLI** for simple queries. Use direct SQLite for complex aggregations, joins, or custom SQL.

## Database Location

Default: `~/.healthsync/healthsync.db`

## Quick Start

```bash
# Recent heart rate readings
healthsync query heart-rate --limit 10

# Steps in a date range
healthsync query steps --from 2024-01-01 --to 2024-06-30 --limit 100

# Workouts as JSON
healthsync query workouts --format json --limit 20

# Sleep data as CSV
healthsync query sleep --format csv --limit 50

# Direct SQLite for aggregations
sqlite3 ~/.healthsync/healthsync.db "SELECT date(start_date) as day, SUM(value) as total_steps FROM steps GROUP BY day ORDER BY day DESC LIMIT 7"

# Average resting heart rate per day
sqlite3 ~/.healthsync/healthsync.db "SELECT date(start_date) as day, ROUND(AVG(value),1) as avg_hr FROM heart_rate GROUP BY day ORDER BY day DESC LIMIT 30"
```

## CLI Reference

### `healthsync query <table>`

Query health data from the database.

| Flag | Description | Default |
|------|-------------|---------|
| `--from` | Filter records from this date (inclusive) | — |
| `--to` | Filter records to this date (inclusive) | — |
| `--limit` | Maximum records to return | 50 |
| `--format` | Output format: `table`, `json`, `csv` | table |
| `--db` | Override database path | `~/.healthsync/healthsync.db` |

**Available tables:** `heart-rate`, `steps`, `spo2`, `vo2max`, `sleep`, `workouts`

### `healthsync parse <file>`

Parse an Apple Health export into the database. (Informational — do not run unless the user asks.)

| Flag | Description | Default |
|------|-------------|---------|
| `-v` | Verbose logging with progress rate | false |
| `--db` | Override database path | `~/.healthsync/healthsync.db` |

### `healthsync server`

Start HTTP server for receiving uploads. (Informational — do not start unless the user asks.)

| Flag | Description | Default |
|------|-------------|---------|
| `--port` | Port to listen on | 8080 |
| `--host` | Host to bind to | 0.0.0.0 |
| `--db` | Override database path | `~/.healthsync/healthsync.db` |

**Endpoints:**
- `POST /api/upload` — Upload `.zip` or `.xml` (multipart form, field: `file`). Returns 202, parses async.
- `GET /api/upload/status` — Poll parse progress.
- `GET /api/health/{table}?from=&to=&limit=` — Query data as JSON.

## Database Schema

See [references/schema.md](references/schema.md) for full table definitions.

### Tables Overview

| Table | CLI Name | Key Columns | Notes |
|-------|----------|-------------|-------|
| `heart_rate` | `heart-rate` | source_name, start_date, end_date, value (BPM), unit | |
| `steps` | `steps` | source_name, start_date, end_date, value (count), unit | |
| `spo2` | `spo2` | source_name, start_date, end_date, value (0-1 fraction), unit | Value is 0.xx not percentage |
| `vo2_max` | `vo2max` | source_name, start_date, end_date, value, unit (mL/min·kg) | |
| `sleep` | `sleep` | source_name, start_date, end_date, value (sleep stage) | No unit column |
| `workouts` | `workouts` | activity_type, source_name, start_date, end_date, duration, distance, energy | Nullable distance/energy |

### Date Format

All dates are stored as text in Apple Health format: `2024-01-15 08:30:00 +0530`

When filtering with `--from` / `--to`, use date prefixes: `2024-01-01` works because SQLite does string comparison.

### Sleep Stage Values

| Value | Meaning |
|-------|---------|
| `HKCategoryValueSleepAnalysisInBed` | In bed |
| `HKCategoryValueSleepAnalysisAsleepCore` | Core sleep |
| `HKCategoryValueSleepAnalysisAsleepDeep` | Deep sleep |
| `HKCategoryValueSleepAnalysisAsleepREM` | REM sleep |
| `HKCategoryValueSleepAnalysisAwake` | Awake |
| `HKCategoryValueSleepAnalysisAsleepUnspecified` | Unspecified |

### Workout Activity Types

Prefixed with `HKWorkoutActivityType`. Common values: `Running`, `Walking`, `Cycling`, `Yoga`, `Swimming`, `HighIntensityIntervalTraining`, `TraditionalStrengthTraining`, etc.

## Common Query Patterns

### Daily step totals
```sql
SELECT date(start_date) as day, ROUND(SUM(value)) as total_steps
FROM steps
GROUP BY day
ORDER BY day DESC
LIMIT 7;
```

### Average heart rate per day
```sql
SELECT date(start_date) as day, ROUND(AVG(value), 1) as avg_hr, MIN(value) as min_hr, MAX(value) as max_hr
FROM heart_rate
GROUP BY day
ORDER BY day DESC
LIMIT 30;
```

### Sleep duration per night
```sql
SELECT date(start_date) as night,
  ROUND(SUM((julianday(end_date) - julianday(start_date)) * 24), 1) as hours
FROM sleep
WHERE value LIKE '%Asleep%'
GROUP BY night
ORDER BY night DESC
LIMIT 14;
```

### Workout summary
```sql
SELECT activity_type, COUNT(*) as count,
  ROUND(AVG(duration), 1) as avg_min,
  ROUND(SUM(total_energy_burned)) as total_kcal
FROM workouts
GROUP BY activity_type
ORDER BY count DESC;
```

### Weekly VO2 Max trend
```sql
SELECT strftime('%Y-W%W', start_date) as week,
  ROUND(AVG(value), 2) as avg_vo2
FROM vo2_max
GROUP BY week
ORDER BY week DESC
LIMIT 12;
```

## Limitations

- **Read-only** — This skill must never write to the database
- **No real-time data** — Data is only as fresh as the last `healthsync parse` run
- **Date filtering is string-based** — Timezone offsets are part of the stored date string
- **SpO2 values are fractions** — 0.98 means 98%, not 98
