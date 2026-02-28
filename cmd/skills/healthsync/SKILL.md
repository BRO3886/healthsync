---
name: healthsync
description: Queries Apple Health data stored in a local SQLite database. Use this skill to read heart rate, steps, SpO2, VO2 Max, sleep, workouts, resting heart rate, HRV, blood pressure, active/basal energy, body metrics, mobility, running metrics, mindful sessions, wrist temperature, and more. Can query via the healthsync CLI or directly via SQLite. Read-only — never write to the database.
metadata:
  author: sidv
  version: "1.1"
compatibility: Requires healthsync binary. Database at ~/.healthsync/healthsync.db must be populated via `healthsync parse`.
---

# healthsync — Apple Health Data Query Skill

## Installing healthsync

```bash
# macOS and Linux (recommended)
curl -fsSL https://healthsync.sidv.dev/install | bash

# Or via Go
go install github.com/BRO3886/healthsync@latest
```

After installing the binary, parse your Apple Health export:

```bash
# Export from Health app → profile picture → Export All Health Data
healthsync parse ~/Downloads/export.zip
```

Install this skill into your agent:

```bash
# Claude Code or Codex
healthsync skills install

# OpenClaw
healthsync skills install --agent openclaw
```

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

# Deduplicated daily step totals
healthsync query steps --total --from 2024-01-01

# Deduplicated daily active energy totals
healthsync query active-energy --total --from 2024-01-01

# Workouts as JSON
healthsync query workouts --format json --limit 20

# Sleep data as CSV
healthsync query sleep --format csv --limit 50

# Resting heart rate trend
healthsync query resting-heart-rate --limit 30

# HRV readings
healthsync query hrv --limit 30

# Blood pressure
healthsync query blood-pressure --limit 20

# Body weight trend
healthsync query body-mass --limit 30

# Direct SQLite for aggregations
sqlite3 ~/.healthsync/healthsync.db "SELECT date(start_date) as day, SUM(value) as total_steps FROM steps GROUP BY day ORDER BY day DESC LIMIT 7"

