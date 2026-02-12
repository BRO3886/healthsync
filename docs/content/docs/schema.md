---
title: "Database Schema"
description: "SQLite table definitions and query examples."
summary: ""
date: 2026-02-12T00:00:00+05:30
lastmod: 2026-02-12T00:00:00+05:30
draft: false
weight: 400
toc: true
seo:
  title: "Database Schema — healthsync"
  description: "SQLite schema reference with table definitions and useful query examples."
  canonical: ""
  noindex: false
---

The database is stored at `~/.healthsync/healthsync.db` by default. You can query it directly with `sqlite3` for advanced analysis.

## Tables

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

> **Note:** The sleep table has no `unit` column — it stores category values, not quantities.

**Sleep stage values:**

| Value | Meaning |
|-------|---------|
| `HKCategoryValueSleepAnalysisInBed` | In bed |
| `HKCategoryValueSleepAnalysisAsleepCore` | Core sleep |
| `HKCategoryValueSleepAnalysisAsleepDeep` | Deep sleep |
| `HKCategoryValueSleepAnalysisAsleepREM` | REM sleep |
| `HKCategoryValueSleepAnalysisAwake` | Awake |
| `HKCategoryValueSleepAnalysisAsleepUnspecified` | Unspecified |

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

All dates are stored as text in Apple Health format:

```
2024-01-15 08:30:00 +0530
```

When filtering with `--from` / `--to`, use date prefixes — SQLite does string comparison, so `2024-01-01` matches all timestamps on that day.

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

```sql
SELECT date(start_date) as night,
  ROUND(SUM((julianday(end_date) - julianday(start_date)) * 24), 1) as hours
FROM sleep
WHERE value LIKE '%Asleep%'
GROUP BY night
ORDER BY night DESC
LIMIT 14;
```

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
