---
title: "Database Schema"
description: "The healthsync SQLite schema: table definitions for every parsed Apple Health metric, where the database lives, and example queries you can run with sqlite3."
date: 2026-02-12T00:00:00+05:30
lastmod: 2026-02-28T00:00:00+05:30
draft: false
weight: 400
toc: true
---

The database is stored at `~/.healthsync/healthsync.db` by default. You can query it directly with `sqlite3` for advanced analysis.

## Tables

### Cardiac / Vitals

### heart_rate

```sql
CREATE TABLE heart_rate (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date TEXT NOT NULL,
    end_date TEXT NOT NULL,
    value REAL NOT NULL,          -- BPM
    unit TEXT NOT NULL,           -- "count/min"
    created_at TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
```

### steps

```sql
CREATE TABLE steps (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date TEXT NOT NULL,
    end_date TEXT NOT NULL,
    value REAL NOT NULL,          -- count
    unit TEXT NOT NULL DEFAULT 'count',
    created_at TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
```

### spo2

```sql
CREATE TABLE spo2 (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date TEXT NOT NULL,
    end_date TEXT NOT NULL,
    value REAL NOT NULL,          -- fraction (0.98 = 98%)
    unit TEXT NOT NULL DEFAULT '%',
    created_at TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
```

> **Note:** SpO2 values are stored as fractions (0.0–1.0), not percentages. 0.98 means 98%.

### vo2_max

```sql
CREATE TABLE vo2_max (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date TEXT NOT NULL,
    end_date TEXT NOT NULL,
    value REAL NOT NULL,          -- mL/min·kg
    unit TEXT NOT NULL,
    created_at TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
```

### sleep

```sql
CREATE TABLE sleep (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date TEXT NOT NULL,
    end_date TEXT NOT NULL,
    value TEXT NOT NULL,           -- sleep stage
    created_at TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
```

> **Note:** The sleep table has no `unit` column — it stores category values, not quantities. Supports `--total` for nightly sleep duration totals (hours).

**Sleep stage values:**

| Value | Meaning |
|-------|---------|
| `HKCategoryValueSleepAnalysisInBed` | In bed |
| `HKCategoryValueSleepAnalysisAsleepCore` | Core sleep |
| `HKCategoryValueSleepAnalysisAsleepDeep` | Deep sleep |
| `HKCategoryValueSleepAnalysisAsleepREM` | REM sleep |
| `HKCategoryValueSleepAnalysisAwake` | Awake |
| `HKCategoryValueSleepAnalysisAsleepUnspecified` | Unspecified |

### resting_heart_rate

```sql
CREATE TABLE resting_heart_rate (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date TEXT NOT NULL,
    end_date TEXT NOT NULL,
    value REAL NOT NULL,          -- count/min (BPM)
    unit TEXT NOT NULL,
    created_at TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
```

### hrv

```sql
CREATE TABLE hrv (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date TEXT NOT NULL,
    end_date TEXT NOT NULL,
    value REAL NOT NULL,          -- ms (SDNN)
    unit TEXT NOT NULL,
    created_at TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
```

### heart_rate_recovery

```sql
CREATE TABLE heart_rate_recovery (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date TEXT NOT NULL,
    end_date TEXT NOT NULL,
    value REAL NOT NULL,          -- count/min
    unit TEXT NOT NULL,
    created_at TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
```

### respiratory_rate

```sql
CREATE TABLE respiratory_rate (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date TEXT NOT NULL,
    end_date TEXT NOT NULL,
    value REAL NOT NULL,          -- breaths/min
    unit TEXT NOT NULL,
    created_at TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
```

### blood_pressure

Blood pressure is stored as a paired reading — systolic and diastolic are matched by `source_name + start_date` during parsing and written as a single row.

```sql
CREATE TABLE blood_pressure (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date TEXT NOT NULL,
    end_date TEXT NOT NULL,
    systolic REAL NOT NULL,       -- mmHg
    diastolic REAL NOT NULL,      -- mmHg
    unit TEXT NOT NULL,           -- "mmHg"
    created_at TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, systolic, diastolic)
);
```

### Activity / Energy

The following tables share the standard schema (`source_name, start_date, end_date, value REAL, unit TEXT`):

- `active_energy` — active calories burned (kcal). Supports `--total` for deduplicated daily totals.
- `basal_energy` — resting/basal calories burned (kcal). Supports `--total`.
- `exercise_time` — exercise minutes
- `stand_time` — stand minutes
- `flights_climbed` — flights of stairs
- `distance_walking_running` — walk/run distance
- `distance_cycling` — cycling distance

