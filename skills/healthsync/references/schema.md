# SQLite Schema Reference

## heart_rate

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
CREATE INDEX idx_heart_rate_start_date ON heart_rate(start_date);
```

## steps

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
CREATE INDEX idx_steps_start_date ON steps(start_date);
```

## spo2

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
CREATE INDEX idx_spo2_start_date ON spo2(start_date);
```

## vo2_max

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
CREATE INDEX idx_vo2_max_start_date ON vo2_max(start_date);
```

## sleep

```sql
CREATE TABLE sleep (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date TEXT NOT NULL,
    end_date TEXT NOT NULL,
    value TEXT NOT NULL,           -- sleep stage (HKCategoryValueSleepAnalysis*)
    created_at TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
CREATE INDEX idx_sleep_start_date ON sleep(start_date);
```

**Note:** Sleep table has no `unit` column — it stores category values, not quantities.

## workouts

```sql
CREATE TABLE workouts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    activity_type TEXT NOT NULL,           -- HKWorkoutActivityType*
    source_name TEXT NOT NULL,
    start_date TEXT NOT NULL,
    end_date TEXT NOT NULL,
    duration REAL,                         -- nullable, in minutes
    duration_unit TEXT,                    -- "min"
    total_distance REAL,                   -- nullable
    total_distance_unit TEXT,              -- "km", "mi", etc.
    total_energy_burned REAL,              -- nullable
    total_energy_burned_unit TEXT,         -- "kcal"
    created_at TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(activity_type, start_date, end_date, source_name)
);
CREATE INDEX idx_workouts_start_date ON workouts(start_date);
```

**Note:** distance, energy fields are nullable — not all workout types track them (e.g. Yoga has no distance).