# Average resting heart rate per week
sqlite3 ~/.healthsync/healthsync.db "SELECT strftime('%Y-W%W', start_date) as week, ROUND(AVG(value),1) as avg_rhr FROM resting_heart_rate GROUP BY week ORDER BY week DESC LIMIT 12"
```

## CLI Reference

### `healthsync query <table>`

| Flag | Description | Default |
|------|-------------|---------|
| `--from` | Filter records from this date (inclusive) | — |
| `--to` | Filter records to this date (inclusive) | — |
| `--limit` | Maximum records to return | 50 |
| `--format` | Output format: `table`, `json`, `csv` | table |
| `--total` | Deduplicated daily totals (steps, active-energy, basal-energy only) | false |
| `--db` | Override database path | `~/.healthsync/healthsync.db` |

### Available Tables

**Cardiac**

| CLI Name | DB Table | Notes |
|----------|----------|-------|
| `heart-rate` | `heart_rate` | BPM; high-frequency |
| `resting-heart-rate` | `resting_heart_rate` | Daily RHR |
| `hrv` | `hrv` | HRV SDNN (ms); nightly |
| `heart-rate-recovery` | `heart_rate_recovery` | Post-exercise HR recovery |
| `respiratory-rate` | `respiratory_rate` | Breaths/min |
| `blood-pressure` | `blood_pressure` | Paired systolic + diastolic (mmHg) |

**Activity / Energy**

| CLI Name | DB Table | Notes |
|----------|----------|-------|
| `steps` | `steps` | Supports `--total` |
| `active-energy` | `active_energy` | kcal; supports `--total` |
| `basal-energy` | `basal_energy` | kcal; supports `--total` |
| `exercise-time` | `exercise_time` | Minutes |
| `stand-time` | `stand_time` | Minutes |
| `flights-climbed` | `flights_climbed` | Count |
| `distance-walking-running` | `distance_walking_running` | km/mi |
| `distance-cycling` | `distance_cycling` | km/mi |

**Body**

| CLI Name | DB Table | Notes |
|----------|----------|-------|
| `body-mass` | `body_mass` | kg/lb |
| `bmi` | `body_mass_index` | |
| `height` | `height` | m/ft |

**Mobility / Walking**

| CLI Name | DB Table | Notes |
|----------|----------|-------|
| `walking-speed` | `walking_speed` | m/s |
| `walking-step-length` | `walking_step_length` | m |
| `walking-asymmetry` | `walking_asymmetry` | % |
| `walking-double-support` | `walking_double_support` | % |
| `walking-steadiness` | `walking_steadiness` | Score |
| `stair-ascent-speed` | `stair_ascent_speed` | ft/s |
| `stair-descent-speed` | `stair_descent_speed` | ft/s |
| `six-minute-walk` | `six_minute_walk` | m |

**Running**

| CLI Name | DB Table | Notes |
|----------|----------|-------|
| `running-speed` | `running_speed` | m/s |
| `running-power` | `running_power` | W |
| `running-stride-length` | `running_stride_length` | m |
| `running-ground-contact-time` | `running_ground_contact_time` | ms |
| `running-vertical-oscillation` | `running_vertical_oscillation` | cm |

**Other**

| CLI Name | DB Table | Notes |
|----------|----------|-------|
| `spo2` | `spo2` | 0-1 fraction (0.98 = 98%) |
| `vo2max` | `vo2_max` | mL/min·kg |
| `sleep` | `sleep` | Sleep stages (category, no unit) |
| `workouts` | `workouts` | duration, distance, energy |
| `wrist-temperature` | `wrist_temperature` | °C deviation |
| `time-in-daylight` | `time_in_daylight` | Minutes |
| `dietary-water` | `dietary_water` | mL/L |
| `physical-effort` | `physical_effort` | MET score |
| `walking-heart-rate` | `walking_heart_rate` | BPM while walking |
| `mindful-sessions` | `mindful_sessions` | Category; no unit column |
| `stand-hours` | `stand_hours` | Category; no unit column |

### `healthsync parse <file>`

Parse an Apple Health export into the database. (Informational — do not run unless the user asks.)

| Flag | Description | Default |
|------|-------------|---------|
| `-v` | Verbose logging with progress rate | false |
| `--db` | Override database path | `~/.healthsync/healthsync.db` |

### `healthsync server`

Start HTTP server for receiving uploads. (Informational — do not start unless the user asks.)

**Endpoints:**
- `POST /api/upload` — Upload `.zip` or `.xml` (multipart form, field: `file`). Returns 202, parses async.
- `GET /api/upload/status` — Poll parse progress.
- `GET /api/health/{table}?from=&to=&limit=` — Query data as JSON.

## Database Schema

### Standard quantity tables

Schema: `id, source_name, start_date, end_date, value REAL, unit TEXT, created_at`

Applies to all tables except `blood_pressure`, `sleep`, `mindful_sessions`, `stand_hours`, and `workouts`.

```sql
CREATE TABLE resting_heart_rate (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date  TEXT NOT NULL,
    end_date    TEXT NOT NULL,
    value       REAL NOT NULL,
    unit        TEXT NOT NULL,
    created_at  TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
```

### blood_pressure (special — paired systolic + diastolic)

```sql
CREATE TABLE blood_pressure (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date  TEXT NOT NULL,
    end_date    TEXT NOT NULL,
    systolic    REAL NOT NULL,   -- mmHg
    diastolic   REAL NOT NULL,   -- mmHg
    unit        TEXT NOT NULL,   -- "mmHg"
    created_at  TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, systolic, diastolic)
);
```

### Category tables — no unit column

Applies to: `sleep`, `mindful_sessions`, `stand_hours`

```sql
CREATE TABLE sleep (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date  TEXT NOT NULL,
    end_date    TEXT NOT NULL,
    value       TEXT NOT NULL,   -- e.g. HKCategoryValueSleepAnalysisAsleepCore
    created_at  TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
```

### workouts

```sql
CREATE TABLE workouts (
    id                       INTEGER PRIMARY KEY AUTOINCREMENT,
    activity_type            TEXT NOT NULL,
    source_name              TEXT NOT NULL,
    start_date               TEXT NOT NULL,
    end_date                 TEXT NOT NULL,
    duration                 REAL,
    duration_unit            TEXT,
    total_distance           REAL,
    total_distance_unit      TEXT,
    total_energy_burned      REAL,
    total_energy_burned_unit TEXT,
    created_at               TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(activity_type, start_date, end_date, source_name)
);
```

### Date Format

All dates stored as text: `2024-01-15 08:30:00 +0530`. Filter with date prefix — `2024-01-01` works via SQLite string comparison.

### Sleep Stage Values

| Value | Meaning |
|-------|---------|
| `HKCategoryValueSleepAnalysisInBed` | In bed |
| `HKCategoryValueSleepAnalysisAsleepCore` | Core sleep |
| `HKCategoryValueSleepAnalysisAsleepDeep` | Deep sleep |
| `HKCategoryValueSleepAnalysisAsleepREM` | REM sleep |
| `HKCategoryValueSleepAnalysisAwake` | Awake |
| `HKCategoryValueSleepAnalysisAsleepUnspecified` | Unspecified |

## Common Query Patterns

### Daily step totals (deduped)
```bash
healthsync query steps --total --from 2024-01-01
```

### Daily active energy totals (deduped)
```bash
healthsync query active-energy --total --from 2024-01-01
```

### Average resting heart rate per week
```sql
SELECT strftime('%Y-W%W', start_date) as week,
  ROUND(AVG(value), 1) as avg_rhr
FROM resting_heart_rate
GROUP BY week ORDER BY week DESC LIMIT 12;
```

### HRV trend
```sql
SELECT date(start_date) as day, ROUND(AVG(value), 1) as hrv_ms
FROM hrv
GROUP BY day ORDER BY day DESC LIMIT 30;
```

### Blood pressure history
```sql
SELECT date(start_date) as day,
  ROUND(AVG(systolic), 1) as avg_sys,
  ROUND(AVG(diastolic), 1) as avg_dia
FROM blood_pressure
GROUP BY day ORDER BY day DESC LIMIT 30;
```

### Body weight trend
```sql
SELECT date(start_date) as day, value as kg
FROM body_mass
ORDER BY day DESC LIMIT 30;
```

### Sleep duration per night
```sql
SELECT date(start_date) as night,
  ROUND(SUM((julianday(end_date) - julianday(start_date)) * 24), 1) as hours
FROM sleep
WHERE value LIKE '%Asleep%'
GROUP BY night ORDER BY night DESC LIMIT 14;
```

### Average heart rate per day
```sql
SELECT date(start_date) as day,
  ROUND(AVG(value), 1) as avg_hr,
  MIN(value) as min_hr,
  MAX(value) as max_hr
FROM heart_rate
GROUP BY day ORDER BY day DESC LIMIT 30;
```

### Workout summary
```sql
SELECT activity_type, COUNT(*) as count,
  ROUND(AVG(duration), 1) as avg_min,
  ROUND(SUM(total_energy_burned)) as total_kcal
FROM workouts
GROUP BY activity_type ORDER BY count DESC;
```

### Weekly VO2 Max trend
```sql
SELECT strftime('%Y-W%W', start_date) as week,
  ROUND(AVG(value), 2) as avg_vo2
FROM vo2_max
GROUP BY week ORDER BY week DESC LIMIT 12;
```

### Mindfulness minutes per week
```sql
SELECT strftime('%Y-W%W', start_date) as week,
  ROUND(SUM((julianday(end_date) - julianday(start_date)) * 1440), 0) as minutes
FROM mindful_sessions
GROUP BY week ORDER BY week DESC LIMIT 12;
```

## Limitations

- **Read-only** — This skill must never write to the database
- **No real-time data** — Data is only as fresh as the last `healthsync parse` run
- **Date filtering is string-based** — Timezone offsets are part of the stored date string
- **SpO2 values are fractions** — 0.98 means 98%, not 98
- **Blood pressure is paired** — systolic and diastolic are stored together in one row per measurement
- **Category tables have no unit column** — `sleep`, `mindful_sessions`, `stand_hours` store text values, not numeric