```sql
CREATE TABLE active_energy (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date TEXT NOT NULL,
    end_date TEXT NOT NULL,
    value REAL NOT NULL,          -- kcal
    unit TEXT NOT NULL,
    created_at TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
```

### Body

The following tables share the standard schema:

- `body_mass` — body weight (kg/lb)
- `body_mass_index` — BMI
- `height` — height (m/ft)

### Mobility / Walking

The following tables share the standard schema:

- `walking_speed` — m/s
- `walking_step_length` — m
- `walking_asymmetry` — %
- `walking_double_support` — %
- `walking_steadiness` — score
- `stair_ascent_speed` — ft/s
- `stair_descent_speed` — ft/s
- `six_minute_walk` — m

### Running metrics

The following tables share the standard schema:

- `running_speed` — m/s
- `running_power` — W
- `running_stride_length` — m
- `running_ground_contact_time` — ms
- `running_vertical_oscillation` — cm

### Other quantity types

The following tables share the standard schema:

- `wrist_temperature` — °C deviation from baseline
- `time_in_daylight` — minutes
- `dietary_water` — mL/L
- `physical_effort` — MET score
- `walking_heart_rate` — BPM average while walking

### mindful_sessions

```sql
CREATE TABLE mindful_sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date TEXT NOT NULL,
    end_date TEXT NOT NULL,
    value TEXT NOT NULL,           -- category value
    created_at TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
```

> **Note:** No `unit` column — this is a category type. Duration can be derived from `end_date - start_date`.

### stand_hours

```sql
CREATE TABLE stand_hours (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date TEXT NOT NULL,
    end_date TEXT NOT NULL,
    value TEXT NOT NULL,           -- HKCategoryValueAppleStandHourStood or Idle
    created_at TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
```

> **Note:** No `unit` column — this is a category type.

### workouts

```sql
CREATE TABLE workouts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    activity_type TEXT NOT NULL,
    source_name TEXT NOT NULL,
    start_date TEXT NOT NULL,
    end_date TEXT NOT NULL,
    duration REAL,                         -- minutes (nullable)
    duration_unit TEXT,
    total_distance REAL,                   -- nullable
    total_distance_unit TEXT,
    total_energy_burned REAL,              -- nullable
    total_energy_burned_unit TEXT,
    created_at TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(activity_type, start_date, end_date, source_name)
);
```

> **Note:** Distance and energy fields are nullable — not all workout types track them (e.g. Yoga has no distance).

Common `activity_type` values: `HKWorkoutActivityTypeRunning`, `HKWorkoutActivityTypeWalking`, `HKWorkoutActivityTypeCycling`, `HKWorkoutActivityTypeYoga`, `HKWorkoutActivityTypeSwimming`, `HKWorkoutActivityTypeHighIntensityIntervalTraining`, `HKWorkoutActivityTypeTraditionalStrengthTraining`.

## Date format

All dates are stored as text in local time (timezone offset stripped during parsing):

```
2024-01-15 08:30:00
```

This format is compatible with SQLite's `date()`, `julianday()`, and other date functions. When filtering with `--from` / `--to`, use date prefixes — SQLite does string comparison, so `2024-01-01` matches all timestamps on that day.

> **Note:** Databases parsed before v0.5.1 store timestamps with the timezone offset appended (e.g. `2024-01-15 08:30:00 +0530`), which causes `julianday()` and `date()` to return NULL. Re-run `healthsync parse` on your export to get normalized timestamps.

## Useful queries

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
SELECT date(start_date) as day,
  ROUND(AVG(value), 1) as avg_hr,
  MIN(value) as min_hr,
  MAX(value) as max_hr
FROM heart_rate
GROUP BY day
ORDER BY day DESC
LIMIT 30;
```

### Sleep duration per night

Use `healthsync query sleep --total` for nightly sleep totals via the CLI, or query directly:

```sql
SELECT date(start_date, '-6 hours') as night,
  ROUND(SUM((julianday(end_date) - julianday(start_date)) * 24), 1) as hours
FROM sleep
WHERE value LIKE '%Asleep%'
GROUP BY night
ORDER BY night DESC
LIMIT 14;
```

The `-6 hours` shift ensures that pre-midnight sleep sessions (e.g. starting at 23:00) are grouped with the same night as post-midnight sessions.

### Workout summary by type

```sql
SELECT activity_type,
  COUNT(*) as count,
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

## Deduplication

All tables use `UNIQUE` constraints and `INSERT OR IGNORE` for idempotent imports. Re-running `healthsync parse` on the same export inserts 0 new rows.

## Indexes

All tables have an index on `start_date` for efficient date-range queries.
